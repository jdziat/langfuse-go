package git

import (
	"strings"
	"testing"
)

func TestTruncateDiff_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		diff       string
		maxLines   int
		wantTrunc  bool
		wantSuffix string
	}{
		{
			name:      "single line no truncation",
			diff:      "single line",
			maxLines:  5,
			wantTrunc: false,
		},
		{
			name:      "exact limit",
			diff:      "line1\nline2\nline3",
			maxLines:  3,
			wantTrunc: false,
		},
		{
			name:       "one over limit",
			diff:       "line1\nline2\nline3\nline4",
			maxLines:   3,
			wantTrunc:  true,
			wantSuffix: "truncated 1 lines",
		},
		{
			name:       "many over limit",
			diff:       "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10",
			maxLines:   3,
			wantTrunc:  true,
			wantSuffix: "truncated 7 lines",
		},
		{
			name:      "max lines equals 1",
			diff:      "line1\nline2\nline3",
			maxLines:  1,
			wantTrunc: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateDiff(tt.diff, tt.maxLines)

			if tt.wantTrunc {
				if !strings.Contains(result, "truncated") {
					t.Error("expected truncation message")
				}
				if tt.wantSuffix != "" && !strings.Contains(result, tt.wantSuffix) {
					t.Errorf("expected suffix %q, got %q", tt.wantSuffix, result)
				}
			} else {
				if result != tt.diff {
					t.Errorf("expected unchanged diff, got %q", result)
				}
			}
		})
	}
}

func TestTruncateDiff_PreservesContent(t *testing.T) {
	diff := "line1\nline2\nline3\nline4\nline5"
	result := TruncateDiff(diff, 3)

	// Should contain the first 3 lines
	if !strings.Contains(result, "line1") {
		t.Error("should contain line1")
	}
	if !strings.Contains(result, "line2") {
		t.Error("should contain line2")
	}
	if !strings.Contains(result, "line3") {
		t.Error("should contain line3")
	}

	// Should NOT contain line4 and line5 in the content (only in truncation msg)
	lines := strings.Split(result, "\n")
	contentLines := lines[:3]
	content := strings.Join(contentLines, "\n")
	if strings.Contains(content, "line4") {
		t.Error("should not contain line4 in content")
	}
}

func TestGetStagedDiff(t *testing.T) {
	if !IsGitRepository() {
		t.Skip("Not running in a git repository")
	}

	// This just tests that the function runs without error
	// We can't easily test the output without staging files
	_, err := GetStagedDiff()
	if err != nil {
		t.Fatalf("GetStagedDiff() error = %v", err)
	}
}

func TestGetStagedFiles(t *testing.T) {
	if !IsGitRepository() {
		t.Skip("Not running in a git repository")
	}

	// This just tests that the function runs without error
	files, err := GetStagedFiles()
	if err != nil {
		t.Fatalf("GetStagedFiles() error = %v", err)
	}

	// files can be nil if nothing is staged, which is fine
	t.Logf("Staged files: %v", files)
}

func TestGetStagedFileStats(t *testing.T) {
	if !IsGitRepository() {
		t.Skip("Not running in a git repository")
	}

	stats, err := GetStagedFileStats()
	if err != nil {
		t.Fatalf("GetStagedFileStats() error = %v", err)
	}

	// Stats can be empty if nothing is staged
	t.Logf("Staged file stats: %s", stats)
}

func TestGetRecentCommitMessages(t *testing.T) {
	if !IsGitRepository() {
		t.Skip("Not running in a git repository")
	}

	tests := []struct {
		count int
	}{
		{count: 1},
		{count: 5},
		{count: 10},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			messages, err := GetRecentCommitMessages(tt.count)
			if err != nil {
				t.Fatalf("GetRecentCommitMessages(%d) error = %v", tt.count, err)
			}

			// Should return at most count messages
			if len(messages) > tt.count {
				t.Errorf("got %d messages, expected at most %d", len(messages), tt.count)
			}

			t.Logf("Recent commit messages (%d): %v", tt.count, messages)
		})
	}
}

func TestGetRecentCommitMessages_ZeroCount(t *testing.T) {
	if !IsGitRepository() {
		t.Skip("Not running in a git repository")
	}

	messages, err := GetRecentCommitMessages(0)
	if err != nil {
		t.Fatalf("GetRecentCommitMessages(0) error = %v", err)
	}

	// With count 0, should return empty or nil
	if len(messages) > 0 {
		t.Logf("Unexpectedly got messages: %v", messages)
	}
}

func TestIsGitRepository_InRepo(t *testing.T) {
	// We're definitely in a git repo (the langfuse-go project)
	if !IsGitRepository() {
		t.Error("IsGitRepository() should return true in a git repository")
	}
}

func TestGetCurrentBranch_NotEmpty(t *testing.T) {
	if !IsGitRepository() {
		t.Skip("Not running in a git repository")
	}

	branch, err := GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch() error = %v", err)
	}

	// In CI (detached HEAD), branch may be empty — that's valid
	if branch == "" {
		t.Skip("Skipping: detached HEAD (no branch name), common in CI")
	}

	// Branch name should not contain newlines
	if strings.Contains(branch, "\n") {
		t.Errorf("branch name should not contain newlines: %q", branch)
	}
}

func TestGetGitRoot_Valid(t *testing.T) {
	if !IsGitRepository() {
		t.Skip("Not running in a git repository")
	}

	root, err := GetGitRoot()
	if err != nil {
		t.Fatalf("GetGitRoot() error = %v", err)
	}

	if root == "" {
		t.Error("GetGitRoot() returned empty string")
	}

	// Should be an absolute path
	if !strings.HasPrefix(root, "/") {
		t.Errorf("GetGitRoot() should return absolute path, got %s", root)
	}

	// Should contain langfuse-go (the project name)
	if !strings.Contains(root, "langfuse-go") {
		t.Errorf("GetGitRoot() = %s, expected to contain 'langfuse-go'", root)
	}
}

func TestGetHooksDir_Valid(t *testing.T) {
	if !IsGitRepository() {
		t.Skip("Not running in a git repository")
	}

	hooksDir, err := GetHooksDir()
	if err != nil {
		t.Fatalf("GetHooksDir() error = %v", err)
	}

	if hooksDir == "" {
		t.Error("GetHooksDir() returned empty string")
	}

	if !strings.Contains(hooksDir, "hooks") {
		t.Errorf("GetHooksDir() = %s, expected to contain 'hooks'", hooksDir)
	}
}
