# LLM-as-a-Judge Integration Design

## Executive Summary

This document outlines a comprehensive design for making the langfuse-go SDK "LLM-as-a-Judge aware" - enabling clients to automatically structure their traces for seamless evaluation without manual field mapping.

## Current State Analysis

### What Langfuse LLM-as-a-Judge Expects

Based on Langfuse documentation, the LLM-as-a-Judge system requires:

1. **Standard Variable Names**: Templates use `{{input}}`, `{{output}}`, `{{ground_truth}}`, `{{context}}`
2. **JSONPath Mapping**: For nested data, uses expressions like `$.choices[0].message.content`
3. **Specific Data Structures**: Different evaluators expect specific field combinations:
   - **Faithfulness**: `context`, `output`
   - **Answer Relevance**: `query`, `output`
   - **Hallucination**: `query`, `context`, `output`
   - **RAGAS**: `user_input`, `retrieved_contexts`, `response`

### Current SDK Capabilities

The SDK already has:
- ✅ Typed evaluation structures (`RAGInput`, `QAInput`, etc.)
- ✅ Specialized trace builders (`RAGTraceBuilder`, etc.)
- ✅ Validation utilities (`ValidateFor`, `ValidateDetailed`)
- ✅ EvaluatorRequirements definitions

### Gaps Identified

| Gap | Impact | Priority |
|-----|--------|----------|
| No automatic field flattening for LLM-as-a-Judge | Users must configure JSONPath mappings in UI | High |
| No evaluation-aware generations | Input/output must be manually structured | High |
| Missing evaluation metadata/tags | Traces not easily filterable for evaluation | Medium |
| No workflow-level orchestration | Complex pipelines require manual structuring | Medium |
| No provider hook auto-extraction | LLM responses not automatically parsed | Medium |

---

## Proposed Design

### 1. Evaluation-Aware Mode

Add a new evaluation mode that automatically flattens and normalizes data for LLM-as-a-Judge.

```go
// New: EvaluationMode controls how data is structured for evaluation
type EvaluationMode string

const (
    // EvaluationModeOff - Standard mode, no special handling
    EvaluationModeOff EvaluationMode = ""

    // EvaluationModeAuto - Automatically structure data for evaluation
    EvaluationModeAuto EvaluationMode = "auto"

    // EvaluationModeRAGAS - Structure data specifically for RAGAS metrics
    EvaluationModeRAGAS EvaluationMode = "ragas"

    // EvaluationModeLangfuse - Structure for Langfuse managed evaluators
    EvaluationModeLangfuse EvaluationMode = "langfuse"
)

// Client configuration
client, _ := langfuse.New(publicKey, secretKey,
    langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
)
```

### 2. Flattened Input/Output Structure

When evaluation mode is active, automatically flatten structured data into top-level fields that LLM-as-a-Judge can directly access without JSONPath.

```go
// Current (requires JSONPath mapping in Langfuse UI):
trace.Input(&RAGInput{
    Query: "What is Go?",
    Context: []string{"Go is a language..."},
})
// Results in: {"query": "...", "context": [...]}

// Proposed (evaluation-aware, direct field access):
trace.Input(&RAGInput{
    Query: "What is Go?",
    Context: []string{"Go is a language..."},
})
// Results in (with EvaluationModeAuto):
// {
//   "query": "What is Go?",
//   "context": ["Go is a language..."],
//   "_langfuse_eval": {
//     "type": "rag",
//     "ready": true,
//     "compatible_evaluators": ["faithfulness", "context_relevance", "hallucination"]
//   }
// }
```

### 3. EvaluationBuilder - High-Level Workflow API

Create a new high-level API that guides users through evaluation-ready trace creation:

