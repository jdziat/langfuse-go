package prompt

import (
	"strings"
	"testing"
)

func TestCommitMessagePrompt(t *testing.T) {
	diff := `diff --git a/file.go b/file.go
--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
+// New comment
 package main`

	files := []string{"file.go"}
	conventions := "Use conventional commits"
	recentCommits := []string{"feat: add feature", "fix: fix bug"}

	prompt := CommitMessagePrompt(diff, files, conventions, recentCommits)

	// Check that key elements are present
	if !strings.Contains(prompt, "langfuse-go") {
		t.Error("prompt should contain repository name")
	}

	if !strings.Contains(prompt, "Conventional Commits") {
		t.Error("prompt should mention conventional commits")
	}

	if !strings.Contains(prompt, diff) {
		t.Error("prompt should contain the diff")
	}

	if !strings.Contains(prompt, "file.go") {
		t.Error("prompt should contain the changed file")
	}

	if !strings.Contains(prompt, "Use conventional commits") {
		t.Error("prompt should contain project conventions")
	}

	if !strings.Contains(prompt, "feat: add feature") {
		t.Error("prompt should contain recent commits")
	}

	if !strings.Contains(prompt, "type(scope): description") {
		t.Error("prompt should explain format")
	}
}

func TestCommitMessagePromptEmpty(t *testing.T) {
	prompt := CommitMessagePrompt("", nil, "", nil)

	// Should still produce a valid prompt
	if !strings.Contains(prompt, "commit message generator") {
		t.Error("empty inputs should still produce a valid prompt")
	}
}

func TestFixCommitMessagePrompt(t *testing.T) {
	originalMessage := "added some stuff"
	conventions := "Use conventional commits"

	prompt := FixCommitMessagePrompt(originalMessage, conventions)

	if !strings.Contains(prompt, originalMessage) {
		t.Error("prompt should contain original message")
	}

	if !strings.Contains(prompt, conventions) {
		t.Error("prompt should contain conventions")
	}

	if !strings.Contains(prompt, "conventional commits format") {
		t.Error("prompt should mention conventional format")
	}

	if !strings.Contains(prompt, "type(scope): description") {
		t.Error("prompt should explain expected format")
	}
}

func TestBranchNamePrompt(t *testing.T) {
	description := "add user authentication"
	format := "{type}/{ticket}-{description}"
	maxLength := 50

	prompt := BranchNamePrompt(description, format, maxLength)

	if !strings.Contains(prompt, description) {
		t.Error("prompt should contain description")
	}

	if !strings.Contains(prompt, format) {
		t.Error("prompt should contain format")
	}

	if !strings.Contains(prompt, "50") {
		t.Error("prompt should contain max length")
	}

	if !strings.Contains(prompt, "lowercase with hyphens") {
		t.Error("prompt should explain naming rules")
	}

	if !strings.Contains(prompt, "feat/") {
		t.Error("prompt should contain examples")
	}
}

func TestBranchNamePromptDefaults(t *testing.T) {
	prompt := BranchNamePrompt("test feature", "", 0)

	// Should use default format
	if !strings.Contains(prompt, "{type}/{description}") {
		t.Error("prompt should use default format when empty")
	}

	// Should not mention max length when 0
	if strings.Contains(prompt, "Max length: 0") {
		t.Error("prompt should not mention max length when 0")
	}
}

func TestBranchNamePromptWithTicket(t *testing.T) {
	prompt := BranchNamePrompt("LANG-123 add metrics", "", 50)

	if !strings.Contains(prompt, "LANG-123") {
		t.Error("prompt should contain ticket number from description")
	}

	// Should have ticket examples
	if !strings.Contains(prompt, "feat/LANG-123-add-metrics") {
		t.Error("prompt should have ticket-based examples")
	}
}
