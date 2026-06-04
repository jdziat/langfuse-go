---
title: Evaluation
weight: 4
---

Langfuse helps you assess the quality of your LLM application. The Go SDK
supports two complementary approaches:

- **Scores** — attach numeric, categorical, or boolean ratings to any
  observation (trace, generation, or span).
- **Evaluation-ready tracing** — the `evaluation` subpackage produces traces
  with typed inputs and outputs structured for RAG, Q&A, summarization, and
  classification workflows, so server-side or LLM-as-a-Judge evaluators have the
  fields they need.

## Overview

Evaluation in Langfuse helps you:

- Assess the quality of LLM outputs
- Compare different models or prompts
- Structure traces so automated evaluators can run against them
- Track evaluation metrics over time

## Scoring

### Adding Scores

Scores are created with the `NewScore()` builder available on traces and
observations. Finalize each score with `Create(ctx)`:

```go
ctx := context.Background()

// Score a trace
err := trace.NewScore().
    Name("user-rating").
    NumericValue(5.0).
    Comment("Excellent response").
    Create(ctx)
if err != nil {
    log.Printf("score failed: %v", err)
}

// Score a generation
err = generation.NewScore().
    Name("accuracy").
    NumericValue(0.95).
    Create(ctx)
if err != nil {
    log.Printf("score failed: %v", err)
}
```

### Score Value Types

Scores can be numeric, categorical, or boolean:

```go
ctx := context.Background()

// Numeric (e.g. 0.0 - 1.0 or 1 - 5)
_ = trace.NewScore().Name("quality").NumericValue(0.9).Create(ctx)

// Categorical (a label)
_ = trace.NewScore().Name("sentiment").CategoricalValue("positive").Create(ctx)

// Boolean (pass/fail)
_ = trace.NewScore().Name("passed").BooleanValue(true).Create(ctx)
```

### Score Builder Methods

```go
s := trace.NewScore()
s.Name("quality")                  // score name
s.NumericValue(0.95)               // numeric value
s.CategoricalValue("good")         // categorical value
s.BooleanValue(true)               // boolean value
s.Comment("explanation")           // optional comment
s.ObservationID("obs-id")          // attach to a specific observation
s.ConfigID("score-config-id")      // link to a score config
s.Metadata(langfuse.Metadata{})    // additional metadata
```

## Evaluation-Ready Tracing

The `evaluation` subpackage wraps the core tracing API with typed builders for
common LLM workflows. The resulting traces carry structured inputs and outputs
plus evaluation metadata.

Import the subpackage alongside the core SDK:

```text
import (
    langfuse "github.com/jdziat/langfuse-go"
    "github.com/jdziat/langfuse-go/evaluation"
)
```

### Question-Answering Workflows

`NewQATrace` builds a trace whose input is a structured `QAInput`. After
generating an answer, record it with `UpdateOutput`:

```go
package main

import (
    "context"
    "log"

    langfuse "github.com/jdziat/langfuse-go"
    "github.com/jdziat/langfuse-go/evaluation"
)

func main() {
    ctx := context.Background()

    client, err := langfuse.New("pk-lf-...", "sk-lf-...")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Shutdown(ctx)

    qa, err := evaluation.NewQATrace(client, "support-qa").
        Query("How do I reset my password?").
        GroundTruth("Go to Settings > Security > Reset Password.").
        Create(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // ... call your model ...
    answer := "To reset your password, go to Settings > Security > Reset Password."

    // Record the answer and a confidence score (0.0 - 1.0)
    if err := qa.UpdateOutput(ctx, answer, 0.9); err != nil {
        log.Printf("update failed: %v", err)
    }

    if err := client.Flush(ctx); err != nil {
        log.Fatal(err)
    }
}
```

### RAG Workflows

`NewRAGTrace` captures the query, the retrieved context chunks, and (optionally)
a ground-truth answer. Use `UpdateOutput` to record the generated answer and any
citations:

