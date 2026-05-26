package cmd

import (
	"fmt"
	"sort"
	"strings"
)

func loadProjectRemoteMap(projectRef string) (map[string]string, *projectContext, error) {
	cfg, ctx, err := loadProjectConfig(projectRef)
	if err != nil {
		return nil, nil, err
	}

	if len(cfg.Git.Remotes) == 0 {
		return nil, ctx, nil
	}

	return cfg.Git.Remotes, ctx, nil
}

func sortedRemoteNames(remotes map[string]string) []string {
	names := make([]string, 0, len(remotes))
	for name := range remotes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func addOrUpdateRemote(name, url string, create bool, public bool) error {
	if create {
		if err := ensureGithubRepoExistsWithVisibility(url, public); err != nil {
			return err
		}
	}

	remoteURL, err := gitOutput("remote", "get-url", name)
	if err == nil {
		if remoteURL == url {
			fmt.Printf("Remote %s already configured\n", name)
			return nil
		}

		if err := gitRun("remote", "set-url", name, url); err != nil {
			return err
		}

		fmt.Printf("Updated remote %s -> %s\n", name, url)
		return nil
	}

	if !isUnknownRemoteErr(err) {
		return err
	}

	if err := gitRun("remote", "add", name, url); err != nil {
		return err
	}

	fmt.Printf("Added remote %s -> %s\n", name, url)
	return nil
}

func listGitRemotes() ([]string, error) {
	output, err := gitOutput("remote")
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(output) == "" {
		return []string{}, nil
	}

	remotes := strings.Split(output, "\n")
	sort.Strings(remotes)
	return remotes, nil
}

func isUnknownRemoteErr(err error) bool {
	return strings.Contains(err.Error(), "No such remote")
}
