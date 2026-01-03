package main

import "testing"

func TestInterpolatePrompt(t *testing.T) {
	tests := []struct {
		name     string
		template string
		elements []string
		expected string
	}{
		{
			name:     "simple $ARGUMENT",
			template: "Fix this: $ARGUMENT",
			elements: []string{"error.go"},
			expected: "Fix this: error.go",
		},
		{
			name:     "$ARGUMENT_1 should not be corrupted by $ARGUMENT replacement",
			template: "File: $ARGUMENT_1, Line: $ARGUMENT_2",
			elements: []string{"main.go", "42"},
			expected: "File: main.go, Line: 42",
		},
		{
			name:     "mixed $ARGUMENT and $ARGUMENT_1 in same template",
			template: "Primary: $ARGUMENT, First: $ARGUMENT_1, Second: $ARGUMENT_2",
			elements: []string{"foo", "bar"},
			expected: "Primary: foo, First: foo, Second: bar",
		},
		{
			name:     "$REMAINING_ARGUMENTS",
			template: "Main: $ARGUMENT, Others: $REMAINING_ARGUMENTS",
			elements: []string{"first", "second", "third"},
			expected: "Main: first, Others: second, third",
		},
		{
			name:     "empty $REMAINING_ARGUMENTS with single element",
			template: "File: $ARGUMENT, Extras: $REMAINING_ARGUMENTS",
			elements: []string{"only.go"},
			expected: "File: only.go, Extras: ",
		},
		{
			name:     "no elements",
			template: "Nothing: $ARGUMENT",
			elements: []string{},
			expected: "Nothing: $ARGUMENT",
		},
		{
			name:     "multiple numbered arguments",
			template: "$ARGUMENT_1:$ARGUMENT_2:$ARGUMENT_3",
			elements: []string{"a", "b", "c"},
			expected: "a:b:c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidate := &Candidate{
				Key:      "test-key",
				Elements: tt.elements,
			}
			result := InterpolatePrompt(tt.template, candidate)
			if result != tt.expected {
				t.Errorf("InterpolatePrompt() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestInterpolateCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		key      string
		taskName string
		expected string
	}{
		{
			name:     "replace $CANDIDATE",
			command:  "echo $CANDIDATE",
			key:      "file.go:10",
			taskName: "lint",
			expected: "echo file.go:10",
		},
		{
			name:     "replace $TASK_NAME",
			command:  "run-$TASK_NAME.sh",
			key:      "test",
			taskName: "build",
			expected: "run-build.sh",
		},
		{
			name:     "replace both",
			command:  "$TASK_NAME: $CANDIDATE",
			key:      "error",
			taskName: "fix",
			expected: "fix: error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidate := &Candidate{
				Key:      tt.key,
				Elements: []string{},
			}
			result := InterpolateCommand(tt.command, candidate, tt.taskName)
			if result != tt.expected {
				t.Errorf("InterpolateCommand() = %q, want %q", result, tt.expected)
			}
		})
	}
}