```go
// New: EvaluationBuilder provides guided workflow creation
type EvaluationBuilder struct {
    client *Client
    workflow WorkflowType
}

type WorkflowType string

const (
    WorkflowRAG            WorkflowType = "rag"
    WorkflowQA             WorkflowType = "qa"
    WorkflowChatCompletion WorkflowType = "chat"
    WorkflowAgentTask      WorkflowType = "agent"
    WorkflowChainOfThought WorkflowType = "cot"
    WorkflowSummarization  WorkflowType = "summarization"
    WorkflowClassification WorkflowType = "classification"
)

// Usage example:
eval := client.Evaluation(langfuse.WorkflowRAG).
    Name("document-qa").
    UserID("user-123")

// Step 1: Define the user query
eval.WithQuery("What are Go's concurrency features?")

// Step 2: Record retrieval (automatically creates retrieval span)
retrieval := eval.Retrieval()
docs := vectorDB.Search(query)
retrieval.Context(docs...).End(ctx)

// Step 3: Record generation (automatically creates generation with correct structure)
gen := eval.Generation().
    Model("gpt-4").
    Messages(messages)

response, _ := openai.CreateCompletion(...)
gen.Complete(ctx, response, usage)

// Step 4: Optional - Set ground truth for evaluation
eval.WithGroundTruth("Go uses goroutines and channels")

// Step 5: Finalize - validates completeness and submits
trace, err := eval.Complete(ctx)

// Automatic validation
if err := trace.ValidateFor(langfuse.EvaluatorFaithfulness); err != nil {
    log.Printf("Missing fields: %v", err)
}
```

### 4. Automatic Metadata and Tagging

Add automatic metadata that enables filtering and evaluation routing:

```go
// Automatic metadata added to evaluation-ready traces:
{
    "_langfuse_eval_metadata": {
        "version": "1.0",
        "workflow_type": "rag",
        "has_ground_truth": true,
        "has_context": true,
        "has_output": true,
        "compatible_evaluators": [
            "faithfulness",
            "answer_relevance",
            "context_precision",
            "hallucination"
        ],
        "ready_at": "2024-01-15T10:30:00Z"
    }
}

// Automatic tags:
["eval:ready", "eval:rag", "eval:has-ground-truth"]
```

### 5. Generation-Level Evaluation Support

Enable evaluation at the generation/observation level, not just trace level:

```go
// Evaluation-aware generation
gen, _ := trace.NewEvalGeneration().
    Name("llm-response").
    Model("gpt-4").
    ForEvaluator(langfuse.EvaluatorFaithfulness).
    WithContext(retrievedChunks...).      // Context for faithfulness check
    WithSystemPrompt(systemPrompt).        // Captured for analysis
    WithUserMessage(userQuery).            // The user's question
    Create(ctx)

// Complete with structured output
gen.CompleteWithEvaluation(ctx, &EvalGenerationResult{
    Output:           response.Choices[0].Message.Content,
    InputTokens:      response.Usage.PromptTokens,
    OutputTokens:     response.Usage.CompletionTokens,
    Model:            response.Model,
    CompletionTime:   completionStartTime,
    Confidence:       extractConfidence(response),
    Citations:        extractCitations(response),
})
```

### 6. Provider Hooks with Auto-Extraction

Enhanced hooks that automatically extract evaluation-relevant data:

```go
// New: EvaluationHook wraps provider hooks to extract eval data
type EvaluationHook struct {
    provider    ProviderHook      // OpenAI, Anthropic, etc.
    evalMode    EvaluationMode
    extractors  []FieldExtractor
}

// Field extractors for common patterns
type FieldExtractor interface {
    ExtractFromRequest(req any) map[string]any
    ExtractFromResponse(resp any) map[string]any
}

// Built-in extractors
var (
    OpenAIExtractor = &openAIFieldExtractor{}
    AnthropicExtractor = &anthropicFieldExtractor{}
    // Extracts: messages, model, temperature, max_tokens, response content, usage
)

// Usage:
client, _ := langfuse.New(publicKey, secretKey,
    langfuse.WithEvaluationHook(
        langfuse.NewOpenAIEvaluationHook(openaiClient),
    ),
)

// Now all OpenAI calls are automatically structured for evaluation
resp, _ := openaiClient.CreateChatCompletion(ctx, req)
// Langfuse automatically captures:
// - System prompt
// - User messages
// - Assistant response
// - Model parameters
// - Token usage
// All in evaluation-ready format
```

