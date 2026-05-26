package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectsSyncAndUseAndGitCommands(t *testing.T) {
	env := newTestEnv(t)
	env.writeProjectConfig(t, "lib/js", testProjectConfig{
		Name:  "JavaScript Library",
		Alias: "js",
		Desc:  "JS subtree",
		Remotes: map[string]string{
			"js": "git@example.com:org/js",
		},
		SyncSubtree: true,
		SyncRemotes: []string{"js"},
	})

	env.writeProjectConfig(t, "lib/go", testProjectConfig{
		Name:  "Go Library",
		Alias: "go",
		Desc:  "Go subtree",
		Remotes: map[string]string{
			"go": "git@example.com:org/go",
		},
		SyncSubtree: true,
		SyncRemotes: []string{"go"},
	})

	env.execOne(t, "projects", "sync")

	idx := env.readIndex(t)
	if len(idx.Projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(idx.Projects))
	}

	env.execOne(t, "projects", "use", "js")
	idx = env.readIndex(t)
	if idx.DefaultProject != filepath.Clean("lib/js") {
		t.Fatalf("expected default project lib/js, got %q", idx.DefaultProject)
	}

	env.execOne(t, "git", "remotes", "sync", "js")
	log := env.gitLog(t)
	if !containsLine(log, "remote add js git@example.com:org/js") {
		t.Fatalf("expected remote add for js, log:\n%s", strings.Join(log, "\n"))
	}

	env.execOne(t, "git", "remotes", "add", "manual", "git@example.com:org/manual")
	log = env.gitLog(t)
	if !containsLine(log, "remote add manual git@example.com:org/manual") {
		t.Fatalf("expected manual remote add, log:\n%s", strings.Join(log, "\n"))
	}

	output := env.execOneOutput(t, "git", "remotes", "list")
	if !strings.Contains(output, "js git@example.com:org/js") {
		t.Fatalf("expected js in remote list, got: %s", output)
	}
	if !strings.Contains(output, "manual git@example.com:org/manual") {
		t.Fatalf("expected manual in remote list, got: %s", output)
	}

	env.execOne(t, "git", "sync")
	log = env.gitLog(t)
	if !containsLine(log, "push origin") {
		t.Fatalf("expected push origin, log:\n%s", strings.Join(log, "\n"))
	}
	if !containsLine(log, "subtree push --prefix lib/js js main") {
		t.Fatalf("expected subtree push for lib/js, log:\n%s", strings.Join(log, "\n"))
	}
}

func TestProjectsSyncSingleProjectArg(t *testing.T) {
	env := newTestEnv(t)
	env.writeProjectConfig(t, "lib/js", testProjectConfig{Name: "JavaScript", Alias: "js", Desc: "JS"})
	env.writeProjectConfig(t, "lib/go", testProjectConfig{Name: "Go", Alias: "go", Desc: "Go"})

	env.execOne(t, "projects", "sync", "lib/go/one.yaml")

	idx := env.readIndex(t)
	if len(idx.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(idx.Projects))
	}
	if idx.Projects[0].Path != filepath.Clean("lib/go") {
		t.Fatalf("expected project path lib/go, got %q", idx.Projects[0].Path)
	}
}

type testEnv struct {
	repoRoot string
	gitLogF  string
	oldPath  string
	oldEnv   string
}

type testProjectConfig struct {
	Name        string
	Alias       string
	Desc        string
	Remotes     map[string]string
	SyncSubtree bool
	SyncRemotes []string
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	workspaceRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve workspace root: %v", err)
	}

	root := filepath.Join(workspaceRoot, ".tmp", "one-tests", strings.ReplaceAll(t.Name(), "/", "-"))
	if err := os.RemoveAll(root); err != nil {
		t.Fatalf("clear test root: %v", err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("create test root: %v", err)
	}

	gitBinDir := filepath.Join(root, "fake-bin")
	if err := os.MkdirAll(gitBinDir, 0o755); err != nil {
		t.Fatalf("create fake bin dir: %v", err)
	}

	gitLogF := filepath.Join(root, "git.log")
	script := filepath.Join(gitBinDir, "git")
	if err := os.WriteFile(script, []byte(fakeGitScript), 0o755); err != nil {
		t.Fatalf("write fake git script: %v", err)
	}

	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", gitBinDir+string(os.PathListSeparator)+oldPath); err != nil {
		t.Fatalf("set PATH: %v", err)
	}

	oldEnv := os.Getenv("ONE_TEST_REPO_ROOT")
	if err := os.Setenv("ONE_TEST_REPO_ROOT", root); err != nil {
		t.Fatalf("set ONE_TEST_REPO_ROOT: %v", err)
	}
	if err := os.Setenv("ONE_TEST_GIT_LOG", gitLogF); err != nil {
		t.Fatalf("set ONE_TEST_GIT_LOG: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Setenv("PATH", oldPath)
		_ = os.Setenv("ONE_TEST_REPO_ROOT", oldEnv)
		_ = os.Unsetenv("ONE_TEST_GIT_LOG")
	})

	t.Chdir(root)

	return &testEnv{repoRoot: root, gitLogF: gitLogF, oldPath: oldPath, oldEnv: oldEnv}
}

