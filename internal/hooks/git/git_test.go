package git

import (
	"strings"
	"testing"
)

func TestTruncateDiff(t *testing.T) {
	tests := []struct {
		name     string
		diff     string
		maxLines int
		wantLen  int
	}{
		{
			name:     "no truncation needed",
			diff:     "line1\nline2\nline3",
			maxLines: 5,
			wantLen:  3,
		},
		{
			name:     "truncation needed",
			diff:     "line1\nline2\nline3\nline4\nline5",
			maxLines: 3,
			wantLen:  3, // 3 lines + truncation message
		},
		{
			name:     "zero max lines",
			diff:     "line1\nline2\nline3",
			maxLines: 0,
			wantLen:  3,
		},
		{
			name:     "negative max lines",
			diff:     "line1\nline2\nline3",
			maxLines: -1,
			wantLen:  3,
		},
		{
			name:     "empty diff",
			diff:     "",
			maxLines: 5,
			wantLen:  1, // empty string splits to 1 element
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateDiff(tt.diff, tt.maxLines)

			if tt.maxLines <= 0 {
				// Should return unchanged
				if result != tt.diff {
					t.Errorf("TruncateDiff() should return unchanged diff when maxLines <= 0")
				}
				return
			}

			originalLines := strings.Split(tt.diff, "\n")

			if len(originalLines) > tt.maxLines {
				// Should be truncated
				if !strings.Contains(result, "truncated") {
					t.Errorf("TruncateDiff() should contain truncation message")
				}
			} else {
				// Should be unchanged
				if result != tt.diff {
					t.Errorf("TruncateDiff() should return unchanged diff when no truncation needed")
				}
			}
		})
	}
}

func TestIsGitRepository(t *testing.T) {
	// This test assumes we're running from within the langfuse-go repo
	if !IsGitRepository() {
		t.Skip("Not running in a git repository")
	}

	// If we get here, IsGitRepository returned true
	t.Log("Confirmed running in a git repository")
}

func TestGetCurrentBranch(t *testing.T) {
	if !IsGitRepository() {
		t.Skip("Not running in a git repository")
	}

	branch, err := GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch() error = %v", err)
	}

	if branch == "" {
		t.Error("GetCurrentBranch() returned empty string")
	}

	t.Logf("Current branch: %s", branch)
}

func TestGetGitRoot(t *testing.T) {
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

	t.Logf("Git root: %s", root)
}

func TestGetHooksDir(t *testing.T) {
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

	t.Logf("Hooks dir: %s", hooksDir)
}
