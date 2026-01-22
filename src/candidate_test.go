package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestParseCandidates(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedKey []string // Expected Key for each candidate
	}{
		{
			name:        "simple string array",
			input:       `["file1.go", "file2.go", "file3.go"]`,
			expectedKey: []string{"file1.go", "file2.go", "file3.go"},
		},
		{
			name:        "array of arrays",
			input:       `[["file1.go", "line 10"], ["file2.go", "line 20"]]`,
			expectedKey: []string{`["file1.go","line 10"]`, `["file2.go","line 20"]`},
		},
		{
			name:        "array of maps",
			input:       `[{"file": "test.go", "line": 10}, {"file": "other.go"}]`,
			expectedKey: []string{`{"file":"test.go","line":10}`, `{"file":"other.go"}`},
		},
		{
			name:        "mixed strings and arrays",
			input:       `["simple.go", ["complex.go", "extra"]]`,
			expectedKey: []string{"simple.go", `["complex.go","extra"]`},
		},
		{
			name:        "empty array",
			input:       `[]`,
			expectedKey: []string{},
		},
		{
			name:        "newline-separated plain text",
			input:       "file1.go\nfile2.go\nfile3.go\n",
			expectedKey: []string{"file1.go", "file2.go", "file3.go"},
		},
		{
			name:        "newline-separated with blank lines",
			input:       "file1.go\n\nfile2.go\n\n\n",
			expectedKey: []string{"file1.go", "file2.go"},
		},
		{
			name:        "newline-separated with whitespace trimming",
			input:       "  file1.go  \n\tfile2.go\t\n  file3.go  ",
			expectedKey: []string{"file1.go", "file2.go", "file3.go"},
		},
		{
			name:        "single line without newline",
			input:       "single.go",
			expectedKey: []string{"single.go"},
		},
		{
			name:        "newline-separated with special characters",
			input:       `file with "quotes".go
file's with apostrophe.go`,
			expectedKey: []string{`file with "quotes".go`, `file's with apostrophe.go`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseCandidates([]byte(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != len(tt.expectedKey) {
				t.Fatalf("got %d candidates, want %d", len(result), len(tt.expectedKey))
			}
			for i, c := range result {
				if c.Key != tt.expectedKey[i] {
					t.Errorf("candidate[%d].Key = %q, want %q", i, c.Key, tt.expectedKey[i])
				}
			}
		})
	}
}

func TestCandidateAccessors(t *testing.T) {
	t.Run("string candidate", func(t *testing.T) {
		candidates, _ := ParseCandidates([]byte(`["hello"]`))
		c := &candidates[0]

		if !c.IsString() {
			t.Error("expected IsString() to be true")
		}
		if c.IsArray() {
			t.Error("expected IsArray() to be false")
		}
		if c.IsMap() {
			t.Error("expected IsMap() to be false")
		}
		if c.String() != "hello" {
			t.Errorf("String() = %q, want %q", c.String(), "hello")
		}
	})

	t.Run("array candidate - GetIndex", func(t *testing.T) {
		candidates, _ := ParseCandidates([]byte(`[["a", "b", "c"]]`))
		c := &candidates[0]

		if !c.IsArray() {
			t.Error("expected IsArray() to be true")
		}

		val, ok := c.GetIndex(0)
		if !ok || val != "a" {
			t.Errorf("GetIndex(0) = %q, %v; want 'a', true", val, ok)
		}

		val, ok = c.GetIndex(1)
		if !ok || val != "b" {
			t.Errorf("GetIndex(1) = %q, %v; want 'b', true", val, ok)
		}

		val, ok = c.GetIndex(10)
		if ok {
			t.Error("GetIndex(10) should return false for out of bounds")
		}
	})

	t.Run("array candidate - GetSlice", func(t *testing.T) {
		candidates, _ := ParseCandidates([]byte(`[["a", "b", "c", "d"]]`))
		c := &candidates[0]

		val, ok := c.GetSlice(1)
		if !ok || val != `["b","c","d"]` {
			t.Errorf("GetSlice(1) = %q, %v; want '[\"b\",\"c\",\"d\"]', true", val, ok)
		}

		val, ok = c.GetSlice(3)
		if !ok || val != `["d"]` {
			t.Errorf("GetSlice(3) = %q, %v; want '[\"d\"]', true", val, ok)
		}

		val, ok = c.GetSlice(10)
		if !ok || val != "[]" {
			t.Errorf("GetSlice(10) = %q, %v; want '[]', true", val, ok)
		}
	})

	t.Run("map candidate - GetKey", func(t *testing.T) {
		candidates, _ := ParseCandidates([]byte(`[{"file": "test.go", "line": 42}]`))
		c := &candidates[0]

		if !c.IsMap() {
			t.Error("expected IsMap() to be true")
		}

		val, ok := c.GetKey("file")
		if !ok || val != "test.go" {
			t.Errorf("GetKey('file') = %q, %v; want 'test.go', true", val, ok)
		}

		val, ok = c.GetKey("line")
		if !ok || val != "42" {
			t.Errorf("GetKey('line') = %q, %v; want '42', true", val, ok)
		}

		val, ok = c.GetKey("missing")
		if ok {
			t.Error("GetKey('missing') should return false")
		}
	})

	t.Run("String() unwraps single-item arrays", func(t *testing.T) {
		candidates, _ := ParseCandidates([]byte(`[["only_item"]]`))
		c := &candidates[0]

		if c.String() != "only_item" {
			t.Errorf("String() = %q, want %q", c.String(), "only_item")
		}
	})

	t.Run("String() returns JSON for multi-item arrays", func(t *testing.T) {
		candidates, _ := ParseCandidates([]byte(`[["a", "b"]]`))
		c := &candidates[0]

		if c.String() != `["a", "b"]` {
			t.Errorf("String() = %q, want %q", c.String(), `["a", "b"]`)
		}
	})
}

func TestFilterByPartition(t *testing.T) {
	candidates := []Candidate{
		{Key: "a"},
		{Key: "b"},
		{Key: "c"},
		{Key: "d"},
		{Key: "e"},
		{Key: "f"},
		{Key: "g"},
		{Key: "h"},
		{Key: "i"},
		{Key: "j"},
	}

	t.Run("single worker returns all", func(t *testing.T) {
		result := FilterByPartition(candidates, HashPartition{WorkerCount: 1, WorkerIndex: 0})
		if len(result) != len(candidates) {
			t.Errorf("got %d candidates, want %d", len(result), len(candidates))
		}
	})

	t.Run("worker count of 0 or less returns all", func(t *testing.T) {
		result := FilterByPartition(candidates, HashPartition{WorkerCount: 0, WorkerIndex: 0})
		if len(result) != len(candidates) {
			t.Errorf("got %d candidates, want %d", len(result), len(candidates))
		}
	})

	t.Run("2-way partitioning is disjoint and complete", func(t *testing.T) {
		partition0 := FilterByPartition(candidates, HashPartition{WorkerCount: 2, WorkerIndex: 0})
		partition1 := FilterByPartition(candidates, HashPartition{WorkerCount: 2, WorkerIndex: 1})

		// Together they should cover all candidates
		if len(partition0)+len(partition1) != len(candidates) {
			t.Errorf("partition0 (%d) + partition1 (%d) != total (%d)", len(partition0), len(partition1), len(candidates))
		}

		// They should be disjoint
		keys0 := make(map[string]bool)
		for _, c := range partition0 {
			keys0[c.Key] = true
		}
		for _, c := range partition1 {
			if keys0[c.Key] {
				t.Errorf("key %q appears in both partitions", c.Key)
			}
		}
	})

	t.Run("4-way partitioning is disjoint and complete", func(t *testing.T) {
		var partitions [][]Candidate
		for i := 0; i < 4; i++ {
			partitions = append(partitions, FilterByPartition(candidates, HashPartition{WorkerCount: 4, WorkerIndex: i}))
		}

		// Count total candidates across all partitions
		total := 0
		allKeys := make(map[string]int)
		for i, p := range partitions {
			total += len(p)
			for _, c := range p {
				allKeys[c.Key]++
				if allKeys[c.Key] > 1 {
					t.Errorf("key %q appears in multiple partitions (seen %d times, now in partition %d)", c.Key, allKeys[c.Key], i)
				}
			}
		}

		if total != len(candidates) {
			t.Errorf("4-way partitions total %d, want %d", total, len(candidates))
		}
	})

	t.Run("10-way partitioning works correctly", func(t *testing.T) {
		// Create more candidates for 10-way partitioning
		largeCandidates := make([]Candidate, 100)
		for i := 0; i < 100; i++ {
			largeCandidates[i] = Candidate{Key: fmt.Sprintf("candidate-%d", i)}
		}

		var partitions [][]Candidate
		for i := 0; i < 10; i++ {
			partitions = append(partitions, FilterByPartition(largeCandidates, HashPartition{WorkerCount: 10, WorkerIndex: i}))
		}

		// Count total candidates across all partitions
		total := 0
		for _, p := range partitions {
			total += len(p)
		}

		if total != len(largeCandidates) {
			t.Errorf("10-way partitions total %d, want %d", total, len(largeCandidates))
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

	t.Run("SetMaxRepeat marks existing entries as done", func(t *testing.T) {
		dir := t.TempDir()
		// Create ignored.log with some entries
		err := os.WriteFile(filepath.Join(dir, "ignored.log"), []byte("func1\nfunc2\n"), 0644)
		if err != nil {
			t.Fatalf("failed to create ignored.log: %v", err)
		}

		list, err := NewIgnoredList(dir)
		if err != nil {
			t.Fatalf("NewIgnoredList failed: %v", err)
		}

		// Before SetMaxRepeat, Contains works normally (attempts = 1)
		if !list.Contains("func1") {
			t.Error("expected func1 to be ignored before SetMaxRepeat")
		}

		// Set repeat mode with maxRepeat = 3
		list.SetMaxRepeat(3)

		// Existing entries should now be considered "done" (attempts = 3)
		if !list.Contains("func1") {
			t.Error("expected func1 to be ignored after SetMaxRepeat (existing entry)")
		}
		if !list.Contains("func2") {
			t.Error("expected func2 to be ignored after SetMaxRepeat (existing entry)")
		}
		// New entries should not be ignored (attempts = 0)
		if list.Contains("func3") {
			t.Error("expected func3 to not be ignored (new entry)")
		}
	})

	t.Run("SetMaxRepeat allows retrying new candidates up to N times", func(t *testing.T) {
		dir := t.TempDir()

		list, err := NewIgnoredList(dir)
		if err != nil {
			t.Fatalf("NewIgnoredList failed: %v", err)
		}

		list.SetMaxRepeat(3)

		// New candidate should not be ignored initially
		if list.Contains("newFunc") {
			t.Error("new candidate should not be ignored")
		}

		// Simulate attempts
		list.Add("newFunc") // attempts = 1
		if list.Contains("newFunc") {
			t.Error("candidate should not be ignored after 1 attempt (max is 3)")
		}

		list.Add("newFunc") // attempts = 2
		if list.Contains("newFunc") {
			t.Error("candidate should not be ignored after 2 attempts (max is 3)")
		}

		list.Add("newFunc") // attempts = 3
		if !list.Contains("newFunc") {
			t.Error("candidate should be ignored after 3 attempts (reached max)")
		}
	})

	t.Run("SetMaxRepeat with 0 disables repeat mode", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "ignored.log"), []byte("func1\n"), 0644)
		if err != nil {
			t.Fatalf("failed to create ignored.log: %v", err)
		}

		list, err := NewIgnoredList(dir)
		if err != nil {
			t.Fatalf("NewIgnoredList failed: %v", err)
		}

		// Set maxRepeat to 0 (no repeat mode)
		list.SetMaxRepeat(0)

		// Should use entries map, not attempts
		if !list.Contains("func1") {
			t.Error("expected func1 to be ignored in non-repeat mode")
		}
		if list.Contains("func2") {
			t.Error("expected func2 to not be ignored in non-repeat mode")
		}
	})

	t.Run("SetMaxRepeat persists to file when limit reached", func(t *testing.T) {
		dir := t.TempDir()

		list, err := NewIgnoredList(dir)
		if err != nil {
			t.Fatalf("NewIgnoredList failed: %v", err)
		}

		list.SetMaxRepeat(3)

		// First two attempts should not write to file
		list.Add("retryFunc") // attempts = 1
		list.Add("retryFunc") // attempts = 2

		// Verify file doesn't exist yet
		_, err = os.Stat(filepath.Join(dir, "ignored.log"))
		if !os.IsNotExist(err) {
			t.Error("file should not exist before reaching repeat limit")
		}

		// Third attempt should write to file
		list.Add("retryFunc") // attempts = 3

		// Verify file was written
		content, err := os.ReadFile(filepath.Join(dir, "ignored.log"))
		if err != nil {
			t.Fatalf("failed to read ignored.log: %v", err)
		}
		if string(content) != "retryFunc\n" {
			t.Errorf("file content = %q, want %q", string(content), "retryFunc\n")
		}

		// Verify it persists across reloads
		list2, err := NewIgnoredList(dir)
		if err != nil {
			t.Fatalf("NewIgnoredList failed: %v", err)
		}
		list2.SetMaxRepeat(3)

		if !list2.Contains("retryFunc") {
			t.Error("candidate should be ignored after reload when limit was reached")
		}
	})
}

func TestDeterministicMapKeys(t *testing.T) {
	t.Run("map keys are deterministic across parses", func(t *testing.T) {
		// Parse the same JSON multiple times
		jsonInput := `[{"file": "test.go", "line": 10}, {"line": 20, "file": "other.go"}]`

		var keys1 []string
		for i := 0; i < 10; i++ {
			candidates, err := ParseCandidates([]byte(jsonInput))
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}

			var currentKeys []string
			for _, c := range candidates {
				currentKeys = append(currentKeys, c.Key)
			}

			if i == 0 {
				keys1 = currentKeys
			} else {
				// Keys should be identical across all parses
				for j, k := range currentKeys {
					if k != keys1[j] {
						t.Errorf("parse %d: key[%d] = %q, want %q", i, j, k, keys1[j])
					}
				}
			}
		}
	})

	t.Run("map keys are sorted for consistency", func(t *testing.T) {
		// Regardless of input order, output should be sorted
		input1 := `[{"file": "test.go", "line": 10}]`
		input2 := `[{"line": 10, "file": "test.go"}]`

		candidates1, _ := ParseCandidates([]byte(input1))
		candidates2, _ := ParseCandidates([]byte(input2))

		if len(candidates1) != 1 || len(candidates2) != 1 {
			t.Fatal("expected 1 candidate each")
		}

		if candidates1[0].Key != candidates2[0].Key {
			t.Errorf("keys differ: %q vs %q", candidates1[0].Key, candidates2[0].Key)
		}

		// Verify keys are alphabetically sorted
		expected := `{"file":"test.go","line":10}`
		if candidates1[0].Key != expected {
			t.Errorf("key = %q, want %q (sorted)", candidates1[0].Key, expected)
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
