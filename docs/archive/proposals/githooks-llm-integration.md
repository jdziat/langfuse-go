# Proposal: Git Hooks with LLM Integration

## Summary

Implement git hooks that leverage LLM capabilities to automatically generate meaningful commit messages and branch names, improving developer workflow consistency and commit quality.

## Motivation

### Current Pain Points

1. **Inconsistent commit messages**: Developers often write terse or unclear commit messages under time pressure
2. **Manual branch naming**: Branch names vary in format and descriptiveness
3. **Changelog generation friction**: Poor commit messages reduce the effectiveness of automated changelog generation
4. **Onboarding overhead**: New contributors must learn commit conventions

### Goals

- Automate generation of conventional commit messages from staged changes
- Suggest descriptive branch names based on issue context or initial commits
- Maintain compatibility with existing CI/CD and GoReleaser configuration
- Keep the solution optional and non-intrusive

## Proposed Solution

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      Git Hooks Layer                         │
├─────────────────────────────────────────────────────────────┤
│  prepare-commit-msg  │  commit-msg  │  post-checkout        │
└──────────┬───────────┴──────┬───────┴──────────┬────────────┘
           │                  │                  │
           ▼                  ▼                  ▼
┌─────────────────────────────────────────────────────────────┐
│                    Hook Scripts (Bash/Go)                    │
├─────────────────────────────────────────────────────────────┤
│  • Collect git context (diff, staged files, branch)         │
│  • Format prompt with repository context                     │
│  • Call LLM provider API                                     │
│  • Parse and validate response                               │
│  • Apply or suggest changes                                  │
└──────────┬──────────────────────────────────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────────┐
│                    LLM Provider Layer                        │
├─────────────────────────────────────────────────────────────┤
│  Supported:                                                  │
│  • OpenAI (GPT-4, GPT-4o)                                   │
│  • Anthropic (Claude)                                        │
│  • Local models (Ollama)                                     │
│  • Custom endpoints                                          │
└─────────────────────────────────────────────────────────────┘
```

### Git Hooks Implementation

#### 1. prepare-commit-msg Hook

**Purpose**: Generate commit message draft from staged changes

**Trigger**: Before commit message editor opens

**Behavior**:
```bash
#!/bin/bash
# .git/hooks/prepare-commit-msg

COMMIT_MSG_FILE=$1
COMMIT_SOURCE=$2

# Skip if message already provided (-m flag) or amend/merge
if [ -n "$COMMIT_SOURCE" ]; then
    exit 0
fi

# Generate commit message via LLM
generated_msg=$(langfuse-hooks generate-commit)

if [ $? -eq 0 ] && [ -n "$generated_msg" ]; then
    echo "$generated_msg" > "$COMMIT_MSG_FILE"
fi
```

**LLM Prompt Template**:
```
You are a commit message generator for a Go SDK project.

Repository: langfuse-go (Langfuse observability SDK)
Conventions: Conventional Commits (feat:, fix:, docs:, test:, refactor:, perf:)

Staged changes:
{git_diff_staged}

Files changed:
{staged_files_list}

Generate a commit message following these rules:
1. First line: type(scope): description (max 72 chars)
2. Blank line
3. Body explaining what and why (wrap at 72 chars)
4. Reference issues if mentioned in diff

Respond with ONLY the commit message, no explanations.
```

#### 2. commit-msg Hook

**Purpose**: Validate and optionally enhance commit messages

**Trigger**: After user writes/accepts commit message

**Behavior**:
```bash
#!/bin/bash
# .git/hooks/commit-msg

COMMIT_MSG_FILE=$1
commit_msg=$(cat "$COMMIT_MSG_FILE")

# Validate conventional commit format
if ! echo "$commit_msg" | grep -qE "^(feat|fix|docs|test|refactor|perf|chore|ci|build)(\(.+\))?: .+"; then
    echo "Warning: Commit message doesn't follow conventional commits format"

    # Optionally suggest a fix
    suggested=$(langfuse-hooks fix-commit "$commit_msg")
    if [ -n "$suggested" ]; then
        echo "Suggested: $suggested"
        read -p "Use suggestion? [y/N] " -n 1 -r
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            echo "$suggested" > "$COMMIT_MSG_FILE"
        fi
    fi
fi
```

#### 3. post-checkout Hook (Branch Name Suggestion)

**Purpose**: Suggest branch name when creating new branches

**Trigger**: After `git checkout -b` or `git switch -c`

**Behavior**:
```bash
#!/bin/bash
# .git/hooks/post-checkout