```go
package main

import (
    "context"
    "log"

    langfuse "github.com/jdziat/langfuse-go"
    "github.com/jdziat/langfuse-go/evaluation"
)

func main() {
    ctx := context.Background()

    client, err := langfuse.New("pk-lf-...", "sk-lf-...")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Shutdown(ctx)

    rag, err := evaluation.NewRAGTrace(client, "docs-rag").
        Query("What is quantum computing?").
        Context(
            "Quantum computing uses quantum mechanics.",
            "Qubits can represent superpositions of states.",
        ).
        Create(ctx)
    if err != nil {
        log.Fatal(err)
    }

    answer := "Quantum computing uses qubits to exploit superposition."

    // Record the answer and the citations that support it
    if err := rag.UpdateOutput(ctx, answer, "doc-1", "doc-2"); err != nil {
        log.Printf("update failed: %v", err)
    }

    if err := client.Flush(ctx); err != nil {
        log.Fatal(err)
    }
}
```

### Classification Workflows

`NewClassificationTrace` records the input text and candidate classes, then the
predicted class and its confidence:

```go
package main

import (
    "context"
    "log"

    langfuse "github.com/jdziat/langfuse-go"
    "github.com/jdziat/langfuse-go/evaluation"
)

func main() {
    ctx := context.Background()

    client, err := langfuse.New("pk-lf-...", "sk-lf-...")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Shutdown(ctx)

    cls, err := evaluation.NewClassificationTrace(client, "intent-classifier").
        Input("I want to cancel my subscription").
        Classes([]string{"billing", "support", "sales"}).
        GroundTruth("billing").
        Create(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Record the predicted class and confidence (0.0 - 1.0)
    if err := cls.UpdateOutput(ctx, "billing", 0.97); err != nil {
        log.Printf("update failed: %v", err)
    }

    if err := client.Flush(ctx); err != nil {
        log.Fatal(err)
    }
}
```

## Combining Tracing and Scores

A common pattern is to trace a generation and then attach evaluation scores once
the result is known. Because every observation exposes `NewScore()`, you can
record several dimensions on the same generation:

```go
ctx := context.Background()

generation, err := trace.NewGeneration().
    Name("support-response").
    Model("gpt-4").
    Input(map[string]any{"question": "How do I reset my password?"}).
    Create(ctx)
if err != nil {
    log.Fatal(err)
}

if err := generation.EndWithUsage(ctx,
    "Go to Settings > Security > Reset Password.", 40, 18); err != nil {
    log.Printf("end failed: %v", err)
}

// Attach evaluation scores across multiple dimensions
for name, value := range map[string]float64{
    "accuracy":    0.95,
    "helpfulness": 0.90,
    "clarity":     0.92,
} {
    if err := generation.NewScore().Name(name).NumericValue(value).Create(ctx); err != nil {
        log.Printf("score %q failed: %v", name, err)
    }
}
```

## Best Practices

### 1. Score What You Can Measure

Numeric scores work well for graded quality; use categorical or boolean scores
for discrete judgments such as pass/fail checks:

```go
ctx := context.Background()

_ = trace.NewScore().Name("contains_pii").BooleanValue(false).Create(ctx)
```

### 2. Provide Ground Truth Where Available

The evaluation builders accept ground-truth values so downstream evaluators can
compute correctness:

```go
qa, _ := evaluation.NewQATrace(client, "support-qa").
    Query("How do I reset my password?").
    GroundTruth("Go to Settings > Security > Reset Password.").
    Create(context.Background())
_ = qa
```

### 3. Handle Errors Gracefully

Always check the error returned from `Create(ctx)`, `Apply(ctx)`, and
`UpdateOutput(ctx, ...)`:

```go
ctx := context.Background()

if err := generation.NewScore().Name("quality").NumericValue(0.5).Create(ctx); err != nil {
    log.Printf("scoring failed, needs manual review: %v", err)
}
```

## Next Steps

- [API Reference](../api-reference/) - Complete type reference
- [Tracing Guide](../tracing/) - Learn about traces and observations
- [Configuration](../configuration/) - Customize SDK behavior
