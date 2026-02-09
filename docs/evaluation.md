# Evaluation

Langfuse provides powerful evaluation capabilities, including built-in support for LLM-as-a-Judge workflows.

## Overview

Evaluation in Langfuse helps you:

- Assess the quality of LLM outputs
- Compare different models or prompts
- Automate quality checks with LLM-as-a-Judge
- Track evaluation metrics over time

## Basic Scoring

### Manual Scores

Add scores directly to observations:

```go
// Score a trace
trace.Score(langfuse.ScoreParams{
    Name:  "user-rating",
    Value: 5.0,
    Comment: "Excellent response",
})

// Score a generation
generation.Score(langfuse.ScoreParams{
    Name:  "accuracy",
    Value: 0.95,
})

// Score a span
span.Score(langfuse.ScoreParams{
    Name:  "latency",
    Value: 0.8,
    Comment: "Fast response",
})
```

### Score Parameters

```go
type ScoreParams struct {
    Name    string  // Score name (e.g., "quality", "accuracy")
    Value   float64 // Score value (typically 0.0 to 1.0 or 1 to 5)
    Comment string  // Optional comment explaining the score
}
```

## LLM-as-a-Judge

LLM-as-a-Judge uses an LLM to automatically evaluate outputs based on criteria you define.

### Basic LLM-as-a-Judge

```go
package main

import (
    "context"
    "log"

    "github.com/jdziat/langfuse-go/langfuse"
)

func main() {
    client, err := langfuse.New()
    if err != nil {
        log.Fatal(err)
    }
    defer client.Shutdown(context.Background())

    // Create a trace with a generation
    trace := client.Trace(langfuse.TraceParams{
        Name: "chat-completion",
    })

    generation := trace.Generation(langfuse.GenerationParams{
        Name:  "gpt-4-response",
        Model: "gpt-4",
        Input: map[string]any{
            "messages": []map[string]string{
                {"role": "user", "content": "Explain quantum computing"},
            },
        },
    })

    generation.Update(langfuse.GenerationParams{
        Output: map[string]any{
            "response": "Quantum computing uses quantum mechanics...",
        },
    })

    // Evaluate with LLM-as-a-Judge
    evaluator := client.Evaluator(langfuse.EvaluatorParams{
        Name: "quality-evaluator",
        Model: "gpt-4",
        Prompt: `Rate the quality of this response on a scale of 0.0 to 1.0.

Input: {{input}}
Output: {{output}}

Provide a score between 0.0 and 1.0.`,
    })

    score, err := evaluator.Evaluate(context.Background(), langfuse.EvaluateParams{
        ObservationID: generation.ID,
        Input: map[string]any{
            "messages": []map[string]string{
                {"role": "user", "content": "Explain quantum computing"},
            },
        },
        Output: map[string]any{
            "response": "Quantum computing uses quantum mechanics...",
        },
    })
    if err != nil {
        log.Printf("Evaluation failed: %v", err)
    }

    log.Printf("Quality score: %.2f", score.Value)
}
```

### Custom Evaluation Criteria

Define specific evaluation criteria:

```go
evaluator := client.Evaluator(langfuse.EvaluatorParams{
    Name: "custom-evaluator",
    Model: "gpt-4",
    Prompt: `Evaluate the response based on these criteria:

1. Accuracy: Is the information correct?
2. Completeness: Does it fully answer the question?
3. Clarity: Is it easy to understand?

Input Question: {{input}}
Response: {{output}}

Rate from 0.0 to 1.0 based on the criteria above.`,
})

score, err := evaluator.Evaluate(ctx, langfuse.EvaluateParams{
    ObservationID: generation.ID,
    Input:  inputData,
    Output: outputData,
})
```

### Multiple Evaluation Dimensions

Evaluate across multiple dimensions:

```go
// Accuracy evaluator
accuracyEvaluator := client.Evaluator(langfuse.EvaluatorParams{
    Name: "accuracy",
    Model: "gpt-4",
    Prompt: "Rate the factual accuracy of this response from 0.0 to 1.0...",
})

// Helpfulness evaluator
helpfulnessEvaluator := client.Evaluator(langfuse.EvaluatorParams{
    Name: "helpfulness",
    Model: "gpt-4",
    Prompt: "Rate how helpful this response is from 0.0 to 1.0...",
})

// Safety evaluator
safetyEvaluator := client.Evaluator(langfuse.EvaluatorParams{
    Name: "safety",
    Model: "gpt-4",
    Prompt: "Rate the safety of this response from 0.0 to 1.0...",
})

// Evaluate on all dimensions
accuracyScore, _ := accuracyEvaluator.Evaluate(ctx, params)
helpfulnessScore, _ := helpfulnessEvaluator.Evaluate(ctx, params)
safetyScore, _ := safetyEvaluator.Evaluate(ctx, params)
```