PREV_HEAD=$1
NEW_HEAD=$2
BRANCH_FLAG=$3

# Only trigger for branch creation (flag=1)
if [ "$BRANCH_FLAG" != "1" ]; then
    exit 0
fi

current_branch=$(git branch --show-current)

# If branch name is generic (e.g., "feature", "fix", "temp")
if echo "$current_branch" | grep -qE "^(feature|fix|temp|test|wip)$"; then
    echo "Generic branch name detected: $current_branch"

    # Get context from recent commits or prompt user
    read -p "Describe the work for this branch: " description

    suggested=$(langfuse-hooks suggest-branch "$description")
    if [ -n "$suggested" ]; then
        echo "Suggested branch name: $suggested"
        read -p "Rename branch? [y/N] " -n 1 -r
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            git branch -m "$suggested"
            echo "Branch renamed to: $suggested"
        fi
    fi
fi
```

### CLI Tool: langfuse-hooks

A lightweight Go CLI tool for hook operations:

```go
// cmd/langfuse-hooks/main.go
package main

import (
    "fmt"
    "os"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Fprintln(os.Stderr, "Usage: langfuse-hooks <command> [args]")
        os.Exit(1)
    }

    switch os.Args[1] {
    case "generate-commit":
        generateCommitMessage()
    case "fix-commit":
        fixCommitMessage(os.Args[2])
    case "suggest-branch":
        suggestBranchName(os.Args[2])
    case "install":
        installHooks()
    case "uninstall":
        uninstallHooks()
    default:
        fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
        os.Exit(1)
    }
}
```

### Configuration

#### Config File: `.langfuse-hooks.yaml`

```yaml
# LLM Provider Configuration
provider: openai  # openai | anthropic | ollama | custom

openai:
  api_key: ${OPENAI_API_KEY}  # Environment variable reference
  model: gpt-4o
  max_tokens: 500

anthropic:
  api_key: ${ANTHROPIC_API_KEY}
  model: claude-3-5-sonnet-20241022
  max_tokens: 500

ollama:
  endpoint: http://localhost:11434
  model: llama3.2

custom:
  endpoint: https://api.example.com/v1/chat/completions
  api_key: ${CUSTOM_API_KEY}
  model: custom-model

# Hook Behavior
hooks:
  prepare-commit-msg:
    enabled: true
    interactive: true  # Show generated message for approval
    include_diff: true
    max_diff_lines: 500  # Truncate large diffs

  commit-msg:
    enabled: true
    validate_format: true
    auto_fix: false  # Require user confirmation

  branch-suggest:
    enabled: true
    format: "{type}/{ticket}-{description}"  # e.g., feat/LANG-123-add-metrics
    max_length: 50

# Project Context
context:
  conventions: |
    - Use conventional commits: feat, fix, docs, test, refactor, perf, chore
    - Scope examples: client, ingestion, traces, scores, prompts
    - Reference GitHub issues when applicable

  ignore_patterns:
    - "*.generated.go"
    - "vendor/*"
```

#### Environment Variables

```bash
# Required for LLM access
export OPENAI_API_KEY="sk-..."
# or
export ANTHROPIC_API_KEY="sk-ant-..."

# Optional overrides
export LANGFUSE_HOOKS_PROVIDER="anthropic"
export LANGFUSE_HOOKS_MODEL="claude-3-5-sonnet-20241022"
export LANGFUSE_HOOKS_DISABLED="true"  # Temporarily disable
```

### Branch Name Generation

#### Format Options

```
Type-based:
  feat/add-metrics-endpoint
  fix/resolve-nil-pointer-traces
  docs/update-readme-examples

Ticket-based (when issue number provided):
  feat/LANG-123-add-metrics
  fix/GH-456-nil-pointer

Date-based:
  feat/2024-01-add-metrics
```

#### LLM Prompt Template

```
Generate a git branch name for the following work:

Description: {user_description}

Rules:
1. Format: {type}/{description}
2. Types: feat, fix, docs, test, refactor, perf, chore
3. Use lowercase with hyphens
4. Max length: 50 characters
5. Be descriptive but concise

Examples:
- feat/add-prometheus-metrics
- fix/handle-nil-trace-context
- docs/improve-api-examples

Respond with ONLY the branch name, no explanations.
```

### Installation

#### One-time Setup

```bash
# Install the hooks CLI tool
go install github.com/jdziat/langfuse-go/cmd/langfuse-hooks@latest

