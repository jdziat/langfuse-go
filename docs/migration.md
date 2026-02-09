# Migration Guide

This guide helps you migrate to the latest version of the Langfuse Go SDK.

## Overview

The SDK has been refactored to provide a cleaner, more type-safe API. This guide covers:

- Breaking changes
- New patterns and best practices
- Migration examples
- Troubleshooting

## Breaking Changes

### 1. Type-Safe Constructors

**Old Pattern:**
```go
// Generic constructors with error strings
generation := trace.AsGeneration(...)
span := trace.AsSpan(...)
```

**New Pattern:**
```go
// Type-specific constructors
generation := trace.Generation(langfuse.GenerationParams{...})
span := trace.Span(langfuse.SpanParams{...})
```

**Why:** Type-safe constructors prevent errors at compile time and provide better IDE support.

### 2. WithOptions Pattern Removed

**Old Pattern:**
```go
// WithOptions parameter
generation := trace.Generation(langfuse.GenerationParams{...}, langfuse.WithOptions(...))
```

**New Pattern:**
```go
// All configuration in params struct
generation := trace.Generation(langfuse.GenerationParams{
    Name: "openai-chat",
    Model: "gpt-4",
    Input: inputData,
    // ... all other options
})
```

**Why:** Simpler API with all configuration in one place.

### 3. Context-Based Operations

**Old Pattern:**
```go
// No context support
client.Flush()
client.Shutdown()
```

**New Pattern:**
```go
// Context for timeout and cancellation
ctx := context.Background()
client.Flush(ctx)
client.Shutdown(ctx)
```

**Why:** Better control over timeouts and cancellation.

## Migration Examples

### Example 1: Basic Trace Creation

**Before:**
```go
trace := client.Trace(langfuse.TraceParams{
    Name: "chat-completion",
})

generation := trace.AsGeneration(langfuse.GenerationParams{
    Name: "openai-chat",
    Model: "gpt-4",
})
```

**After:**
```go
trace := client.Trace(langfuse.TraceParams{
    Name: "chat-completion",
})

generation := trace.Generation(langfuse.GenerationParams{
    Name: "openai-chat",
    Model: "gpt-4",
})
```

### Example 2: Spans and Events

**Before:**
```go
span := trace.AsSpan(langfuse.SpanParams{
    Name: "retrieve-documents",
})

event := trace.AsEvent(langfuse.EventParams{
    Name: "cache-hit",
})
```

**After:**
```go
span := trace.Span(langfuse.SpanParams{
    Name: "retrieve-documents",
})

event := trace.Event(langfuse.EventParams{
    Name: "cache-hit",
})
```

### Example 3: Nested Observations

**Before:**
```go
span := trace.AsSpan(langfuse.SpanParams{Name: "parent"})
childSpan := span.AsSpan(langfuse.SpanParams{Name: "child"})
```

**After:**
```go
span := trace.Span(langfuse.SpanParams{Name: "parent"})
childSpan := span.Span(langfuse.SpanParams{Name: "child"})
```

### Example 4: Graceful Shutdown

**Before:**
```go
defer client.Shutdown()
```

**After:**
```go
defer client.Shutdown(context.Background())

// Or with timeout
defer func() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    client.Shutdown(ctx)
}()
```

### Example 5: Flushing Events

**Before:**
```go
client.Flush()
```

**After:**
```go
if err := client.Flush(context.Background()); err != nil {
    log.Printf("Flush failed: %v", err)
}

// Or with timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

if err := client.Flush(ctx); err != nil {
    log.Printf("Flush failed: %v", err)
}
```

## Step-by-Step Migration

### Step 1: Update Imports

Ensure you're importing the latest version:

```go
import "github.com/jdziat/langfuse-go/langfuse"
```

Update your `go.mod`:

```bash
go get -u github.com/jdziat/langfuse-go
```

### Step 2: Replace As* Methods

Find and replace all `As*` method calls:

```bash
# Find usage
grep -r "AsGeneration\|AsSpan\|AsEvent" .

# Replace in your code
AsGeneration → Generation
AsSpan → Span
AsEvent → Event
```

### Step 3: Add Context Support

Add context to `Flush` and `Shutdown` calls:

**Before:**
```go
defer client.Shutdown()
client.Flush()
```

**After:**
```go
defer client.Shutdown(context.Background())

if err := client.Flush(context.Background()); err != nil {
    log.Printf("Flush error: %v", err)
}
```

### Step 4: Remove WithOptions

Remove any `WithOptions` usage and move options into params:

**Before:**
```go
generation := trace.Generation(
    langfuse.GenerationParams{Name: "test"},
    langfuse.WithOptions(langfuse.Options{...}),
)
```

**After:**
```go
generation := trace.Generation(langfuse.GenerationParams{
    Name: "test",
    // All options here
})
```

### Step 5: Test Your Changes

