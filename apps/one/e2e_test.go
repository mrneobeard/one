package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIEndToEndProjectsSyncAndUse(t *testing.T) {
	workspaceRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve workspace root: %v", err)
	}

	testRoot := filepath.Join(workspaceRoot, ".tmp", "one-e2e", strings.ReplaceAll(t.Name(), "/", "-"))
	if err := os.RemoveAll(testRoot); err != nil {
		t.Fatalf("clear e2e root: %v", err)
	}
	if err := os.MkdirAll(testRoot, 0o755); err != nil {
		t.Fatalf("create e2e root: %v", err)
	}

	if err := runCmd(testRoot, "git", "init"); err != nil {
		t.Fatalf("init git repo: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(testRoot, "lib", "js"), 0o755); err != nil {
		t.Fatalf("create project dir: %v", err)
	}

	oneYAML := `name: JavaScript Library
alias: js
desc: JS subtree
git:
  remotes:
    js: git@example.com:org/js
  sync:
    subtree: true
    remotes: [js]
`
	if err := os.WriteFile(filepath.Join(testRoot, "lib", "js", "one.yaml"), []byte(oneYAML), 0o644); err != nil {
		t.Fatalf("write one.yaml: %v", err)
	}

	if _, stderr, err := runOneCmd(testRoot, "projects", "sync"); err != nil {
		t.Fatalf("projects sync failed: %v stderr=%s", err, stderr)
	}

	if _, stderr, err := runOneCmd(testRoot, "projects", "use", "js"); err != nil {
		t.Fatalf("projects use failed: %v stderr=%s", err, stderr)
	}

	idxPath := filepath.Join(testRoot, ".one", "projects.json")
	raw, err := os.ReadFile(idxPath)
	if err != nil {
		t.Fatalf("read projects index: %v", err)
	}

	content := string(raw)
	if !strings.Contains(content, `"default_project": "lib/js"`) {
		t.Fatalf("expected default_project in projects index, got: %s", content)
	}
}

func runOneCmd(repoRoot string, args ...string) (string, string, error) {
	cmdArgs := append([]string{"run", "."}, args...)
	cmd := exec.Command("go", cmdArgs...)
	cmd.Dir = filepath.Clean(".")
	cmd.Env = append(os.Environ(), "ONE_TEST_REPO_ROOT="+repoRoot)
	stdout, stderr, err := runCmdOutput(cmd)
	return stdout, stderr, err
}

func runCmd(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	_, _, err := runCmdOutput(cmd)
	return err
}

func runCmdOutput(cmd *exec.Cmd) (string, string, error) {
	stdout, err := cmd.Output()
	if err == nil {
		return string(stdout), "", nil
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return string(stdout), "", err
	}

	return string(stdout), string(exitErr.Stderr), err
}