## Evaluation Patterns

### Pattern 1: Inline Evaluation

Evaluate immediately after generation:

```go
// Generate response
generation := trace.Generation(langfuse.GenerationParams{
    Name:  "chat-completion",
    Model: "gpt-4",
    Input: inputData,
})

generation.Update(langfuse.GenerationParams{
    Output: responseData,
})

// Evaluate immediately
evaluator := client.Evaluator(langfuse.EvaluatorParams{
    Name: "quality",
    Model: "gpt-4",
    Prompt: "Evaluate quality...",
})

score, err := evaluator.Evaluate(ctx, langfuse.EvaluateParams{
    ObservationID: generation.ID,
    Input:  inputData,
    Output: responseData,
})

if err == nil {
    log.Printf("Quality: %.2f", score.Value)
}
```

### Pattern 2: Batch Evaluation

Evaluate multiple generations in batch:

```go
// Collect generations
generations := []struct {
    ID     string
    Input  map[string]any
    Output map[string]any
}{
    {gen1.ID, input1, output1},
    {gen2.ID, input2, output2},
    {gen3.ID, input3, output3},
}

// Batch evaluate
evaluator := client.Evaluator(langfuse.EvaluatorParams{
    Name: "quality",
    Model: "gpt-4",
    Prompt: "Rate quality...",
})

for _, gen := range generations {
    score, err := evaluator.Evaluate(ctx, langfuse.EvaluateParams{
        ObservationID: gen.ID,
        Input:  gen.Input,
        Output: gen.Output,
    })

    if err != nil {
        log.Printf("Evaluation failed for %s: %v", gen.ID, err)
        continue
    }

    log.Printf("Generation %s: quality=%.2f", gen.ID, score.Value)
}
```

### Pattern 3: Asynchronous Evaluation

Evaluate in the background without blocking:

```go
// Generate response
generation := trace.Generation(langfuse.GenerationParams{
    Name:  "chat-completion",
    Model: "gpt-4",
    Input: inputData,
})

generation.Update(langfuse.GenerationParams{
    Output: responseData,
})

// Evaluate asynchronously
go func() {
    evaluator := client.Evaluator(langfuse.EvaluatorParams{
        Name: "quality",
        Model: "gpt-4",
        Prompt: "Evaluate quality...",
    })

    score, err := evaluator.Evaluate(context.Background(), langfuse.EvaluateParams{
        ObservationID: generation.ID,
        Input:  inputData,
        Output: responseData,
    })

    if err != nil {
        log.Printf("Async evaluation failed: %v", err)
        return
    }

    log.Printf("Async evaluation complete: %.2f", score.Value)
}()

// Continue processing
```

## Evaluation Use Cases

### Use Case 1: Response Quality

Evaluate overall response quality:

```go
evaluator := client.Evaluator(langfuse.EvaluatorParams{
    Name: "response-quality",
    Model: "gpt-4",
    Prompt: `Evaluate this AI response for quality.

Question: {{input.question}}
Response: {{output.response}}

Consider:
- Accuracy of information
- Completeness of answer
- Clarity of explanation

Rate from 0.0 (poor) to 1.0 (excellent).`,
})
```

### Use Case 2: Hallucination Detection

Detect hallucinations in responses:

```go
evaluator := client.Evaluator(langfuse.EvaluatorParams{
    Name: "hallucination-detector",
    Model: "gpt-4",
    Prompt: `Detect if this response contains hallucinations.

Context: {{input.context}}
Response: {{output.response}}

Check if the response makes claims not supported by the context.

Rate from 0.0 (many hallucinations) to 1.0 (no hallucinations).`,
})
```

### Use Case 3: Tone and Style

Evaluate tone and communication style:

```go
evaluator := client.Evaluator(langfuse.EvaluatorParams{
    Name: "tone-evaluator",
    Model: "gpt-4",
    Prompt: `Evaluate the tone of this response.

Response: {{output.response}}

Is it professional, friendly, and appropriate?

Rate from 0.0 (inappropriate) to 1.0 (perfect tone).`,
})
```

### Use Case 4: Task Completion

Check if a task was completed correctly:

```go
evaluator := client.Evaluator(langfuse.EvaluatorParams{
    Name: "task-completion",
    Model: "gpt-4",
    Prompt: `Did the AI complete the requested task?

Task: {{input.task}}
Response: {{output.response}}

Rate from 0.0 (task not completed) to 1.0 (task fully completed).`,
})
```

## Evaluation Best Practices

### 1. Use Clear Evaluation Prompts

Write specific, unambiguous prompts:

```go
// Good: Specific criteria
Prompt: "Rate accuracy from 0.0 to 1.0. Consider: factual correctness, source reliability."

// Bad: Vague
Prompt: "Is this good?"
```

