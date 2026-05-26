package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func gitRun(args ...string) error {
	command := exec.Command("git", args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if err := command.Run(); err != nil {
		return fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}

	return nil
}

func gitOutput(args ...string) (string, error) {
	command := exec.Command("git", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func execOutput(name string, args ...string) (string, error) {
	command := exec.Command(name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), msg)
	}

	return strings.TrimSpace(stdout.String()), nil
}
