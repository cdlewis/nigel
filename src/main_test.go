package main

import (
	"reflect"
	"testing"
)

func TestReorderArgsBooleanOffPeakFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "off peak before task",
			args: []string{"--off-peak-only", "mytask"},
			want: []string{"--off-peak-only", "mytask"},
		},
		{
			name: "china off peak before task",
			args: []string{"--china-off-peak-only", "mytask"},
			want: []string{"--china-off-peak-only", "mytask"},
		},
		{
			name: "off peak after task",
			args: []string{"mytask", "--off-peak-only", "--limit", "3"},
			want: []string{"--off-peak-only", "--limit", "3", "mytask"},
		},
		{
			name: "china off peak after task",
			args: []string{"mytask", "--china-off-peak-only", "--limit", "3"},
			want: []string{"--china-off-peak-only", "--limit", "3", "mytask"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reorderArgs(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("reorderArgs(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
