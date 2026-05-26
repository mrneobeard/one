package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	projectConfigFileName   = "one.yaml"
	projectsIndexRelPath    = ".one/projects.json"
	defaultProjectsFilePerm = 0o644
)

type projectConfig struct {
	Name  string    `yaml:"name"`
	Alias string    `yaml:"alias"`
	Desc  string    `yaml:"desc"`
	Git   gitConfig `yaml:"git"`
}

type gitConfig struct {
	Remotes map[string]string `yaml:"remotes"`
	Push    syncConfig        `yaml:"push"`
	Sync    syncConfig        `yaml:"sync"`
}

type syncConfig struct {
	Subtree bool     `yaml:"subtree"`
	Remotes []string `yaml:"remotes"`
}

type projectContext struct {
	RepoRoot       string
	ProjectDir     string
	ProjectRelPath string
	ProjectFile    string
}

type projectsIndex struct {
	DefaultProject string         `json:"default_project,omitempty"`
	Projects       []indexProject `json:"projects"`
}

type indexProject struct {
	Name  string `json:"name"`
	Alias string `json:"alias,omitempty"`
	Desc  string `json:"desc,omitempty"`
	Path  string `json:"path"`
}

func loadProjectConfig(projectRef string) (*projectConfig, *projectContext, error) {
	repoRoot, err := getRepoRoot()
	if err != nil {
		return nil, nil, err
	}

	idx, err := loadProjectsIndex(repoRoot)
	if err != nil {
		return nil, nil, err
	}

	projectDir, err := resolveProjectDir(projectRef, repoRoot, idx)
	if err != nil {
		return nil, nil, err
	}

	projectFile := filepath.Join(projectDir, projectConfigFileName)
	cfg, err := parseProjectConfigFile(projectFile)
	if err != nil {
		return nil, nil, err
	}

	rel, err := filepath.Rel(repoRoot, projectDir)
	if err != nil {
		return nil, nil, fmt.Errorf("compute project path: %w", err)
	}

	ctx := &projectContext{
		RepoRoot:       repoRoot,
		ProjectDir:     projectDir,
		ProjectRelPath: filepath.Clean(rel),
		ProjectFile:    projectFile,
	}

	return cfg, ctx, nil
}

func parseProjectConfigFile(projectFile string) (*projectConfig, error) {
	abs, err := filepath.Abs(projectFile)
	if err != nil {
		return nil, fmt.Errorf("resolve project file path: %w", err)
	}

	raw, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", abs, err)
	}

	var cfg projectConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", abs, err)
	}

	return &cfg, nil
}

func subtreeSyncConfig(gitCfg gitConfig) syncConfig {
	if gitCfg.Sync.Subtree || len(gitCfg.Sync.Remotes) > 0 {
		return gitCfg.Sync
	}

	return gitCfg.Push
}

func getRepoRoot() (string, error) {
	if override := os.Getenv("ONE_TEST_REPO_ROOT"); override != "" {
		return filepath.Clean(override), nil
	}

	repoRoot, err := gitOutput("rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}

	return filepath.Clean(repoRoot), nil
}

func projectsIndexPath(repoRoot string) string {
	return filepath.Join(repoRoot, projectsIndexRelPath)
}

func loadProjectsIndex(repoRoot string) (*projectsIndex, error) {
	path := projectsIndexPath(repoRoot)
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var idx projectsIndex
	if err := json.Unmarshal(raw, &idx); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	return &idx, nil
}

func writeProjectsIndex(repoRoot string, idx *projectsIndex) error {
	if idx.Projects == nil {
		idx.Projects = []indexProject{}
	}

	dir := filepath.Join(repoRoot, ".one")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}

	raw, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("encode projects index: %w", err)
	}

	raw = append(raw, '\n')
	path := projectsIndexPath(repoRoot)
	if err := os.WriteFile(path, raw, defaultProjectsFilePerm); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

