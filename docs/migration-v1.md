# Langfuse Go v1 Migration Guide

This guide helps you migrate from the v0 API to the simplified v1 API. The new API provides a more consistent, intuitive interface while maintaining all existing functionality.

## Table of Contents

1. [Quick Start](#quick-start)
2. [API Changes Overview](#api-changes-overview)
3. [Migration Patterns](#migration-patterns)
4. [Step-by-Step Migration](#step-by-step-migration)
5. [Compatibility Layer](#compatibility-layer)
6. [Automated Migration](#automated-migration)
7. [Troubleshooting](#troubleshooting)

## Quick Start

### Before (v0 API)
```go
// Create client with complex configuration
client, err := langfuse.NewWithConfig(&langfuse.Config{
    PublicKey: "pk-xxx",
    SecretKey: "sk-xxx",
    Region:    langfuse.RegionUS,
    BatchSize: 100,
    Debug:     true,
})

// Create trace with builder pattern
trace, err := client.NewTrace().
    Name("user-request").
    UserID("user-123").
    Tags([]string{"api", "v1"}).
    Create(ctx)

// Update trace with separate Apply()
err = trace.Update().
    Output(responseData).
    Apply(ctx)

// Create generation with separate methods
generation, err := trace.Generation().
    Name("gpt-4").
    Model("gpt-4").
    Create(ctx)
err = generation.EndWithUsage(ctx, "Hello!", 10, 8)
```

### After (v1 API)
```go
// Create client with sensible defaults
client := langfuse.NewClient("pk-xxx", "sk-xxx",
    langfuse.WithRegion(langfuse.RegionUS),
    langfuse.WithDebug(true),
)

// Create trace with unified pattern
trace, err := client.NewTrace(ctx, "user-request",
    langfuse.WithUserID("user-123"),
    langfuse.WithTags([]string{"api", "v1"}),
)

// Update trace consistently
trace, err = trace.Update(ctx,
    langfuse.WithOutput(responseData),
)

// Create generation with unified pattern
generation, err := trace.NewGeneration(ctx, "gpt-4",
    langfuse.WithModel("gpt-4"),
)
generation, err = generation.End(ctx,
    langfuse.WithEndOutput("Hello!"),
    langfuse.WithTokenUsage(10, 8),
)
```

## API Changes Overview

### 1. Client Creation

| Before | After |
|--------|-------|
| `NewWithConfig(&Config{...})` | `NewClient(pubKey, secKey, opts...)` |
| 25+ config fields | Essential fields + functional options |
| Manual config validation | Automatic validation with defaults |

### 2. Entity Creation

| Before | After |
|--------|-------|
| `client.NewTrace().Name("test").Create(ctx)` | `client.NewTrace(ctx, "test", opts...)` |
| `trace.Span().Name("op").Create(ctx)` | `trace.NewSpan(ctx, "op", opts...)` |
| `generation.Score().Name("q").Value(0.9).Create(ctx)` | `generation.Score(ctx, "quality", 0.9)` |

### 3. Context Handling

| Before | After |
|--------|-------|
| Context at end of chain | Context always first parameter |
| Variable parameter positions | Consistent: `ctx, name, options...` |

### 4. Return Patterns

| Before | After |
|--------|-------|
| Sometimes `(T, error)`, sometimes `(error)` | Always `(T, error)` |
| Separate `Apply()` methods | Single `Update()` method |

## Migration Patterns

### 1. Trace Creation

**Before:**
```go
trace, err := client.NewTrace().
    Name("user-request").
    UserID("user-123").
    Tags([]string{"api", "v1"}).
    Metadata(map[string]interface{}{
        "endpoint": "/api/chat",
    }).
    Create(ctx)
```

**After:**
```go
trace, err := client.NewTrace(ctx, "user-request",
    langfuse.WithUserID("user-123"),
    langfuse.WithTags([]string{"api", "v1"}),
    langfuse.WithMetadata(map[string]interface{}{
        "endpoint": "/api/chat",
    }),
)
```

### 2. Trace Updates

**Before:**
```go
err = trace.Update().
    Output(responseData).
    Tags([]string{"completed"}).
    Apply(ctx)
```

**After:**
```go
trace, err = trace.Update(ctx,
    langfuse.WithOutput(responseData),
    langfuse.WithTags([]string{"completed"}),
)
```

### 3. Nested Observations

**Before:**
```go
span, err := trace.Span().Name("process").Create(ctx)
defer span.End(ctx)

generation, err := span.Generation().
    Name("gpt-4").
    Model("gpt-4").
    Create(ctx)
err = generation.EndWithUsage(ctx, "Hello!", 10, 8)

event, err := span.Event().
    Name("cache-hit").
    Input("user:123").
    Create(ctx)
```

**After:**
```go
span, err := trace.NewSpan(ctx, "process")
defer span.End(ctx)

generation, err := span.NewGeneration(ctx, "gpt-4",
    langfuse.WithModel("gpt-4"),
)
generation, err = generation.End(ctx,
    langfuse.WithEndOutput("Hello!"),
    langfuse.WithTokenUsage(10, 8),
)

err = span.NewEvent(ctx, "cache-hit",
    langfuse.WithEventInput("user:123"),
)
```

### 4. Score Creation

**Before:**
```go
err = generation.Score().
    Name("quality").
    NumericValue(0.95).
    Comment("Excellent response").
    Create(ctx)

// Alternative
err = trace.Score().
    Name("speed").
    NumericValue(0.8).
    Create(ctx)
```

**After:**
```go
err = generation.Score(ctx, "quality", 0.95,
    langfuse.WithScoreComment("Excellent response"),
)

err = trace.Score(ctx, "speed", 0.8)

// Or add existing score struct
score := &langfuse.Score{
    Name:     "accuracy",
    Value:    0.92,
    DataType: langfuse.ScoreDataTypeNumeric,
    Comment:  "High accuracy",
}
err = generation.AddScore(ctx, score)
```

## Step-by-Step Migration

### Step 1: Update Client Creation

Replace `NewWithConfig` with `NewClient`:

```go
// OLD
client, err := langfuse.NewWithConfig(&langfuse.Config{
    PublicKey: "pk-xxx",
    SecretKey: "sk-xxx",
    Region:    langfuse.RegionUS,
    BatchSize: 100,
    Debug:     true,
})

// NEW
client := langfuse.NewClient("pk-xxx", "sk-xxx",
    langfuse.WithRegion(langfuse.RegionUS),
    langfuse.WithDebug(true),
    // BatchSize and other options have sensible defaults
)
```

### Step 2: Update Entity Creation Patterns

Replace builder pattern with unified pattern:

```go
// OLD - Trace
trace, err := client.NewTrace().
    Name("user-request").
    UserID("user-123").
    Tags([]string{"api"}).
    Create(ctx)

// NEW - Trace
trace, err := client.NewTrace(ctx, "user-request",
    langfuse.WithUserID("user-123"),
    langfuse.WithTags([]string{"api"}),
)
```

```go
// OLD - Generation
generation, err := trace.Generation().
    Name("gpt-4").
    Model("gpt-4").
    Input(messages).
    Create(ctx)
err = generation.EndWithUsage(ctx, response, inputTokens, outputTokens)

// NEW - Generation
generation, err := trace.NewGeneration(ctx, "gpt-4",
    langfuse.WithModel("gpt-4"),
    langfuse.WithGenerationInput(messages),
)
generation, err = generation.End(ctx,
    langfuse.WithEndOutput(response),
    langfuse.WithTokenUsage(inputTokens, outputTokens),
)
```

### Step 3: Update Update/Apply Calls

Replace separate `Apply()` methods with `Update()`:

```go
// OLD
err = trace.Update().
    Output(data).
    Tags([]string{"completed"}).
    Apply(ctx)

// NEW
trace, err = trace.Update(ctx,
    langfuse.WithOutput(data),
    langfuse.WithTags([]string{"completed"}),
)
```

### Step 4: Update Score Creation

Use simplified score methods:

```go
// OLD
err = generation.Score().
    Name("quality").
    NumericValue(0.9).
    Create(ctx)

// NEW
err = generation.Score(ctx, "quality", 0.9)
```

### Step 5: Update Context Usage

Ensure context is first parameter:

```go
// OLD - Various patterns
trace.Update().Apply(ctx)
generation.EndWithUsage(ctx, text, input, output)
client.Prompts().Get(ctx, "prompt-name")

// NEW - Consistent pattern
trace.Update(ctx, opts...)
generation.End(ctx, langfuse.WithEndOutput(text), langfuse.WithTokenUsage(input, output))
client.Prompts().Get(ctx, "prompt-name", opts...)
```

## Compatibility Layer

For gradual migration, use the legacy package:

```go
import "github.com/jdziat/langfuse-go/legacy"

// Convert old config to new client
oldConfig := &langfuse.Config{
    PublicKey: "pk-xxx",
    SecretKey: "sk-xxx",
    Region:    langfuse.RegionUS,
    Debug:     true,
}

// Migrate to v1 client
client, err := legacy.CreateMigratedClient(oldConfig)
if err != nil {
    log.Fatal(err)
}

// Or use migration helpers to manually convert
opts := legacy.MigrateConfig(oldConfig)
client := langfuse.NewClient(oldConfig.PublicKey, oldConfig.SecretKey, opts...)
```

## Automated Migration

### Migration Tool Script

Create a migration script to automate common patterns:

```bash
#!/bin/bash
# migrate_langfuse.sh

echo "Migrating Langfuse Go v0 to v1..."

# 1. Replace client creation
find . -name "*.go" -type f -exec sed -i \
  's/langfuse\.NewWithConfig(\([^)]*\))/langfuse.NewClient(\1.PublicKey, \1.SecretKey)/g' {} \;

# 2. Replace trace creation patterns
find . -name "*.go" -type f -exec sed -i \
  's/\.NewTrace()\.\([^)]*)\)\.Create(ctx)/.NewTrace(ctx, \1/g' {} \;

# 3. Replace update patterns
find . -name "*.go" -type f -exec sed -i \
  's/\.Update()\.\([^)]*)\)\.Apply(ctx)/.Update(ctx, \1/g' {} \;

# 4. Replace score patterns
find . -name "*.go" -type f -exec sed -i \
  's/\.Score()\.\([^)]*)\)\.Create(ctx)/.Score(ctx, \1.Name, \1.Value)/g' {} \;

echo "Migration complete. Please review changes manually."
```

### Using the Linter

Add to your GoMakefile:

```makefile
migrate-langfuse:
    @echo "Migrating to Langfuse v1 API..."
    go run cmd/migrator/main.go ./...
    goimports -w .
    go vet ./...
    go test ./...

.PHONY: migrate-langfuse
```

## Troubleshooting

### Common Issues

#### Issue: Compilation errors after migration

**Solution:** Check that all method calls now return `(T, error)`:

```go
// WRONG - forgetting to handle error
trace.Update(ctx, opts...)

// CORRECT - handle the error
trace, err = trace.Update(ctx, opts...)
if err != nil {
    return err
}
```

#### Issue: Context parameter position

**Solution:** Context is always the first parameter now:

```go
// WRONG
trace.NewSpan(ctx, "name", ctx)  // Duplicate context

// CORRECT
trace.NewSpan(ctx, "name", opts...)
```

#### Issue: Missing With prefixes

**Solution:** All options now use With prefixes:

```go
// WRONG
trace.NewTrace(ctx, "name", UserID("user123"))  // Missing With

// CORRECT
trace.NewTrace(ctx, "name", langfuse.WithUserID("user123"))
```

### Migration Checklist

- [ ] Update client creation to use `NewClient`
- [ ] Replace all builder patterns with unified creation
- [ ] Update all `Update().Apply()` calls to `Update()`
- [ ] Ensure context is first parameter everywhere
- [ ] Update score creation to use simplified methods
- [ ] Add proper error handling for new return patterns
- [ ] Test migration with existing functionality
- [ ] Update documentation and examples

### Getting Help

If you encounter issues during migration:

1. Check [the migration examples](#migration-patterns)
2. Review the [test files](./api_v1_test.go) for correct usage
3. Enable debug logging to see deprecation warnings
4. Create an issue with your problematic code snippet

## Performance Impact

The v1 API is designed to have **zero performance impact**:

- Same underlying implementation
- No additional allocations
- Identical batching and network behavior
- Better ergonomics without performance cost

## Future Compatibility

- Legacy methods will be deprecated in v1.2
- Legacy methods will be removed in v2.0
- Migration tools and warnings help smooth transition
- Backward compatibility maintained through v1.x releases

---

*Happy migrating! The new API should make your code more readable and maintainable.*