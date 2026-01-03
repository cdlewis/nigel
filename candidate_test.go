package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCandidates(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    []Candidate
		expectError bool
	}{
		{
			name:  "simple string array",
			input: `["file1.go", "file2.go", "file3.go"]`,
			expected: []Candidate{
				{Key: "file1.go", Elements: []string{"file1.go"}},
				{Key: "file2.go", Elements: []string{"file2.go"}},
				{Key: "file3.go", Elements: []string{"file3.go"}},
			},
		},
		{
			name:  "array of arrays",
			input: `[["file1.go", "line 10"], ["file2.go", "line 20"]]`,
			expected: []Candidate{
				{Key: "file1.go", Elements: []string{"file1.go", "line 10"}},
				{Key: "file2.go", Elements: []string{"file2.go", "line 20"}},
			},
		},
		{
			name:  "mixed format",
			input: `["simple.go", ["complex.go", "extra", "data"]]`,
			expected: []Candidate{
				{Key: "simple.go", Elements: []string{"simple.go"}},
				{Key: "complex.go", Elements: []string{"complex.go", "extra", "data"}},
			},
		},
		{
			name:     "empty array",
			input:    `[]`,
			expected: []Candidate{},
		},
		{
			name:        "invalid JSON",
			input:       `not json`,
			expectError: true,
		},
		{
			name:        "invalid element type",
			input:       `[123, 456]`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseCandidates([]byte(tt.input))
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != len(tt.expected) {
				t.Fatalf("got %d candidates, want %d", len(result), len(tt.expected))
			}
			for i, c := range result {
				if c.Key != tt.expected[i].Key {
					t.Errorf("candidate[%d].Key = %q, want %q", i, c.Key, tt.expected[i].Key)
				}
				if len(c.Elements) != len(tt.expected[i].Elements) {
					t.Errorf("candidate[%d].Elements length = %d, want %d", i, len(c.Elements), len(tt.expected[i].Elements))
				}
				for j, elem := range c.Elements {
					if elem != tt.expected[i].Elements[j] {
						t.Errorf("candidate[%d].Elements[%d] = %q, want %q", i, j, elem, tt.expected[i].Elements[j])
					}
				}
			}
		})
	}
}

func TestFilterByHash(t *testing.T) {
	candidates := []Candidate{
		{Key: "a"},
		{Key: "b"},
		{Key: "c"},
		{Key: "d"},
	}

	t.Run("no filter returns all", func(t *testing.T) {
		result := FilterByHash(candidates, HashFilterNone)
		if len(result) != len(candidates) {
			t.Errorf("got %d candidates, want %d", len(result), len(candidates))
		}
	})

	t.Run("evens and odds are disjoint and complete", func(t *testing.T) {
		evens := FilterByHash(candidates, HashFilterEvens)
		odds := FilterByHash(candidates, HashFilterOdds)

		// Together they should cover all candidates
		if len(evens)+len(odds) != len(candidates) {
			t.Errorf("evens (%d) + odds (%d) != total (%d)", len(evens), len(odds), len(candidates))
		}

		// They should be disjoint
		evenKeys := make(map[string]bool)
		for _, c := range evens {
			evenKeys[c.Key] = true
		}
		for _, c := range odds {
			if evenKeys[c.Key] {
				t.Errorf("key %q appears in both evens and odds", c.Key)
			}
		}
	})
}

func TestIgnoredList(t *testing.T) {
	t.Run("contains works correctly", func(t *testing.T) {
		dir := t.TempDir()
		ignoredPath := filepath.Join(dir, "ignored.log")

		// Create ignored.log with some entries
		err := os.WriteFile(ignoredPath, []byte("file1.go\nfile2.go\n"), 0644)
		if err != nil {
			t.Fatalf("failed to create ignored.log: %v", err)
		}

		list, err := NewIgnoredList(dir)
		if err != nil {
			t.Fatalf("NewIgnoredList failed: %v", err)
		}

		if !list.Contains("file1.go") {
			t.Error("expected file1.go to be ignored")
		}
		if !list.Contains("file2.go") {
			t.Error("expected file2.go to be ignored")
		}
		if list.Contains("file3.go") {
			t.Error("expected file3.go to not be ignored")
		}
	})

	t.Run("add appends to file", func(t *testing.T) {
		dir := t.TempDir()

		list, err := NewIgnoredList(dir)
		if err != nil {
			t.Fatalf("NewIgnoredList failed: %v", err)
		}

		if err := list.Add("newfile.go"); err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		if !list.Contains("newfile.go") {
			t.Error("expected newfile.go to be ignored after adding")
		}

		// Verify file was written
		content, err := os.ReadFile(filepath.Join(dir, "ignored.log"))
		if err != nil {
			t.Fatalf("failed to read ignored.log: %v", err)
		}
		if string(content) != "newfile.go\n" {
			t.Errorf("file content = %q, want %q", string(content), "newfile.go\n")
		}
	})

	t.Run("empty directory creates new list", func(t *testing.T) {
		dir := t.TempDir()

		list, err := NewIgnoredList(dir)
		if err != nil {
			t.Fatalf("NewIgnoredList failed: %v", err)
		}

		if list.Contains("anything") {
			t.Error("new list should not contain any entries")
		}
	})
}

func TestSelectCandidate(t *testing.T) {
	t.Run("selects first non-ignored candidate", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "ignored.log"), []byte("file1.go\nfile2.go\n"), 0644)
		if err != nil {
			t.Fatalf("failed to create ignored.log: %v", err)
		}

		list, _ := NewIgnoredList(dir)
		candidates := []Candidate{
			{Key: "file1.go"},
			{Key: "file2.go"},
			{Key: "file3.go"},
			{Key: "file4.go"},
		}

		result := SelectCandidate(candidates, list)
		if result == nil {
			t.Fatal("expected a candidate to be selected")
		}
		if result.Key != "file3.go" {
			t.Errorf("selected %q, want %q", result.Key, "file3.go")
		}
	})

	t.Run("returns nil when all ignored", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "ignored.log"), []byte("file1.go\nfile2.go\n"), 0644)
		if err != nil {
			t.Fatalf("failed to create ignored.log: %v", err)
		}

		list, _ := NewIgnoredList(dir)
		candidates := []Candidate{
			{Key: "file1.go"},
			{Key: "file2.go"},
		}

		result := SelectCandidate(candidates, list)
		if result != nil {
			t.Errorf("expected nil, got %q", result.Key)
		}
	})

	t.Run("counts ignored candidates correctly", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "ignored.log"), []byte("a\nb\nc\n"), 0644)
		if err != nil {
			t.Fatalf("failed to create ignored.log: %v", err)
		}

		list, _ := NewIgnoredList(dir)
		candidates := []Candidate{
			{Key: "a"},
			{Key: "b"},
			{Key: "c"},
			{Key: "d"},
			{Key: "e"},
		}

		// Count ignored (same logic as runner.go)
		ignoredCount := 0
		for _, c := range candidates {
			if list.Contains(c.Key) {
				ignoredCount++
			}
		}

		if ignoredCount != 3 {
			t.Errorf("ignoredCount = %d, want 3", ignoredCount)
		}
		if len(candidates)-ignoredCount != 2 {
			t.Errorf("available = %d, want 2", len(candidates)-ignoredCount)
		}
	})
}
