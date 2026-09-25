package main

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func ensureGitRepository(dir string) error {
	output, err := git(dir, "rev-parse", "--is-inside-work-tree")
	if err != nil || strings.TrimSpace(output) != "true" {
		return errors.New("the project must be a Git working tree")
	}
	if _, err := git(dir, "rev-parse", "--verify", "HEAD"); err != nil {
		return errors.New("the Git repository must have an initial commit")
	}
	return nil
}

func scopedPaths(paths []string) []string {
	result := append([]string(nil), paths...)
	for _, configPath := range configRelativePaths {
		result = append(result, ":(exclude)"+configPath)
	}
	return result
}

func dirtyPaths(dir string, paths []string) (string, error) {
	args := []string{"status", "--porcelain=v1", "--untracked-files=all"}
	if len(paths) > 0 {
		args = append(args, "--")
		args = append(args, paths...)
	}
	output, err := git(dir, args...)
	return strings.TrimSpace(output), err
}

func acceptChanges(dir string, paths []string, summary string) (string, error) {
	args := append([]string{"add", "--"}, paths...)
	if _, err := git(dir, args...); err != nil {
		return "", fmt.Errorf("stage candidate: %w", err)
	}
	staged, err := git(dir, "diff", "--cached", "--name-only")
	if err != nil {
		return "", fmt.Errorf("inspect staged candidate: %w", err)
	}
	if strings.TrimSpace(staged) == "" {
		return "", errors.New("candidate produced no staged changes")
	}
	summary = strings.TrimSpace(strings.Split(summary, "\n")[0])
	if len(summary) > 72 {
		summary = summary[:72]
	}
	if _, err := git(dir, "commit", "-m", "perf(autoresearch): "+summary); err != nil {
		return "", err
	}
	commit, err := git(dir, "rev-parse", "--short", "HEAD")
	return strings.TrimSpace(commit), err
}

func restorePaths(dir string, paths []string) error {
	args := append([]string{"restore", "--source=HEAD", "--staged", "--worktree", "--"}, paths...)
	if output, err := git(dir, args...); err != nil {
		return fmt.Errorf("restore tracked files: %w (%s)", err, strings.TrimSpace(output))
	}
	args = append([]string{"clean", "-fd", "--"}, paths...)
	if output, err := git(dir, args...); err != nil {
		return fmt.Errorf("remove candidate-created files: %w (%s)", err, strings.TrimSpace(output))
	}
	return nil
}

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	if err != nil {
		return output.String(), fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(output.String()))
	}
	return output.String(), nil
}
