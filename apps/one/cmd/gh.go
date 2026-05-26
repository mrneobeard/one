package cmd

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var githubSSHRemotePattern = regexp.MustCompile(`^git@github\.com:([^/]+)/([^/]+?)(\.git)?$`)
var githubHTTPSRemotePattern = regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+?)(\.git)?$`)

func ensureGithubRepoExists(remoteURL string) error {
	owner, repo, ok := parseGithubRepo(remoteURL)
	if !ok {
		return nil
	}

	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("--create requested but gh not found in PATH")
	}

	if err := ghRepoView(owner, repo); err == nil {
		return nil
	}

	if err := ghRepoCreate(owner, repo); err != nil {
		return err
	}

	return nil
}

func parseGithubRepo(remoteURL string) (string, string, bool) {
	remoteURL = strings.TrimSpace(remoteURL)
	if matches := githubSSHRemotePattern.FindStringSubmatch(remoteURL); len(matches) == 4 {
		return matches[1], strings.TrimSuffix(matches[2], ".git"), true
	}

	if matches := githubHTTPSRemotePattern.FindStringSubmatch(remoteURL); len(matches) == 4 {
		return matches[1], strings.TrimSuffix(matches[2], ".git"), true
	}

	return "", "", false
}

func ghRepoView(owner, repo string) error {
	_, err := execOutput("gh", "repo", "view", owner+"/"+repo)
	return err
}

func ghRepoCreate(owner, repo string) error {
	if _, err := execOutput("gh", "repo", "create", owner+"/"+repo, "--private", "--confirm"); err != nil {
		return fmt.Errorf("create github repo %s/%s: %w", owner, repo, err)
	}

	return nil
}
