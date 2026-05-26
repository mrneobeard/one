//go:build gh_integration

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGitRemotesAddCreateGitHubRepo(t *testing.T) {
	if os.Getenv("ONE_GH_INTEGRATION") != "1" {
		t.Skip("set ONE_GH_INTEGRATION=1 to run GitHub integration tests")
	}

	if _, err := exec.LookPath("gh"); err != nil {
		t.Skip("gh not found in PATH")
	}

	owner := strings.TrimSpace(os.Getenv("ONE_GH_TEST_OWNER"))
	if owner == "" {
		var err error
		owner, err = execOutput("gh", "api", "user", "--jq", ".login")
		if err != nil {
			t.Fatalf("resolve github owner: %v", err)
		}
		owner = strings.TrimSpace(owner)
	}

	repoName := fmt.Sprintf("one-gh-create-test-%d", time.Now().UnixNano())
	fullRepo := owner + "/" + repoName
	repoURL := "git@github.com:" + fullRepo + ".git"

	t.Cleanup(func() {
		_, _ = execOutput("gh", "repo", "delete", fullRepo, "--yes")
	})

	workspaceRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve workspace root: %v", err)
	}

	repoRoot := filepath.Join(workspaceRoot, ".tmp", "one-gh-tests", strings.ReplaceAll(t.Name(), "/", "-"))
	if err := os.RemoveAll(repoRoot); err != nil {
		t.Fatalf("clear repo root: %v", err)
	}
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}

	if _, err := execOutput("git", "-C", repoRoot, "init"); err != nil {
		t.Fatalf("init git repo: %v", err)
	}

	t.Chdir(repoRoot)

	oldOverride := os.Getenv("ONE_TEST_REPO_ROOT")
	_ = os.Unsetenv("ONE_TEST_REPO_ROOT")
	t.Cleanup(func() {
		if oldOverride == "" {
			_ = os.Unsetenv("ONE_TEST_REPO_ROOT")
			return
		}
		_ = os.Setenv("ONE_TEST_REPO_ROOT", oldOverride)
	})

	remotesSyncProjectRef = ""
	remotesSyncCreate = false
	remotesAddCreate = false
	syncProjectRef = ""
	useProjectRef = ""

	rootCmd.SetArgs([]string{"git", "remotes", "add", "ghcreate", repoURL, "--create"})
	if _, err := rootCmd.ExecuteC(); err != nil {
		t.Fatalf("execute git remotes add --create: %v", err)
	}

	if err := ghRepoView(owner, repoName); err != nil {
		t.Fatalf("expected github repo %s to exist: %v", fullRepo, err)
	}
}
