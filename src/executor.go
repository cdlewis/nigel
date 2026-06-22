package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// timeoutError indicates AI execution timed out
type timeoutError struct {
	duration time.Duration
}

// StreamCallback is called for each chunk of text received from the AI backend.
type StreamCallback func(text string)

func (e *timeoutError) Error() string {
	return fmt.Sprintf("timeout after %s", e.duration)
}

func (e *timeoutError) IsTimeout() bool {
	return true
}

// RunCandidateSource executes a candidate source command and returns its stdout.
func RunCandidateSource(source, workDir string) ([]byte, error) {
	cmd := exec.Command("bash", "-c", source)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("candidate source failed: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// runningProcess tracks the currently running AI process for signal forwarding
var runningProcess *os.Process

// KillRunningProcess terminates the running AI process if any
func KillRunningProcess() {
	if p := runningProcess; p != nil {
		// Kill the entire process group
		syscall.Kill(-p.Pid, syscall.SIGTERM)
	}
}

// RunAICommand executes an AI command with prompt, timeout, and streaming output.
// The streamCb callback is invoked for each chunk of text received.
// Returns the accumulated output (for rate limit detection) and any error.
func RunAICommand(backend Backend, baseCmd, extraFlags, prompt, workDir string, logWriter io.Writer, timeout time.Duration, streamCb StreamCallback) (string, error) {
	// Build the command via the backend
	cmdStr := backend.BuildCommand(baseCmd, extraFlags, prompt)

	// Log the exact command being executed (for debugging hangs)
	if logWriter != nil {
		fmt.Fprintf(logWriter, "Command: %s\n", cmdStr)
	}

	args := []string{"-c", cmdStr}

	cmd := exec.Command("bash", args...)
	cmd.Dir = workDir
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:   true,
		Pdeathsig: syscall.SIGTERM,
	}

	// Create pipe for stdout so we can read line-by-line
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	// Capture stderr to buffer
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	// Start the process and track it for signal forwarding
	if err := cmd.Start(); err != nil {
		return "", err
	}
	runningProcess = cmd.Process

	// Goroutine to read stdout line-by-line and delegate parsing to the backend
	type streamResult struct {
		fullOutput string
		err        error
	}
	resultCh := make(chan streamResult, 1)

	go func() {
		var fullOutput strings.Builder
		scanner := bufio.NewScanner(stdoutPipe)
		scanner.Buffer(nil, 10*1024*1024) // 10MB max token size

		for scanner.Scan() {
			line := scanner.Text()

			text, done := backend.ProcessLine(line)
			if text != "" {
				if streamCb != nil {
					streamCb(text)
				}
				if logWriter != nil {
					fmt.Fprint(logWriter, text)
				}
				fullOutput.WriteString(text)
			}
			if done {
				// Drain remaining output to avoid blocking the process
				for scanner.Scan() {
					if logWriter != nil {
						fmt.Fprintln(logWriter, scanner.Text())
					}
				}
				break
			}

			// Log raw line for debugging if it wasn't consumed as stream text
			if text == "" && logWriter != nil {
				fmt.Fprintln(logWriter, line)
			}
			fullOutput.WriteString(line + "\n")
		}

		// Add a final newline after streaming is complete
		if streamCb != nil {
			streamCb("\n")
		}
		if logWriter != nil {
			fmt.Fprintln(logWriter)
		}

		// Include stderr in output for rate limit detection
		if stderrBuf.Len() > 0 {
			fullOutput.WriteString(stderrBuf.String())
		}

		resultCh <- streamResult{
			fullOutput: fullOutput.String(),
			err:        scanner.Err(),
		}
	}()

	// Wait for completion or timeout
	var waitErr error
	if timeout > 0 {
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-time.After(timeout):
			KillRunningProcess()
			runningProcess = nil
			// Wait for the stream reader to finish
			result := <-resultCh
			return result.fullOutput, &timeoutError{duration: timeout}
		case waitErr = <-done:
			runningProcess = nil
		}
	} else {
		waitErr = cmd.Wait()
		runningProcess = nil
	}

	// Get the full output from the stream reader
	result := <-resultCh
	if result.err != nil {
		return result.fullOutput, result.err
	}

	return result.fullOutput, waitErr
}

// Regex patterns for $INPUT interpolation
var (
	// $INPUT["key"] - map key access
	inputMapKeyRe = regexp.MustCompile(`\$INPUT\["([^"]+)"\]`)
	// $INPUT[n:] - slice from index
	inputSliceRe = regexp.MustCompile(`\$INPUT\[(\d+):\]`)
	// $INPUT[n] - array index access
	inputIndexRe = regexp.MustCompile(`\$INPUT\[(\d+)\]`)
	// $INPUT - bare input (must be checked last)
	inputBareRe = regexp.MustCompile(`\$INPUT\b`)
)