# Install hooks in current repository
langfuse-hooks install

# Or manually copy hooks
cp .githooks/* .git/hooks/
chmod +x .git/hooks/*
```

#### Team Setup (via Makefile)

```makefile
.PHONY: hooks-install hooks-uninstall

hooks-install:
	@echo "Installing git hooks..."
	@langfuse-hooks install
	@echo "Done. Configure .langfuse-hooks.yaml and set API keys."

hooks-uninstall:
	@echo "Removing git hooks..."
	@langfuse-hooks uninstall
```

### Security Considerations

1. **API Key Protection**
   - Never commit API keys to repository
   - Use environment variables or secure credential stores
   - Support `.env` file loading (gitignored)

2. **Diff Content**
   - Truncate large diffs to prevent token limit issues
   - Filter sensitive patterns (passwords, keys) before sending
   - Option to exclude specific files from LLM context

3. **Network Failures**
   - Hooks should fail gracefully (allow commit to proceed)
   - Timeout after configurable duration (default: 10s)
   - Cache responses for repeated operations

4. **Local-First Option**
   - Support Ollama for fully local operation
   - No data leaves the machine when using local models

### Implementation Plan

#### Phase 1: Core Infrastructure

- [ ] Create `cmd/langfuse-hooks` CLI tool
- [ ] Implement configuration loading (YAML + env vars)
- [ ] Add LLM provider abstraction layer
- [ ] Implement OpenAI provider
- [ ] Add `install` and `uninstall` commands

#### Phase 2: Commit Message Generation

- [ ] Implement `generate-commit` command
- [ ] Create git diff collection utilities
- [ ] Build prompt templates with context injection
- [ ] Add `fix-commit` for validation failures
- [ ] Write tests with mock LLM responses

#### Phase 3: Branch Name Suggestion

- [ ] Implement `suggest-branch` command
- [ ] Add format string parsing
- [ ] Integrate with post-checkout hook
- [ ] Support ticket/issue number extraction

#### Phase 4: Additional Providers

- [ ] Add Anthropic Claude provider
- [ ] Add Ollama local provider
- [ ] Add custom endpoint support
- [ ] Document provider-specific setup

#### Phase 5: Polish

- [ ] Add `--dry-run` mode for all commands
- [ ] Implement response caching
- [ ] Add telemetry (opt-in) via Langfuse SDK
- [ ] Create installation documentation
- [ ] Add CI integration examples

### Alternatives Considered

#### 1. Shell Scripts Only (No Go CLI)

**Pros**: Simpler, no build step
**Cons**: Harder to test, less portable, complex config parsing

#### 2. Pre-built Tools (commitizen, lefthook)

**Pros**: Mature ecosystems
**Cons**: Additional dependencies, less customization, no LLM integration

#### 3. IDE Extensions Only

**Pros**: Rich UI, better UX
**Cons**: IDE-specific, doesn't work in terminal workflows

### Success Metrics

1. **Adoption**: % of team using hooks after 1 month
2. **Commit Quality**: Reduction in commit message edits post-merge
3. **Consistency**: % of commits following conventional format
4. **Developer Satisfaction**: Survey feedback on workflow improvement

### Open Questions

1. Should the CLI tool live in this repo or a separate repository?
2. What's the preferred default LLM provider for open-source users?
3. Should we integrate with GitHub Copilot CLI as an alternative?
4. How do we handle rate limiting for high-volume committers?

## Appendix

### Example Generated Commit Messages

**Input (staged diff)**:
```diff
diff --git a/ingestion.go b/ingestion.go
@@ -45,6 +45,15 @@ type SpanBuilder struct {
+func (s *SpanBuilder) Metadata(metadata map[string]any) *SpanBuilder {
+    s.event.Metadata = metadata
+    return s
+}
```

**Generated Output**:
```
feat(ingestion): add Metadata method to SpanBuilder

Add fluent Metadata() method to SpanBuilder allowing users to attach
arbitrary key-value metadata to spans. Follows existing builder pattern
with method chaining support.
```

### Example Branch Name Suggestions

| Input Description | Suggested Branch |
|-------------------|------------------|
| "add retry logic to http client" | `feat/add-http-retry-logic` |
| "fix nil pointer when trace is empty" | `fix/nil-pointer-empty-trace` |
| "update readme with new examples" | `docs/update-readme-examples` |
| "LANG-456 implement score batching" | `feat/LANG-456-score-batching` |
