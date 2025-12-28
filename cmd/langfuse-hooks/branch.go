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

// suggestBranch suggests a branch name based on a description.
func suggestBranch(ctx context.Context, description string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !cfg.Hooks.BranchSuggest.Enabled {
		return nil
	}

	// Check if we're in a git repository
	if !git.IsGitRepository() {
		return fmt.Errorf("not in a git repository")
	}

	// Create LLM provider
	llm, err := provider.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create LLM provider: %w", err)
	}

	// Generate prompt
	promptText := prompt.BranchNamePrompt(
		description,
		cfg.Hooks.BranchSuggest.Format,
		cfg.Hooks.BranchSuggest.MaxLength,
	)

	// Send request to LLM
	resp, err := llm.Complete(ctx, &provider.Request{
		Messages: []provider.Message{
			{Role: "user", Content: promptText},
		},
	})
	if err != nil {
		return fmt.Errorf("LLM request failed: %w", err)
	}

	// Clean up the branch name
	branchName := cleanBranchName(resp.Content)

	// Enforce max length
	if cfg.Hooks.BranchSuggest.MaxLength > 0 && len(branchName) > cfg.Hooks.BranchSuggest.MaxLength {
		branchName = branchName[:cfg.Hooks.BranchSuggest.MaxLength]
	}

	fmt.Println(branchName)

	return nil
}

// cleanBranchName sanitizes a branch name.
func cleanBranchName(name string) string {
	name = strings.TrimSpace(name)

	// Remove any backticks or quotes that might be in LLM output
	name = strings.Trim(name, "`\"'")

	// Replace spaces with hyphens
	name = strings.ReplaceAll(name, " ", "-")

	// Convert to lowercase
	name = strings.ToLower(name)

	// Remove any characters that aren't alphanumeric, hyphens, or slashes
	var cleaned strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '/' || r == '_' {
			cleaned.WriteRune(r)
		}
	}

	return cleaned.String()
}
