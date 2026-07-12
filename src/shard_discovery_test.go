package main

import (
	"encoding/json"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"
)

func TestShardDiscoveryAddress(t *testing.T) {
	addr1 := shardDiscoveryAddress("/project", "task", 4)
	addr2 := shardDiscoveryAddress("/project", "task", 4)
	if addr1 != addr2 {
		t.Fatalf("address is not stable: %q != %q", addr1, addr2)
	}

	if got := shardDiscoveryAddress("/project", "task", 8); got == addr1 {
		t.Fatalf("address should change when worker count changes")
	}
	if got := shardDiscoveryAddress("/project", "other", 4); got == addr1 {
		t.Fatalf("address should change when task changes")
	}
}

func TestDefaultSharedIgnorePathDoesNotDependOnShardTotal(t *testing.T) {
	taskDir := filepath.Join(t.TempDir(), "task")
	path4 := defaultSharedIgnorePath(taskDir, 4)
	path8 := defaultSharedIgnorePath(taskDir, 8)

	if path4 != path8 {
		t.Fatalf("shared ignore path should survive shard-count changes: %q != %q", path4, path8)
	}
	if path4 != filepath.Join(taskDir, "ignored.shared.log") {
		t.Fatalf("shared ignore path = %q, want ignored.shared.log in task dir", path4)
	}
}

func TestShardDiscoveryServerPublishesIgnorePath(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "project")
	taskName := "test-task"
	workerCount := 99
	ignorePath := filepath.Join(projectDir, "shared.log")

	server, err := startShardDiscoveryServer(projectDir, taskName, workerCount, ignorePath)
	if err != nil {
		t.Fatalf("startShardDiscoveryServer failed: %v", err)
	}
	defer server.Close()

	got, err := queryShardIgnorePath(projectDir, taskName, workerCount, time.Second)
	if err != nil {
		t.Fatalf("queryShardIgnorePath failed: %v", err)
	}
	if got != ignorePath {
		t.Fatalf("ignore path = %q, want %q", got, ignorePath)
	}
}

func TestShardDiscoveryQueryValueCanBeRememberedAfterServerStops(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "project")
	taskName := "test-task"
	workerCount := 98
	ignorePath := filepath.Join(projectDir, "shared.log")

	server, err := startShardDiscoveryServer(projectDir, taskName, workerCount, ignorePath)
	if err != nil {
		t.Fatalf("startShardDiscoveryServer failed: %v", err)
	}

	got, err := queryShardIgnorePath(projectDir, taskName, workerCount, time.Second)
	if err != nil {
		t.Fatalf("queryShardIgnorePath failed: %v", err)
	}
	if err := server.Close(); err != nil {
		t.Fatalf("failed to close server: %v", err)
	}

	if got != ignorePath {
		t.Fatalf("remembered ignore path = %q, want %q", got, ignorePath)
	}
}

func TestShardDiscoveryRejectsIdentityMismatch(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "project")
	taskName := "test-task"
	workerCount := 97

	ln, err := net.Listen("tcp", shardDiscoveryAddress(projectDir, taskName, workerCount))
	if err != nil {
		t.Fatalf("failed to listen on discovery address: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ignore-path", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(shardIgnorePathResponse{
			ProjectDir:  projectDir,
			TaskName:    "other-task",
			WorkerCount: workerCount,
			IgnorePath:  filepath.Join(projectDir, "shared.log"),
		})
	})
	server := &http.Server{Handler: mux}
	go server.Serve(ln)
	defer server.Close()

	if _, err := queryShardIgnorePathOnce(projectDir, taskName, workerCount); err == nil {
		t.Fatal("expected identity mismatch")
	}
}