func (e *testEnv) writeProjectConfig(t *testing.T, relDir string, cfg testProjectConfig) {
	t.Helper()

	path := filepath.Join(e.repoRoot, relDir)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("create project dir: %v", err)
	}

	remotes := ""
	for name, url := range cfg.Remotes {
		remotes += "    " + name + ": " + url + "\n"
	}

	syncRemotes := ""
	if len(cfg.SyncRemotes) > 0 {
		syncRemotes = "    remotes: [" + strings.Join(cfg.SyncRemotes, ", ") + "]\n"
	}

	body := "name: " + cfg.Name + "\n" +
		"alias: " + cfg.Alias + "\n" +
		"desc: " + cfg.Desc + "\n" +
		"git:\n"

	if remotes != "" {
		body += "  remotes:\n" + remotes
	}
	body += "  sync:\n"
	if cfg.SyncSubtree {
		body += "    subtree: true\n"
	}
	body += syncRemotes

	if err := os.WriteFile(filepath.Join(path, "one.yaml"), []byte(body), 0o644); err != nil {
		t.Fatalf("write one.yaml: %v", err)
	}
}

func (e *testEnv) execOne(t *testing.T, args ...string) {
	t.Helper()

	remotesSyncProjectRef = ""
	remotesSyncCreate = false
	remotesSyncPublic = false
	remotesAddCreate = false
	remotesAddPublic = false
	syncProjectRef = ""
	useProjectRef = ""

	rootCmd.SetArgs(args)
	_, err := rootCmd.ExecuteC()
	if err != nil {
		t.Fatalf("execute one %v: %v", args, err)
	}
}

func (e *testEnv) execOneOutput(t *testing.T, args ...string) string {
	t.Helper()

	remotesSyncProjectRef = ""
	remotesSyncCreate = false
	remotesSyncPublic = false
	remotesAddCreate = false
	remotesAddPublic = false
	syncProjectRef = ""
	useProjectRef = ""

	original := rootCmd.OutOrStdout()
	defer rootCmd.SetOut(original)

	var out strings.Builder
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs(args)

	_, err := rootCmd.ExecuteC()
	if err != nil {
		t.Fatalf("execute one %v: %v", args, err)
	}

	return out.String()
}

func (e *testEnv) readIndex(t *testing.T) projectsIndex {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(e.repoRoot, ".one", "projects.json"))
	if err != nil {
		t.Fatalf("read projects index: %v", err)
	}

	var idx projectsIndex
	if err := json.Unmarshal(raw, &idx); err != nil {
		t.Fatalf("unmarshal projects index: %v", err)
	}

	return idx
}

func (e *testEnv) gitLog(t *testing.T) []string {
	t.Helper()

	raw, err := os.ReadFile(e.gitLogF)
	if err != nil {
		t.Fatalf("read git log: %v", err)
	}

	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return []string{}
	}

	return strings.Split(trimmed, "\n")
}

func containsLine(lines []string, want string) bool {
	for _, line := range lines {
		if strings.Contains(line, want) {
			return true
		}
	}

	return false
}

const fakeGitScript = `#!/usr/bin/env bash
set -euo pipefail

log_file="${ONE_TEST_GIT_LOG:?missing ONE_TEST_GIT_LOG}"
repo_root="${ONE_TEST_REPO_ROOT:?missing ONE_TEST_REPO_ROOT}"

printf "%s\n" "$*" >> "$log_file"

if [[ "$#" -ge 2 && "$1" == "rev-parse" && "$2" == "--show-toplevel" ]]; then
  printf "%s\n" "$repo_root"
  exit 0
fi

if [[ "$#" -ge 2 && "$1" == "remote" && "$2" == "get-url" ]]; then
  name="$3"
  f="$repo_root/.git-remote-$name"
  if [[ -f "$f" ]]; then
    cat "$f"
    exit 0
  fi
  printf "No such remote '%s'\n" "$name" >&2
  exit 2
fi

if [[ "$1" == "remote" && "$#" -eq 1 ]]; then
  for f in "$repo_root"/.git-remote-*; do
    [[ -e "$f" ]] || continue
    printf "%s\n" "${f##*/.git-remote-}"
  done
  exit 0
fi

if [[ "$#" -ge 2 && "$1" == "remote" && "$2" == "add" ]]; then
  name="$3"
  url="$4"
  printf "%s\n" "$url" > "$repo_root/.git-remote-$name"
  exit 0
fi

if [[ "$#" -ge 2 && "$1" == "remote" && "$2" == "set-url" ]]; then
  name="$3"
  url="$4"
  printf "%s\n" "$url" > "$repo_root/.git-remote-$name"
  exit 0
fi

if [[ "$#" -ge 2 && "$1" == "push" && "$2" == "origin" ]]; then
  exit 0
fi

if [[ "$#" -ge 2 && "$1" == "subtree" && "$2" == "push" ]]; then
  exit 0
fi

exit 0
`