### 7. Workflow Templates

Pre-built workflow templates for common patterns:

```go
// RAG Pipeline Template
type RAGPipeline struct {
    trace      *EvaluationTrace
    retrieval  *RetrievalSpan
    generation *EvalGeneration
}

func (c *Client) NewRAGPipeline(ctx context.Context, name string) *RAGPipeline {
    return &RAGPipeline{
        trace: c.Evaluation(WorkflowRAG).Name(name),
    }
}

// Usage - completely guided workflow
pipeline := client.NewRAGPipeline(ctx, "document-qa")

// Record query
pipeline.Query("What is dependency injection?")

// Record retrieval step - creates span with correct structure
docs, _ := pipeline.Retrieve(ctx, func() ([]string, error) {
    return vectorDB.Search(query)
})

// Record generation - creates generation with correct structure
response, _ := pipeline.Generate(ctx, func(prompt string) (string, Usage, error) {
    resp, _ := openai.Complete(prompt)
    return resp.Content, Usage{Input: resp.InputTokens, Output: resp.OutputTokens}, nil
})

// Optional: Set ground truth
pipeline.GroundTruth("DI is a design pattern...")

// Complete and validate
trace, _ := pipeline.Complete(ctx)
```

### 8. Evaluation Score Integration

Seamlessly integrate with Langfuse scores:

```go
// After LLM-as-a-Judge runs, scores can be accessed
scores, _ := client.GetScores(ctx, trace.ID())

for _, score := range scores {
    fmt.Printf("%s: %.2f (%s)\n", score.Name, score.Value, score.Comment)
}

// Or set scores programmatically with proper source
trace.Score(ctx, "faithfulness", 0.95,
    langfuse.WithScoreSource(langfuse.ScoreSourceEval),
    langfuse.WithScoreComment("LLM-as-a-Judge: No hallucinations detected"),
)
```

---

## Implementation Plan

### Phase 1: Core Evaluation Mode (Foundation)

**Files to modify/create:**
- `evaluation_mode.go` - EvaluationMode types and configuration
- `evaluation_input.go` - Input flattening and normalization
- `evaluation_metadata.go` - Automatic metadata/tags
- `config.go` - Add WithEvaluationMode option

**Key changes:**
1. Add `EvaluationMode` to client configuration
2. Implement input/output flattening in ingestion layer
3. Add automatic metadata enrichment
4. Add automatic tagging based on workflow type

### Phase 2: Evaluation-Aware Observations

**Files to modify/create:**
- `eval_generation.go` - EvalGeneration with structured capture
- `eval_span.go` - EvalSpan for retrieval/processing steps
- `generation.go` - Add `ForEvaluator()` method

**Key changes:**
1. New `EvalGeneration` type with evaluation-specific methods
2. New `EvalSpan` for retrieval steps with context capture
3. Automatic field mapping for evaluators

### Phase 3: Workflow Builders

**Files to modify/create:**
- `evaluation/workflow.go` - Base workflow interface
- `evaluation/rag_workflow.go` - RAG pipeline builder
- `evaluation/qa_workflow.go` - Q&A workflow builder
- `evaluation/agent_workflow.go` - Agent task workflow

**Key changes:**
1. High-level `EvaluationBuilder` API
2. Workflow-specific builders (RAG, QA, Agent, etc.)
3. Guided step-by-step trace creation
4. Automatic validation at each step

### Phase 4: Provider Hooks Enhancement

**Files to modify/create:**
- `internal/hooks/evaluation_hook.go` - Evaluation wrapper hook
- `internal/hooks/extractors/openai.go` - OpenAI field extractor
- `internal/hooks/extractors/anthropic.go` - Anthropic extractor
- `internal/hooks/extractors/base.go` - Base extractor interface

**Key changes:**
1. Field extractor interface for providers
2. Provider-specific extractors
3. Automatic evaluation data capture from LLM calls

---

## API Examples

### Example 1: Zero-Config RAG Evaluation