// interpolationError is returned when $INPUT variable type doesn't match the operation.
type interpolationError struct {
	Variable string // The variable that caused the error (e.g., "$INPUT[0]")
	Op       string // The operation being attempted (e.g., "array index")
	Actual   string // The actual type (e.g., "string", "map", "array")
}

func (e *interpolationError) Error() string {
	return fmt.Sprintf("prompt interpolation error: cannot use %s (requires array) on %s candidate", e.Variable, e.Actual)
}

// InterpolatePrompt replaces template variables with candidate values.
// Supports: $INPUT, $INPUT[n], $INPUT[n:], $INPUT["key"], $TASK_ID
// Returns an error if the input type doesn't match the operation (e.g., using array index on a string).
func InterpolatePrompt(template string, candidate *Candidate, taskID int64) (string, error) {
	result := template

	// Replace $TASK_ID - unique task identifier
	result = strings.ReplaceAll(result, "$TASK_ID", fmt.Sprintf("%d", taskID))

	// Replace $INPUT["key"] - map key access
	result = inputMapKeyRe.ReplaceAllStringFunc(result, func(match string) string {
		submatch := inputMapKeyRe.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		key := submatch[1]
		if val, ok := candidate.GetKey(key); ok {
			return val
		}
		return ""
	})

	// Check for type mismatches BEFORE replacement
	// If we find $INPUT[n:] or $INPUT[n] but the candidate is not an array, error

	// Replace $INPUT[n:] - slice from index
	matches := inputSliceRe.FindAllStringSubmatch(result, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			if !candidate.IsArray() {
				// Determine the actual type for the error message
				actualType := "string"
				if candidate.IsMap() {
					actualType = "map"
				}
				return "", &interpolationError{
					Variable: match[0],
					Op:       "slice",
					Actual:   actualType,
				}
			}
		}
	}

	result = inputSliceRe.ReplaceAllStringFunc(result, func(match string) string {
		submatch := inputSliceRe.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		idx, _ := strconv.Atoi(submatch[1])
		if val, ok := candidate.GetSlice(idx); ok {
			return val
		}
		return "[]"
	})

	// Replace $INPUT[n] - array index access
	matches = inputIndexRe.FindAllStringSubmatch(result, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			if !candidate.IsArray() {
				// Determine the actual type for the error message
				actualType := "string"
				if candidate.IsMap() {
					actualType = "map"
				}
				return "", &interpolationError{
					Variable: match[0],
					Op:       "array index",
					Actual:   actualType,
				}
			}
		}
	}

	result = inputIndexRe.ReplaceAllStringFunc(result, func(match string) string {
		submatch := inputIndexRe.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		idx, _ := strconv.Atoi(submatch[1])
		if val, ok := candidate.GetIndex(idx); ok {
			return val
		}
		return ""
	})

	// Replace bare $INPUT - whole value (with single-item unwrap)
	result = inputBareRe.ReplaceAllStringFunc(result, func(match string) string {
		return candidate.String()
	})

	return result, nil
}

// shellQuote wraps a value in single quotes for safe shell interpolation.
// Single quotes within the value are handled by ending the quote, adding an escaped quote, and restarting.
// Example: O'Reilly -> 'O'"'"'Reilly'
func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	// Single quotes make everything literal, except single quotes themselves.
	// To handle single quotes in the value, we exit the single-quote context,
	// add an escaped double-quote, and re-enter single-quote context.
	// 'value' -> 'value'
	// O'Reilly -> 'O'"'"'Reilly'
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

// shouldSkipSuccessCommand determines if a command should be skipped based on
// whether there are changes and what type of command it is.
// Returns true if the command should be skipped.
func shouldSkipSuccessCommand(cmd string, hasChanges bool) bool {
    // Always run if there are changes
    if hasChanges {
        return false
    }

    // When there are no changes, skip git commit operations
    // to avoid "nothing to commit" errors
    lowerCmd := strings.ToLower(cmd)
    return strings.Contains(lowerCmd, "git commit") ||
        strings.Contains(lowerCmd, "git merge") ||
        strings.Contains(lowerCmd, "git rebase")
}

// InterpolateCommand replaces template variables in commands.
// Supports: $CANDIDATE, $TASK_NAME
// $CANDIDATE is shell-quoted to safely handle special characters.
func InterpolateCommand(command string, candidate *Candidate, taskName string) string {
	result := strings.ReplaceAll(command, "$CANDIDATE", shellQuote(candidate.Key))
	result = strings.ReplaceAll(result, "$TASK_NAME", taskName)
	return result
}

// LoadTemplate reads a template file and returns its contents.
func LoadTemplate(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read template: %w", err)
	}
	return string(data), nil
}

// HasUncommittedChanges is now defined in command_executor.go.

// CheckAICommand verifies the AI command is accessible.
func CheckAICommand(cmd string) error {
	// Extract just the command name (first part before any spaces)
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	_, err := exec.LookPath(parts[0])
	if err != nil {
		return fmt.Errorf("command not found: %s", parts[0])
	}
	return nil
}

// parseInt parses a string to int, returning 0 on error.
func parseInt(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
