package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const shardDiscoveryRetryInterval = 200 * time.Millisecond

type shardIgnorePathResponse struct {
	DiscoveryID string `json:"discovery_id"`
	ProjectDir  string `json:"project_dir"`
	TaskName    string `json:"task_name"`
	WorkerCount int    `json:"worker_count"`
	IgnorePath  string `json:"ignore_path"`
}

type shardDiscoveryServer struct {
	server *http.Server
	ln     net.Listener
}

func defaultSharedIgnorePath(taskDir string, workerCount int) string {
	return filepath.Join(taskDir, "ignored.shared.log")
}

func shardDiscoveryAddress(projectDir, taskName string, workerCount int) string {
	discoveryID := shardDiscoveryID(projectDir)
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s", discoveryID, taskName)))
	// Keep the deterministic port in the ephemeral/private range.
	port := 49152 + int(binary.LittleEndian.Uint16(sum[:2])%16384)
	return fmt.Sprintf("127.0.0.1:%d", port)
}

func shardDiscoveryID(projectDir string) string {
	cmd := exec.Command("git", "-C", projectDir, "rev-parse", "--git-common-dir")
	output, err := cmd.Output()
	if err != nil {
		return projectDir
	}

	gitDir := strings.TrimSpace(string(output))
	if gitDir == "" {
		return projectDir
	}
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(projectDir, gitDir)
	}
	if abs, err := filepath.Abs(gitDir); err == nil {
		return abs
	}
	return gitDir
}

func startShardDiscoveryServer(projectDir, taskName string, workerCount int, ignorePath string) (*shardDiscoveryServer, error) {
	addr := shardDiscoveryAddress(projectDir, taskName, workerCount)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to start shard discovery server on %s: %w", addr, err)
	}

	response := shardIgnorePathResponse{
		DiscoveryID: shardDiscoveryID(projectDir),
		ProjectDir:  projectDir,
		TaskName:    taskName,
		WorkerCount: workerCount,
		IgnorePath:  ignorePath,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ignore-path", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})

	server := &http.Server{Handler: mux}
	discovery := &shardDiscoveryServer{server: server, ln: ln}
	go func() {
		if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("%s\n", ColorWarning(fmt.Sprintf("Shard discovery server stopped: %v", err)))
		}
	}()

	return discovery, nil
}

func startOrQueryShardDiscovery(projectDir, taskName string, workerCount int, ignorePath string, timeout time.Duration) (*shardDiscoveryServer, string, error) {
	server, err := startShardDiscoveryServer(projectDir, taskName, workerCount, ignorePath)
	if err == nil {
		return server, ignorePath, nil
	}

	existingPath, queryErr := queryShardIgnorePath(projectDir, taskName, workerCount, timeout)
	if queryErr != nil {
		return nil, "", err
	}
	return nil, existingPath, nil
}

func (s *shardDiscoveryServer) Close() error {
	if s == nil || s.server == nil {
		return nil
	}
	return s.server.Close()
}

func queryShardIgnorePath(projectDir, taskName string, workerCount int, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for {
		path, err := queryShardIgnorePathOnce(projectDir, taskName, workerCount)
		if err == nil {
			return path, nil
		}
		lastErr = err

		if time.Now().Add(shardDiscoveryRetryInterval).After(deadline) {
			break
		}
		time.Sleep(shardDiscoveryRetryInterval)
	}

	return "", fmt.Errorf("failed to discover shard 1 ignore path for %s; start shard 1 first: %w", taskName, lastErr)
}

func queryShardIgnorePathOnce(projectDir, taskName string, workerCount int) (string, error) {
	addr := shardDiscoveryAddress(projectDir, taskName, workerCount)
	client := http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get("http://" + addr + "/ignore-path")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected discovery status %s", resp.Status)
	}

	var got shardIgnorePathResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		return "", fmt.Errorf("failed to decode discovery response: %w", err)
	}

	if got.DiscoveryID != shardDiscoveryID(projectDir) || got.TaskName != taskName {
		return "", fmt.Errorf("shard discovery identity mismatch: got discovery_id=%q project=%q task=%q workers=%d", got.DiscoveryID, got.ProjectDir, got.TaskName, got.WorkerCount)
	}
	if got.IgnorePath == "" {
		return "", fmt.Errorf("shard discovery returned empty ignore path")
	}

	return got.IgnorePath, nil
}
