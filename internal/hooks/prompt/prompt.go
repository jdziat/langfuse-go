// Package prompt provides prompt templates for LLM interactions.
package prompt

import (
	"strconv"
	"strings"
)

// CommitMessagePrompt generates a prompt for commit message generation.
func CommitMessagePrompt(diff string, files []string, conventions string, recentCommits []string) string {
	var sb strings.Builder

	sb.WriteString(`You are a commit message generator for a Go SDK project.

Repository: langfuse-go (Langfuse observability SDK for Go)
Conventions: Conventional Commits (feat:, fix:, docs:, test:, refactor:, perf:, chore:)

`)

	if conventions != "" {
		sb.WriteString("Project conventions:\n")
		sb.WriteString(conventions)
		sb.WriteString("\n\n")
	}

	if len(recentCommits) > 0 {
		sb.WriteString("Recent commit messages (for style reference):\n")
		for _, c := range recentCommits {
			sb.WriteString("- ")
			sb.WriteString(c)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Staged changes:\n```diff\n")
	sb.WriteString(diff)
	sb.WriteString("\n```\n\n")

	if len(files) > 0 {
		sb.WriteString("Files changed:\n")
		for _, f := range files {
			sb.WriteString("- ")
			sb.WriteString(f)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(`Generate a commit message following these rules:
1. First line: type(scope): description (max 72 chars)
2. Blank line after first line
3. Body explaining what and why (wrap at 72 chars)
4. Use present tense ("add" not "added")
5. Reference issues if mentioned in diff

Valid types: feat, fix, docs, test, refactor, perf, chore, ci, build
Valid scopes for this project: client, config, ingestion, traces, observations, scores, prompts, datasets, sessions, models, http, errors, hooks

Respond with ONLY the commit message, no explanations or markdown formatting.`)

	return sb.String()
}

// FixCommitMessagePrompt generates a prompt to fix a non-conventional commit message.
func FixCommitMessagePrompt(originalMessage string, conventions string) string {
	var sb strings.Builder

	sb.WriteString(`You are a commit message editor. The following commit message does not follow conventional commit format.

Original message:
`)
	sb.WriteString(originalMessage)
	sb.WriteString("\n\n")

	if conventions != "" {
		sb.WriteString("Project conventions:\n")
		sb.WriteString(conventions)
		sb.WriteString("\n\n")
	}

	sb.WriteString(`Rewrite this message to follow conventional commits format:
1. First line: type(scope): description (max 72 chars)
2. Blank line after first line (if body needed)
3. Body explaining what and why (wrap at 72 chars)

Valid types: feat, fix, docs, test, refactor, perf, chore, ci, build

Preserve the intent and meaning of the original message.
Respond with ONLY the improved commit message, no explanations.`)

	return sb.String()
}

// BranchNamePrompt generates a prompt for branch name suggestion.
func BranchNamePrompt(description string, format string, maxLength int) string {
	var sb strings.Builder

	sb.WriteString(`Generate a git branch name for the following work:

Description: `)
	sb.WriteString(description)
	sb.WriteString("\n\n")

	sb.WriteString("Rules:\n")
	if format != "" {
		sb.WriteString("1. Format: ")
		sb.WriteString(format)
		sb.WriteString("\n")
	} else {
		sb.WriteString("1. Format: {type}/{description}\n")
	}

	sb.WriteString(`2. Types: feat, fix, docs, test, refactor, perf, chore
3. Use lowercase with hyphens (no spaces or underscores)
4. Be descriptive but concise
`)

	if maxLength > 0 {
		sb.WriteString("5. Max length: ")
		sb.WriteString(strconv.Itoa(maxLength))
		sb.WriteString(" characters\n")
	}

	sb.WriteString(`
Examples:
- feat/add-prometheus-metrics
- fix/handle-nil-trace-context
- docs/improve-api-examples
- refactor/simplify-batch-logic

If the description mentions a ticket/issue number (like LANG-123 or #456), include it:
- feat/LANG-123-add-metrics
- fix/456-nil-pointer

Respond with ONLY the branch name, no explanations.`)

	return sb.String()
}
