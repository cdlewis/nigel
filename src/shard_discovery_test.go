package main

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
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

	if got := shardDiscoveryAddress("/project", "task", 8); got != addr1 {
		t.Fatalf("address should not change when worker count changes: %q != %q", got, addr1)
	}
	if got := shardDiscoveryAddress("/project", "other", 4); got == addr1 {
		t.Fatalf("address should change when task changes")
	}
}

func TestShardDiscoveryIDUsesGitCommonDir(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	mainDir := filepath.Join(dir, "repo")
	worktreeDir := filepath.Join(dir, "repo-a")
	if err := os.Mkdir(mainDir, 0755); err != nil {
		t.Fatalf("failed to create repo dir: %v", err)
	}
	runGit(t, mainDir, "init")
	runGit(t, mainDir, "config", "user.email", "test@example.com")
	runGit(t, mainDir, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(mainDir, "README.md"), []byte("test\n"), 0644); err != nil {
		t.Fatalf("failed to write README: %v", err)
	}
	runGit(t, mainDir, "add", "README.md")
	runGit(t, mainDir, "commit", "-m", "init")
	runGit(t, mainDir, "worktree", "add", worktreeDir)

	mainID := shardDiscoveryID(mainDir)
	worktreeID := shardDiscoveryID(worktreeDir)
	if mainID != worktreeID {
		t.Fatalf("git worktrees should share discovery ID: %q != %q", mainID, worktreeID)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
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

func TestShardDiscoveryWorksAcrossShardTotals(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "project")
	taskName := "test-task"
	ignorePath := filepath.Join(projectDir, "shared.log")

	server, err := startShardDiscoveryServer(projectDir, taskName, 4, ignorePath)
	if err != nil {
		t.Fatalf("startShardDiscoveryServer failed: %v", err)
	}
	defer server.Close()

	got, err := queryShardIgnorePath(projectDir, taskName, 8, time.Second)
	if err != nil {
		t.Fatalf("queryShardIgnorePath with different shard total failed: %v", err)
	}
	if got != ignorePath {
		t.Fatalf("ignore path = %q, want %q", got, ignorePath)
	}
}

func TestShardDiscoveryShardOneReusesExistingPublisher(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "project")
	taskName := "test-task"
	ignorePath := filepath.Join(projectDir, "shared.log")

	server, err := startShardDiscoveryServer(projectDir, taskName, 4, ignorePath)
	if err != nil {
		t.Fatalf("startShardDiscoveryServer failed: %v", err)
	}
	defer server.Close()

	secondServer, got, err := startOrQueryShardDiscovery(projectDir, taskName, 8, filepath.Join(projectDir, "other.log"), time.Second)
	if err != nil {
		t.Fatalf("startOrQueryShardDiscovery failed: %v", err)
	}
	if secondServer != nil {
		defer secondServer.Close()
		t.Fatal("expected second shard 1 to reuse existing publisher")
	}
	if got != ignorePath {
		t.Fatalf("ignore path = %q, want existing publisher path %q", got, ignorePath)
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
			DiscoveryID: shardDiscoveryID(projectDir),
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