func resolveProjectDir(projectRef, repoRoot string, idx *projectsIndex) (string, error) {
	if projectRef != "" {
		projectDir, err := resolveExplicitProjectDir(projectRef, repoRoot, idx)
		if err != nil {
			return "", err
		}

		if _, err := os.Stat(filepath.Join(projectDir, projectConfigFileName)); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return "", fmt.Errorf("%s does not contain %s", projectDir, projectConfigFileName)
			}

			return "", fmt.Errorf("check %s: %w", projectDir, err)
		}

		return projectDir, nil
	}

	if idx != nil && idx.DefaultProject != "" {
		projectDir := filepath.Join(repoRoot, filepath.Clean(idx.DefaultProject))
		if stat, err := os.Stat(projectDir); err == nil && stat.IsDir() {
			return filepath.Clean(projectDir), nil
		}
	}

	return "", fmt.Errorf("no project selected: set --project to a directory or run `one projects use`")
}

func resolveExplicitProjectDir(projectRef, repoRoot string, idx *projectsIndex) (string, error) {
	for _, candidate := range projectCandidates(projectRef, repoRoot) {
		projectDir, ok, err := projectDirFromCandidate(candidate)
		if err != nil {
			return "", err
		}
		if !ok {
			continue
		}

		if !isWithinRepo(repoRoot, projectDir) {
			return "", fmt.Errorf("project directory must be inside repo root %s", repoRoot)
		}

		return filepath.Clean(projectDir), nil
	}

	if idx != nil {
		if byAlias, ok := indexProjectByAliasOrName(*idx, projectRef); ok {
			return filepath.Join(repoRoot, byAlias.Path), nil
		}
	}

	return "", fmt.Errorf("project %q not found as directory or known alias", projectRef)
}

func projectCandidates(projectRef, repoRoot string) []string {
	candidates := []string{}

	absRef, err := filepath.Abs(projectRef)
	if err == nil {
		candidates = append(candidates, absRef)
	}

	if !filepath.IsAbs(projectRef) {
		candidates = append(candidates, filepath.Join(repoRoot, projectRef))
	}

	return candidates
}

func projectDirFromCandidate(candidate string) (string, bool, error) {
	stat, err := os.Stat(candidate)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}

		return "", false, fmt.Errorf("check %s: %w", candidate, err)
	}

	if stat.IsDir() {
		return candidate, true, nil
	}

	if filepath.Base(candidate) == projectConfigFileName {
		return filepath.Dir(candidate), true, nil
	}

	return "", false, fmt.Errorf("project reference must be a directory, alias, or %s path", projectConfigFileName)
}

func pathFromRoot(repoRoot, path string) (string, error) {
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return "", fmt.Errorf("compute relative path: %w", err)
	}

	clean := filepath.Clean(rel)
	if clean == "." {
		return "", fmt.Errorf("project path cannot be repo root")
	}

	return clean, nil
}

func indexProjectByAliasOrName(idx projectsIndex, selector string) (indexProject, bool) {
	for _, project := range idx.Projects {
		if project.Alias == selector || project.Name == selector || project.Path == selector {
			return project, true
		}
	}

	return indexProject{}, false
}

func isWithinRepo(repoRoot, candidate string) bool {
	rel, err := filepath.Rel(repoRoot, candidate)
	if err != nil {
		return false
	}

	return rel == "." || (!strings.HasPrefix(rel, "..") && rel != "")
}

func projectRefFromInput(flagValue string, args []string) (string, bool, error) {
	if flagValue != "" && len(args) > 0 {
		return "", false, fmt.Errorf("provide project as either argument or --project, not both")
	}

	if len(args) > 1 {
		return "", false, fmt.Errorf("accepts at most one project argument")
	}

	if flagValue != "" {
		return flagValue, true, nil
	}

	if len(args) == 1 {
		return args[0], false, nil
	}

	return "", false, nil
}

func validateProjectFlagDirectory(projectFlag string) error {
	if projectFlag == "" {
		return nil
	}

	abs, err := filepath.Abs(projectFlag)
	if err != nil {
		return fmt.Errorf("resolve project path: %w", err)
	}

	stat, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("--project must point to an existing directory: %w", err)
	}

	if !stat.IsDir() {
		return fmt.Errorf("--project must be a directory")
	}

	return nil
}
