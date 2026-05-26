package cmd

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
)

var projectsSyncCmd = &cobra.Command{
	Use:   "sync [project]",
	Short: "Scan repo and update .one/projects.json",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRef, _, err := projectRefFromInput("", args)
		if err != nil {
			return err
		}

		repoRoot, err := getRepoRoot()
		if err != nil {
			return err
		}

		existing, err := loadProjectsIndex(repoRoot)
		if err != nil {
			return err
		}

		aliasByPath := map[string]string{}
		if existing != nil {
			for _, project := range existing.Projects {
				if project.Alias != "" {
					aliasByPath[project.Path] = project.Alias
				}
			}
		}

		idx, err := syncProjectsIndex(repoRoot, existing, aliasByPath, projectRef)
		if err != nil {
			return err
		}

		if err := writeProjectsIndex(repoRoot, idx); err != nil {
			return err
		}

		fmt.Printf("Synced %d projects to %s\n", len(idx.Projects), projectsIndexPath(repoRoot))
		return nil
	},
}

func init() {
	projectsCmd.AddCommand(projectsSyncCmd)
}

func syncProjectsIndex(repoRoot string, existing *projectsIndex, aliasByPath map[string]string, projectRef string) (*projectsIndex, error) {
	idx := &projectsIndex{Projects: []indexProject{}}
	if existing != nil {
		idx.DefaultProject = existing.DefaultProject
		idx.Projects = existing.Projects
	}

	if projectRef == "" {
		projects, err := scanProjects(repoRoot, aliasByPath)
		if err != nil {
			return nil, err
		}

		idx.Projects = projects
		return idx, nil
	}

	projectDir, err := resolveExplicitProjectDir(projectRef, repoRoot, existing)
	if err != nil {
		return nil, err
	}

	project, err := scanSingleProject(repoRoot, projectDir, aliasByPath)
	if err != nil {
		return nil, err
	}

	idx.Projects = upsertProject(idx.Projects, project)
	if err := validateUniqueAliases(idx.Projects); err != nil {
		return nil, err
	}

	sort.Slice(idx.Projects, func(i, j int) bool {
		return idx.Projects[i].Path < idx.Projects[j].Path
	})

	return idx, nil
}

func scanProjects(repoRoot string, aliasByPath map[string]string) ([]indexProject, error) {
	projects := make([]indexProject, 0)
	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == ".one" {
				return filepath.SkipDir
			}
			return nil
		}

		if d.Name() != projectConfigFileName {
			return nil
		}

		cfg, parseErr := parseProjectConfigFile(path)
		if parseErr != nil {
			return parseErr
		}

		projectDir := filepath.Dir(path)
		rel, relErr := filepath.Rel(repoRoot, projectDir)
		if relErr != nil {
			return relErr
		}

		rel = filepath.Clean(rel)
		if rel == "." {
			return nil
		}

		project := indexProject{
			Name: cfg.Name,
			Desc: cfg.Desc,
			Path: rel,
		}

		if cfg.Alias != "" {
			project.Alias = cfg.Alias
		} else if keptAlias, ok := aliasByPath[rel]; ok {
			project.Alias = keptAlias
		}

		projects = append(projects, project)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan projects: %w", err)
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Path < projects[j].Path
	})

	if err := validateUniqueAliases(projects); err != nil {
		return nil, err
	}

	return projects, nil
}

func scanSingleProject(repoRoot, projectDir string, aliasByPath map[string]string) (indexProject, error) {
	projectFile := filepath.Join(projectDir, projectConfigFileName)
	cfg, err := parseProjectConfigFile(projectFile)
	if err != nil {
		return indexProject{}, err
	}

	rel, err := pathFromRoot(repoRoot, projectDir)
	if err != nil {
		return indexProject{}, err
	}

	project := indexProject{
		Name: cfg.Name,
		Desc: cfg.Desc,
		Path: rel,
	}

	if cfg.Alias != "" {
		project.Alias = cfg.Alias
	} else if keptAlias, ok := aliasByPath[rel]; ok {
		project.Alias = keptAlias
	}

	return project, nil
}

func upsertProject(projects []indexProject, project indexProject) []indexProject {
	for i := range projects {
		if projects[i].Path == project.Path {
			projects[i] = project
			return projects
		}
	}

	return append(projects, project)
}

func validateUniqueAliases(projects []indexProject) error {
	seen := map[string]string{}
	for _, project := range projects {
		if project.Alias == "" {
			continue
		}

		if existingPath, ok := seen[project.Alias]; ok {
			return fmt.Errorf("duplicate alias %q for %s and %s", project.Alias, existingPath, project.Path)
		}

		seen[project.Alias] = project.Path
	}

	return nil
}