### 2. Define Score Ranges

Clearly define what scores mean:

```go
evaluator := client.Evaluator(langfuse.EvaluatorParams{
    Prompt: `Rate from 0.0 to 1.0 where:
- 0.0-0.3: Poor quality
- 0.4-0.6: Acceptable
- 0.7-0.9: Good
- 0.9-1.0: Excellent`,
})
```

### 3. Provide Examples

Include examples in prompts:

```go
evaluator := client.Evaluator(langfuse.EvaluatorParams{
    Prompt: `Rate clarity from 0.0 to 1.0.

Example 1.0: "The sky is blue because of Rayleigh scattering."
Example 0.3: "The sky is blue due to atmospheric phenomena."

Response: {{output.response}}`,
})
```

### 4. Handle Evaluation Errors

Always handle evaluation failures gracefully:

```go
score, err := evaluator.Evaluate(ctx, params)
if err != nil {
    log.Printf("Evaluation failed: %v", err)

    // Fall back to default score or manual review
    generation.Score(langfuse.ScoreParams{
        Name:    "quality",
        Value:   0.5,
        Comment: "Evaluation failed, needs manual review",
    })
}
```

### 5. Track Evaluation Costs

Monitor evaluation costs:

```go
generation := trace.Generation(langfuse.GenerationParams{
    Name: "evaluation-generation",
    Model: "gpt-4",
})

generation.Update(langfuse.GenerationParams{
    Usage: &langfuse.Usage{
        PromptTokens:     100,
        CompletionTokens: 10,
        TotalTokens:      110,
    },
})
```

## Complete Evaluation Example

Here's a complete example with multiple evaluation dimensions:

```go
package main

import (
    "context"
    "log"

    "github.com/jdziat/langfuse-go/langfuse"
)

func main() {
    client, err := langfuse.New()
    if err != nil {
        log.Fatal(err)
    }
    defer client.Shutdown(context.Background())

    // Create trace and generation
    trace := client.Trace(langfuse.TraceParams{
        Name: "customer-support",
        UserID: "user-123",
    })

    generation := trace.Generation(langfuse.GenerationParams{
        Name:  "support-response",
        Model: "gpt-4",
        Input: map[string]any{
            "question": "How do I reset my password?",
        },
    })

    generation.Update(langfuse.GenerationParams{
        Output: map[string]any{
            "response": "To reset your password, go to Settings > Security > Reset Password.",
        },
    })

    // Evaluate across multiple dimensions
    evaluationResults := evaluateResponse(client, generation.ID,
        "How do I reset my password?",
        "To reset your password, go to Settings > Security > Reset Password.",
    )

    // Log results
    for dimension, score := range evaluationResults {
        log.Printf("%s: %.2f", dimension, score)
    }

    client.Flush(context.Background())
}

func evaluateResponse(client *langfuse.Client, generationID string, input, output string) map[string]float64 {
    ctx := context.Background()
    results := make(map[string]float64)

    // Accuracy
    accuracyEval := client.Evaluator(langfuse.EvaluatorParams{
        Name: "accuracy",
        Model: "gpt-4",
        Prompt: "Rate the factual accuracy of this support response from 0.0 to 1.0.",
    })

    if score, err := accuracyEval.Evaluate(ctx, langfuse.EvaluateParams{
        ObservationID: generationID,
        Input:  map[string]any{"question": input},
        Output: map[string]any{"response": output},
    }); err == nil {
        results["accuracy"] = score.Value
    }

    // Helpfulness
    helpfulnessEval := client.Evaluator(langfuse.EvaluatorParams{
        Name: "helpfulness",
        Model: "gpt-4",
        Prompt: "Rate how helpful this response is from 0.0 to 1.0.",
    })

    if score, err := helpfulnessEval.Evaluate(ctx, langfuse.EvaluateParams{
        ObservationID: generationID,
        Input:  map[string]any{"question": input},
        Output: map[string]any{"response": output},
    }); err == nil {
        results["helpfulness"] = score.Value
    }

    // Clarity
    clarityEval := client.Evaluator(langfuse.EvaluatorParams{
        Name: "clarity",
        Model: "gpt-4",
        Prompt: "Rate the clarity and readability from 0.0 to 1.0.",
    })

    if score, err := clarityEval.Evaluate(ctx, langfuse.EvaluateParams{
        ObservationID: generationID,
        Input:  map[string]any{"question": input},
        Output: map[string]any{"response": output},
    }); err == nil {
        results["clarity"] = score.Value
    }

    return results
}
```

## Next Steps

- [API Reference](api-reference.md) - Complete type reference
- [Tracing Guide](tracing.md) - Learn about traces and observations
- [Configuration](configuration.md) - Customize SDK behavior
