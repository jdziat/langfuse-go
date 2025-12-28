// Package main provides the langfuse-hooks CLI tool for git hook integration.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jdziat/langfuse-go/internal/hooks/config"
)

const (
	version = "1.0.0"
	timeout = 30 * time.Second
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Check if hooks are disabled
	if config.IsDisabled() {
		os.Exit(0)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var err error
	switch os.Args[1] {
	case "generate-commit":
		err = generateCommit(ctx)
	case "fix-commit":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: langfuse-hooks fix-commit <message>")
			os.Exit(1)
		}
		err = fixCommit(ctx, os.Args[2])
	case "suggest-branch":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: langfuse-hooks suggest-branch <description>")
			os.Exit(1)
		}
		err = suggestBranch(ctx, os.Args[2])
	case "install":
		err = install()
	case "uninstall":
		err = uninstall()
	case "version", "--version", "-v":
		fmt.Printf("langfuse-hooks version %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`langfuse-hooks - Git hooks with LLM-powered commit messages and branch names

Usage:
  langfuse-hooks <command> [arguments]

Commands:
  generate-commit       Generate a commit message from staged changes
  fix-commit <message>  Fix a commit message to follow conventional format
  suggest-branch <desc> Suggest a branch name based on description
  install               Install git hooks in the current repository
  uninstall             Remove git hooks from the current repository
  version               Print version information
  help                  Show this help message

Environment Variables:
  OPENAI_API_KEY          OpenAI API key (required for openai provider)
  ANTHROPIC_API_KEY       Anthropic API key (required for anthropic provider)
  LANGFUSE_HOOKS_PROVIDER Override the configured provider
  LANGFUSE_HOOKS_MODEL    Override the configured model
  LANGFUSE_HOOKS_DISABLED Set to "true" to disable hooks

Configuration:
  Create .langfuse-hooks.yaml in your repository root.
  See https://github.com/jdziat/langfuse-go for configuration options.`)
}