```go
// Client with evaluation mode
client, _ := langfuse.New(pk, sk,
    langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
)

// Create RAG trace - automatically evaluation-ready
trace, _ := client.Trace(ctx, "rag-query",
    langfuse.WithWorkflow(langfuse.WorkflowRAG),
)

// Step 1: Set query
trace.WithQuery("What are microservices?")

// Step 2: Retrieval span (auto-structured)
retrieval, _ := trace.Retrieval(ctx)
docs := vectorDB.Search(query)
retrieval.WithContext(docs...).End(ctx)

// Step 3: Generation (auto-structured for evaluation)
gen, _ := trace.Generation(ctx, "gpt-4",
    langfuse.WithGenerationInput(prompt),
)
response := openai.Complete(prompt)
gen.EndWith(ctx,
    langfuse.WithOutput(response),
    langfuse.WithUsage(inputTokens, outputTokens),
)

// Step 4: Ground truth (optional)
trace.WithGroundTruth("Microservices are...")

// Result: Trace is automatically structured for:
// - Faithfulness evaluation
// - Answer Relevance evaluation
// - Context Precision evaluation
// - Hallucination detection
// No JSONPath configuration needed in Langfuse UI!
```

### Example 2: Agent Task Workflow

```go
agent := client.Evaluation(langfuse.WorkflowAgentTask).
    Name("research-task").
    Task("Find the latest Golang release notes")

// Tool calls are automatically captured
for _, step := range agent.Execute(ctx) {
    switch step.Type {
    case "tool_call":
        span, _ := agent.ToolCall(ctx, step.Name)
        result := executeTool(step)
        span.WithResult(result).End(ctx)
    case "reasoning":
        agent.Thought(ctx, step.Content)
    case "generation":
        gen, _ := agent.LLMCall(ctx, step.Model)
        response := callLLM(step)
        gen.Complete(ctx, response)
    }
}

trace, _ := agent.Complete(ctx)
// Structured for: Task completion, Tool usage accuracy, Reasoning quality
```

### Example 3: Auto-Instrumented OpenAI

```go
// Wrap OpenAI client with evaluation hook
evalOpenAI := langfuse.WrapOpenAI(openaiClient,
    langfuse.WithAutoEvaluation(true),
    langfuse.WithEvaluators(
        langfuse.EvaluatorToxicity,
        langfuse.EvaluatorFaithfulness,
    ),
)

// All calls are automatically traced with eval-ready structure
response, _ := evalOpenAI.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
    Model: "gpt-4",
    Messages: messages,
}, langfuse.WithTraceContext(trace))

// Generation automatically includes:
// - System prompt (for instruction following eval)
// - User messages (for relevance eval)
// - Response (for toxicity, faithfulness eval)
// - All in flat, accessible format
```

---

## Backward Compatibility

All changes are additive and backward compatible:

1. **Default behavior unchanged**: Without `WithEvaluationMode()`, SDK works exactly as before
2. **Existing evaluation package**: `evaluation.RAGTrace`, etc. continue to work
3. **Gradual adoption**: Users can adopt new features incrementally
4. **No breaking changes**: All existing APIs remain stable

---

## Success Metrics

| Metric | Current State | Target |
|--------|--------------|--------|
| Lines of code to create eval-ready RAG trace | ~30 | ~10 |
| Manual JSONPath configurations needed | 3-5 per evaluator | 0 |
| Time to first evaluation | ~30 min setup | ~5 min |
| Evaluator compatibility auto-detection | Manual | Automatic |

---

## Open Questions

1. **Standardization**: Should we follow OpenTelemetry semantic conventions for LLM tracing?
2. **Sync with Langfuse**: Should we coordinate with Langfuse team on evaluation field standards?
3. **Caching**: Should evaluation metadata be cached to avoid re-validation?
4. **Async Scores**: Should we provide a callback mechanism for when LLM-as-a-Judge scores arrive?

---

## Next Steps

1. Review and approve design
2. Implement Phase 1 (Core Evaluation Mode)
3. Add comprehensive tests
4. Update documentation
5. Iterate based on user feedback
