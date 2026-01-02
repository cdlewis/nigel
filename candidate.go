package main

import (
	"bufio"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
)

// Candidate represents a work item from the candidate source output.
// It can be a single string or an array of strings (first element is the key).
type Candidate struct {
	Key      string
	Elements []string
}

type HashFilter int

const (
	HashFilterNone HashFilter = iota
	HashFilterEvens
	HashFilterOdds
)

// ParseCandidates parses the JSON output from a candidate source.
// Supports both ["a", "b"] and [["a", "related"], ["b", "related"]] formats.
func ParseCandidates(jsonData []byte) ([]Candidate, error) {
	var raw []json.RawMessage
	if err := json.Unmarshal(jsonData, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse JSON array: %w", err)
	}

	candidates := make([]Candidate, 0, len(raw))
	for _, item := range raw {
		// Try parsing as string first
		var str string
		if err := json.Unmarshal(item, &str); err == nil {
			candidates = append(candidates, Candidate{
				Key:      str,
				Elements: []string{str},
			})
			continue
		}

		// Try parsing as array of strings
		var arr []string
		if err := json.Unmarshal(item, &arr); err == nil {
			if len(arr) == 0 {
				continue
			}
			candidates = append(candidates, Candidate{
				Key:      arr[0],
				Elements: arr,
			})
			continue
		}

		return nil, fmt.Errorf("candidate must be string or array of strings")
	}

	return candidates, nil
}

// FilterByHash filters candidates by MD5 hash parity.
func FilterByHash(candidates []Candidate, filter HashFilter) []Candidate {
	if filter == HashFilterNone {
		return candidates
	}

	filtered := make([]Candidate, 0)
	for _, c := range candidates {
		hash := md5.Sum([]byte(c.Key))
		hashInt := new(big.Int).SetBytes(hash[:])
		isEven := hashInt.Bit(0) == 0

		if (filter == HashFilterEvens && isEven) || (filter == HashFilterOdds && !isEven) {
			filtered = append(filtered, c)
		}
	}

	return filtered
}

// IgnoredList manages the list of already-processed candidates.
type IgnoredList struct {
	path    string
	entries map[string]bool
}

func NewIgnoredList(taskDir string) (*IgnoredList, error) {
	path := filepath.Join(taskDir, "ignored.log")
	entries := make(map[string]bool)

	file, err := os.Open(path)
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				entries[line] = true
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("failed to read ignored list: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to open ignored list: %w", err)
	}

	return &IgnoredList{
		path:    path,
		entries: entries,
	}, nil
}

func (l *IgnoredList) Contains(key string) bool {
	return l.entries[key]
}

func (l *IgnoredList) Add(key string) error {
	if l.entries[key] {
		return nil
	}

	file, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open ignored list for writing: %w", err)
	}
	defer file.Close()

	if _, err := fmt.Fprintln(file, key); err != nil {
		return fmt.Errorf("failed to write to ignored list: %w", err)
	}

	l.entries[key] = true
	return nil
}

// SelectCandidate returns the first candidate not in the ignored list.
func SelectCandidate(candidates []Candidate, ignored *IgnoredList) *Candidate {
	for _, c := range candidates {
		if !ignored.Contains(c.Key) {
			return &c
		}
	}
	return nil
}
