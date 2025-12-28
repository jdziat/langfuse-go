// Package git provides utilities for interacting with git repositories.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// GetStagedDiff returns the diff of staged changes.
func GetStagedDiff() (string, error) {
	cmd := exec.Command("git", "diff", "--cached", "--no-color")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git diff failed: %s", stderr.String())
	}

	return stdout.String(), nil
}

// GetStagedFiles returns a list of staged file names.
func GetStagedFiles() ([]string, error) {
	cmd := exec.Command("git", "diff", "--cached", "--name-only")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git diff failed: %s", stderr.String())
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return nil, nil
	}

	return strings.Split(output, "\n"), nil
}

// GetStagedFileStats returns file change statistics for staged files.
func GetStagedFileStats() (string, error) {
	cmd := exec.Command("git", "diff", "--cached", "--stat", "--no-color")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git diff failed: %s", stderr.String())
	}

	return stdout.String(), nil
}

// GetRecentCommitMessages returns the most recent commit messages.
func GetRecentCommitMessages(count int) ([]string, error) {
	cmd := exec.Command("git", "log", fmt.Sprintf("-%d", count), "--pretty=format:%s")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git log failed: %s", stderr.String())
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return nil, nil
	}

	return strings.Split(output, "\n"), nil
}

// GetCurrentBranch returns the current branch name.
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git branch failed: %s", stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// IsGitRepository checks if the current directory is a git repository.
func IsGitRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// GetGitRoot returns the root directory of the git repository.
func GetGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("not in a git repository: %s", stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// GetHooksDir returns the path to the git hooks directory.
func GetHooksDir() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--git-path", "hooks")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get hooks directory: %s", stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// TruncateDiff truncates a diff to the specified number of lines.
func TruncateDiff(diff string, maxLines int) string {
	if maxLines <= 0 {
		return diff
	}

	lines := strings.Split(diff, "\n")
	if len(lines) <= maxLines {
		return diff
	}

	truncated := strings.Join(lines[:maxLines], "\n")
	return truncated + fmt.Sprintf("\n\n... (truncated %d lines)", len(lines)-maxLines)
}

// RenameBranch renames the current branch.
func RenameBranch(newName string) error {
	cmd := exec.Command("git", "branch", "-m", newName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git branch rename failed: %s", stderr.String())
	}

	return nil
}
