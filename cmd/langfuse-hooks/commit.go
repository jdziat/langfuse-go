package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/jdziat/langfuse-go/internal/hooks/config"
	"github.com/jdziat/langfuse-go/internal/hooks/git"
	"github.com/jdziat/langfuse-go/internal/hooks/prompt"
	"github.com/jdziat/langfuse-go/internal/hooks/provider"
)

// generateCommit generates a commit message from staged changes.
func generateCommit(ctx context.Context) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !cfg.Hooks.PrepareCommitMsg.Enabled {
		return nil
	}

	// Check if we're in a git repository
	if !git.IsGitRepository() {
		return fmt.Errorf("not in a git repository")
	}

	// Get staged diff
	diff, err := git.GetStagedDiff()
	if err != nil {
		return fmt.Errorf("failed to get staged diff: %w", err)
	}

	if strings.TrimSpace(diff) == "" {
		return fmt.Errorf("no staged changes")
	}

	// Truncate diff if needed
	if cfg.Hooks.PrepareCommitMsg.MaxDiffLines > 0 {
		diff = git.TruncateDiff(diff, cfg.Hooks.PrepareCommitMsg.MaxDiffLines)
	}

	// Get staged files
	files, err := git.GetStagedFiles()
	if err != nil {
		return fmt.Errorf("failed to get staged files: %w", err)
	}

	// Get recent commit messages for style reference
	recentCommits, _ := git.GetRecentCommitMessages(5)

	// Create LLM provider
	llm, err := provider.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create LLM provider: %w", err)
	}

	// Generate prompt
	promptText := prompt.CommitMessagePrompt(diff, files, cfg.Context.Conventions, recentCommits)

	// Send request to LLM
	resp, err := llm.Complete(ctx, &provider.Request{
		Messages: []provider.Message{
			{Role: "user", Content: promptText},
		},
	})
	if err != nil {
		return fmt.Errorf("LLM request failed: %w", err)
	}

	// Output the generated message
	message := strings.TrimSpace(resp.Content)
	fmt.Println(message)

	return nil
}

// fixCommit fixes a commit message to follow conventional format.
func fixCommit(ctx context.Context, originalMessage string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !cfg.Hooks.CommitMsg.Enabled {
		return nil
	}

	// Create LLM provider
	llm, err := provider.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create LLM provider: %w", err)
	}

	// Generate prompt
	promptText := prompt.FixCommitMessagePrompt(originalMessage, cfg.Context.Conventions)

	// Send request to LLM
	resp, err := llm.Complete(ctx, &provider.Request{
		Messages: []provider.Message{
			{Role: "user", Content: promptText},
		},
	})
	if err != nil {
		return fmt.Errorf("LLM request failed: %w", err)
	}

	// Output the fixed message
	message := strings.TrimSpace(resp.Content)
	fmt.Println(message)

	return nil
}