Run your tests to verify the migration:

```bash
go test ./...
```

## Common Migration Issues

### Issue 1: Compile Error - "As* method not found"

**Error:**
```
trace.AsGeneration undefined (type *Trace has no field or method AsGeneration)
```

**Solution:**
Replace with type-specific constructor:
```go
generation := trace.Generation(langfuse.GenerationParams{...})
```

### Issue 2: Context Required Error

**Error:**
```
not enough arguments in call to client.Flush
```

**Solution:**
Add context parameter:
```go
client.Flush(context.Background())
```

### Issue 3: WithOptions Not Found

**Error:**
```
undefined: WithOptions
```

**Solution:**
Move all options into the params struct:
```go
generation := trace.Generation(langfuse.GenerationParams{
    Name: "test",
    Model: "gpt-4",
    // All configuration here
})
```

## Migration Checklist

Use this checklist to ensure complete migration:

- [ ] Updated to latest SDK version
- [ ] Replaced `AsGeneration` with `Generation`
- [ ] Replaced `AsSpan` with `Span`
- [ ] Replaced `AsEvent` with `Event`
- [ ] Added context to `Flush()` calls
- [ ] Added context to `Shutdown()` calls
- [ ] Removed `WithOptions` usage
- [ ] Moved options to params structs
- [ ] Added error handling for `Flush()` and `Shutdown()`
- [ ] Updated tests
- [ ] Verified in development environment
- [ ] Deployed to staging
- [ ] Monitored for issues

## Code Modernization

Beyond breaking changes, consider these improvements:

### Use Meaningful Names

```go
// Good
trace := client.Trace(langfuse.TraceParams{
    Name: "user-chat-completion",
    UserID: "user-123",
})

// Avoid
trace := client.Trace(langfuse.TraceParams{
    Name: "trace1",
})
```

### Add Rich Metadata

```go
generation := trace.Generation(langfuse.GenerationParams{
    Name: "openai-chat",
    Model: "gpt-4",
    Metadata: map[string]any{
        "environment": "production",
        "version": "1.2.3",
        "feature_flags": map[string]bool{
            "new_model": true,
        },
    },
})
```

### Proper Error Handling

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

if err := client.Flush(ctx); err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Printf("Flush timeout: %v", err)
    } else {
        log.Printf("Flush failed: %v", err)
    }
}
```

### Use Token Usage

Always include usage information for generations:

```go
generation.Update(langfuse.GenerationParams{
    Output: responseData,
    Usage: &langfuse.Usage{
        PromptTokens:     100,
        CompletionTokens: 50,
        TotalTokens:      150,
    },
})
```

## Complete Before/After Example

### Before (Old API)

```go
package main

import (
    "log"

    "github.com/jdziat/langfuse-go/langfuse"
)

func main() {
    client, err := langfuse.New(
        langfuse.WithPublicKey("pk_lf_..."),
        langfuse.WithSecretKey("sk_lf_..."),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Shutdown()

    trace := client.Trace(langfuse.TraceParams{
        Name: "chat-completion",
    })

    generation := trace.AsGeneration(langfuse.GenerationParams{
        Name: "openai-chat",
        Model: "gpt-4",
    })

    generation.Update(langfuse.GenerationParams{
        Output: map[string]any{"response": "Hello!"},
    })

    client.Flush()
}
```

### After (New API)

```go
package main

import (
    "context"
    "log"

    "github.com/jdziat/langfuse-go/langfuse"
)

func main() {
    client, err := langfuse.New(
        langfuse.WithPublicKey("pk_lf_..."),
        langfuse.WithSecretKey("sk_lf_..."),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Shutdown(context.Background())

    trace := client.Trace(langfuse.TraceParams{
        Name: "chat-completion",
    })

    generation := trace.Generation(langfuse.GenerationParams{
        Name: "openai-chat",
        Model: "gpt-4",
    })

    generation.Update(langfuse.GenerationParams{
        Output: map[string]any{"response": "Hello!"},
        Usage: &langfuse.Usage{
            PromptTokens:     10,
            CompletionTokens: 5,
            TotalTokens:      15,
        },
    })

    if err := client.Flush(context.Background()); err != nil {
        log.Printf("Flush failed: %v", err)
    }
}
```

## Getting Help

If you encounter issues during migration:

1. **Check the documentation**: [Getting Started Guide](getting-started.md)
2. **Review examples**: [API Reference](api-reference.md)
3. **Open an issue**: [GitHub Issues](https://github.com/jdziat/langfuse-go/issues)
4. **Ask the community**: [Langfuse Discord](https://langfuse.com/discord)

## Next Steps

- [Getting Started](getting-started.md) - Learn the new API
- [Configuration](configuration.md) - Optimize your setup
- [Tracing Guide](tracing.md) - Master tracing patterns
- [API Reference](api-reference.md) - Complete type reference
