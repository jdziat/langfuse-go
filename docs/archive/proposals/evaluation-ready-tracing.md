# Proposal: Evaluation-Ready Tracing

**Status:** Draft
**Created:** 2025-12-25
**Author:** SDK Team

## Executive Summary

This proposal addresses a critical gap between how developers instrument their LLM applications and how Langfuse's canned evaluators expect trace data to be structured. Currently, users freely set Input/Output/Metadata fields, leading to evaluation failures when traces don't match evaluator expectations. We propose adding Go-idiomatic, type-safe helpers that guide users toward creating evaluation-ready traces while maintaining full backwards compatibility.

## Problem Statement

### Current State

The Langfuse Go SDK provides flexible, untyped interfaces for trace input/output:

```go
trace, err := client.NewTrace().
    Input(map[string]interface{}{
        "message": "What is the capital of France?",
    }).
    Output(map[string]interface{}{
        "response": "The capital of France is Paris.",
    }).
    Create()
```

### The Problem

Langfuse's canned evaluators (LLM-as-judge evaluations) expect specific field names and structures:

**RAG Evaluators** expect:
- `query`: The user's question/input
- `context`: Retrieved context chunks
- `ground_truth`: Expected correct answer (optional)
- `output`: The LLM's final response

**Q&A Evaluators** expect:
- `query`: The user's question
- `output`: The LLM's response
- `ground_truth`: Expected answer (for evaluation)

**Summarization Evaluators** expect:
- `input`: The original text to summarize
- `output`: The generated summary
- `ground_truth`: Reference summary (optional)

### Impact

When trace structure doesn't match evaluator expectations, users encounter errors like:

```
Error: The input is missing the query, context, and ground truth
Evaluation failed: Expected field 'context' not found in trace input
```

This creates a poor developer experience:
1. Trial-and-error to discover required field names
2. Inconsistent trace structures across teams
3. Debugging evaluation failures instead of focusing on application quality
4. Manual documentation hunting for each evaluator type

### Root Causes

1. **No structural guidance**: The SDK doesn't communicate evaluator requirements
2. **Lack of type safety**: Free-form maps allow any structure
3. **Missing validation**: No warnings when traces won't work with evaluators
4. **Documentation gap**: Field requirements are scattered across Langfuse docs

## Goals

1. **Make evaluation-ready tracing the easy path** - Provide helpers that naturally create compatible structures
2. **Maintain backwards compatibility** - Existing code must continue working unchanged
3. **Follow Go idioms** - Use struct types, builder patterns, and functional options
4. **Enable validation** - Allow opt-in validation and warnings for evaluator compatibility
5. **Support all evaluator types** - Cover RAG, Q&A, summarization, and custom patterns
6. **Preserve flexibility** - Don't force users into rigid structures when not needed

## Non-Goals

1. Replacing the existing flexible Input/Output API
2. Automatic field mapping or transformations
3. Client-side evaluation execution
4. Schema enforcement at runtime (validation is opt-in)

## Proposed Solutions

### Solution 1: Typed Input/Output Structs

Provide predefined structs that match evaluator expectations:

```go
package langfuse

// RAGInput represents input for RAG (Retrieval-Augmented Generation) workflows.
// This structure matches Langfuse's RAG evaluator expectations.
type RAGInput struct {
    Query       string   `json:"query"`                 // User's question or search query
    Context     []string `json:"context"`               // Retrieved context chunks
    GroundTruth string   `json:"ground_truth,omitempty"` // Optional expected answer
}

// RAGOutput represents output from a RAG workflow.
type RAGOutput struct {
    Output      string                 `json:"output"`              // Generated response
    Citations   []string               `json:"citations,omitempty"` // Source citations
    Metadata    map[string]interface{} `json:"metadata,omitempty"`  // Additional metadata
}

// QAInput represents input for question-answering workflows.
type QAInput struct {
    Query       string `json:"query"`                  // User's question
    GroundTruth string `json:"ground_truth,omitempty"` // Expected answer for evaluation
}

// QAOutput represents output from a question-answering workflow.
type QAOutput struct {
    Output     string                 `json:"output"`             // Generated answer
    Confidence float64                `json:"confidence,omitempty"` // Confidence score
    Metadata   map[string]interface{} `json:"metadata,omitempty"` // Additional metadata
}

// SummarizationInput represents input for summarization workflows.
type SummarizationInput struct {
    Input       string `json:"input"`                  // Original text to summarize
    GroundTruth string `json:"ground_truth,omitempty"` // Reference summary
    MaxLength   int    `json:"max_length,omitempty"`   // Target summary length
}

// SummarizationOutput represents output from summarization workflows.
type SummarizationOutput struct {
    Output   string                 `json:"output"`             // Generated summary
    Length   int                    `json:"length,omitempty"`   // Summary length
    Metadata map[string]interface{} `json:"metadata,omitempty"` // Additional metadata
}
```

**Usage Example:**

```go
// RAG workflow with typed inputs
ragInput := &langfuse.RAGInput{
    Query: "What are the key features of Go?",
    Context: []string{
        "Go is a statically typed, compiled programming language.",
        "Go features garbage collection and built-in concurrency.",
    },
}

trace, err := client.NewTrace().
    Name("rag-query").
    Input(ragInput).  // Type-safe, evaluator-ready
    Create()

// Later, set the output
ragOutput := &langfuse.RAGOutput{
    Output: "Go features static typing, garbage collection, and built-in concurrency support.",
    Citations: []string{"doc1.txt", "doc2.txt"},
}

err = trace.Update().
    Output(ragOutput).
    Apply()
```

**Advantages:**
- Type safety at compile time
- IDE autocomplete for field names
- Clear documentation via struct tags and comments
- Works seamlessly with existing API (Input/Output accept `interface{}`)
- Zero breaking changes

**Disadvantages:**
- Users must know which struct to use
- Doesn't prevent mixing incompatible input/output types
- Still allows using raw maps if desired

### Solution 2: Evaluation Context Builders

Add specialized builders that configure traces for specific evaluation scenarios:

```go
package langfuse

// EvaluationType represents supported evaluation scenarios
type EvaluationType string

const (
    EvaluationTypeRAG            EvaluationType = "rag"
    EvaluationTypeQA             EvaluationType = "qa"
    EvaluationTypeSummarization  EvaluationType = "summarization"
    EvaluationTypeClassification EvaluationType = "classification"
)

// WithEvaluationContext configures a trace for a specific evaluation type.
// This is a functional option for TraceBuilder.
func WithEvaluationContext(evalType EvaluationType) func(*TraceBuilder) {
    return func(tb *TraceBuilder) {
        tb.evaluationType = evalType
        if tb.trace.Metadata == nil {
            tb.trace.Metadata = make(map[string]interface{})
        }
        tb.trace.Metadata["evaluation_type"] = string(evalType)
    }
}

// RAGContext provides a fluent builder for RAG evaluation contexts
type RAGContext struct {
    builder *TraceBuilder
    query   string
    context []string
    groundTruth string
}

// WithRAGContext creates a RAG evaluation context builder
func (tb *TraceBuilder) WithRAGContext() *RAGContext {
    return &RAGContext{
        builder: tb,
    }
}

// Query sets the user's query/question
func (r *RAGContext) Query(query string) *RAGContext {
    r.query = query
    return r
}

// Context sets the retrieved context chunks
func (r *RAGContext) Context(chunks ...string) *RAGContext {
    r.context = chunks
    return r
}

// GroundTruth sets the expected correct answer (optional)
func (r *RAGContext) GroundTruth(truth string) *RAGContext {
    r.groundTruth = truth
    return r
}

// Build applies the RAG context to the trace and returns the builder
func (r *RAGContext) Build() *TraceBuilder {
    ragInput := &RAGInput{
        Query:       r.query,
        Context:     r.context,
        GroundTruth: r.groundTruth,
    }
    return r.builder.Input(ragInput)
}
```

**Usage Example:**

```go
// Fluent RAG context building
trace, err := client.NewTrace().
    Name("product-search").
    WithRAGContext().
        Query("What are the best wireless headphones?").
        Context(
            "Sony WH-1000XM5 features industry-leading noise cancellation.",
            "Bose QuietComfort 45 offers excellent comfort for long listening.",
            "Apple AirPods Max provides seamless Apple ecosystem integration.",
        ).
        GroundTruth("The best wireless headphones depend on your priorities...").
        Build().
    Create()

// Alternative: using functional options
trace, err := client.NewTrace().
    Name("product-search").
    Apply(WithEvaluationContext(langfuse.EvaluationTypeRAG)).
    Create()
```

**Advantages:**
- Discoverable API through method chaining
- Clear intent through named methods
- Prevents missing required fields
- Can add validation during Build()

**Disadvantages:**
- More API surface area to maintain
- Potential for builder complexity
- May be over-engineered for simple cases

### Solution 3: Evaluation Preset Methods

Provide high-level convenience methods for common evaluation patterns:

```go
package langfuse

// NewRAGTrace creates a trace pre-configured for RAG evaluation
func (c *Client) NewRAGTrace(name string) *RAGTraceBuilder {
    return &RAGTraceBuilder{
        TraceBuilder: c.NewTrace().Name(name),
        ragInput:     &RAGInput{},
    }
}

// RAGTraceBuilder extends TraceBuilder with RAG-specific methods
type RAGTraceBuilder struct {
    *TraceBuilder
    ragInput  *RAGInput
    ragOutput *RAGOutput
}

// Query sets the user's query
func (b *RAGTraceBuilder) Query(query string) *RAGTraceBuilder {
    b.ragInput.Query = query
    return b
}

// Context adds retrieved context chunks
func (b *RAGTraceBuilder) Context(chunks ...string) *RAGTraceBuilder {
    b.ragInput.Context = append(b.ragInput.Context, chunks...)
    return b
}

// GroundTruth sets the expected answer
func (b *RAGTraceBuilder) GroundTruth(truth string) *RAGTraceBuilder {
    b.ragInput.GroundTruth = truth
    return b
}

// Create creates the trace with RAG input structure
func (b *RAGTraceBuilder) Create() (*RAGTraceContext, error) {
    b.TraceBuilder.Input(b.ragInput)

    ctx, err := b.TraceBuilder.Create()
    if err != nil {
        return nil, err
    }

    return &RAGTraceContext{
        TraceContext: ctx,
        ragInput:     b.ragInput,
    }, nil
}

// RAGTraceContext extends TraceContext with RAG-specific methods
type RAGTraceContext struct {
    *TraceContext
    ragInput  *RAGInput
    ragOutput *RAGOutput
}

// UpdateOutput sets the RAG output
func (r *RAGTraceContext) UpdateOutput(output string, citations ...string) error {
    r.ragOutput = &RAGOutput{
        Output:    output,
        Citations: citations,
    }
    return r.Update().Output(r.ragOutput).Apply()
}

// ValidateForEvaluation checks if the trace has all required fields
func (r *RAGTraceContext) ValidateForEvaluation() error {
    if r.ragInput.Query == "" {
        return NewValidationError("query", "query is required for RAG evaluation")
    }
    if len(r.ragInput.Context) == 0 {
        return NewValidationError("context", "context is required for RAG evaluation")
    }
    if r.ragOutput == nil || r.ragOutput.Output == "" {
        return NewValidationError("output", "output is required for RAG evaluation")
    }
    return nil
}
```

**Usage Example:**

```go
// Clean, purpose-built API
trace, err := client.NewRAGTrace("customer-support").
    Query("How do I reset my password?").
    Context(
        "Users can reset passwords from the login page.",
        "Click 'Forgot Password' to receive a reset email.",
    ).
    GroundTruth("Click 'Forgot Password' on the login page to reset.").
    UserID("user-123").
    Tags([]string{"support", "auth"}).
    Create()

if err != nil {
    log.Fatal(err)
}

// Process the query...
response := "To reset your password, click 'Forgot Password' on the login page."

// Update with output
err = trace.UpdateOutput(response, "help-doc-auth.md")
if err != nil {
    log.Fatal(err)
}

// Validate before running evaluations
if err := trace.ValidateForEvaluation(); err != nil {
    log.Printf("Warning: Trace may not be evaluation-ready: %v", err)
}

// Similarly for Q&A
qaTrace, err := client.NewQATrace("trivia-bot").
    Query("What is the capital of Japan?").
    GroundTruth("Tokyo").
    Create()

// And for summarization
summaryTrace, err := client.NewSummarizationTrace("article-summary").
    Text(longArticle).
    MaxLength(500).
    GroundTruth(expertSummary).
    Create()
```

**Advantages:**
- Most discoverable - clear from function names
- Prevents structural errors entirely
- Built-in validation support
- Excellent IDE support with autocomplete
- Self-documenting code

**Disadvantages:**
- Larger API surface
- Separate builder types to maintain
- Less flexible for custom patterns

### Solution 4: Validation and Warnings

Add opt-in validation that warns users when traces don't match evaluator expectations:

```go
package langfuse

// EvaluatorRequirements defines what fields an evaluator needs
type EvaluatorRequirements struct {
    Name           string
    RequiredFields []string
    OptionalFields []string
}

var (
    // RAGEvaluator defines requirements for RAG evaluations
    RAGEvaluator = EvaluatorRequirements{
        Name:           "RAG",
        RequiredFields: []string{"query", "context", "output"},
        OptionalFields: []string{"ground_truth", "citations"},
    }

    // QAEvaluator defines requirements for Q&A evaluations
    QAEvaluator = EvaluatorRequirements{
        Name:           "Q&A",
        RequiredFields: []string{"query", "output"},
        OptionalFields: []string{"ground_truth"},
    }

    // SummarizationEvaluator defines requirements
    SummarizationEvaluator = EvaluatorRequirements{
        Name:           "Summarization",
        RequiredFields: []string{"input", "output"},
        OptionalFields: []string{"ground_truth"},
    }
)

// ValidateForEvaluator checks if trace structure matches evaluator requirements
func (t *TraceContext) ValidateForEvaluator(reqs EvaluatorRequirements) error {
    // Extract fields from input/output
    inputFields := extractFields(t.trace.Input)
    outputFields := extractFields(t.trace.Output)
    allFields := append(inputFields, outputFields...)

    var missing []string
    for _, required := range reqs.RequiredFields {
        if !contains(allFields, required) {
            missing = append(missing, required)
        }
    }

    if len(missing) > 0 {
        return fmt.Errorf("trace missing required fields for %s evaluator: %v",
            reqs.Name, missing)
    }

    return nil
}

// WithValidation enables validation for specific evaluators
func WithValidation(evaluators ...EvaluatorRequirements) ClientOption {
    return func(c *Client) error {
        c.validationEnabled = true
        c.evaluators = evaluators
        return nil
    }
}

// Helper to extract field names from interface{}
func extractFields(data interface{}) []string {
    // Implementation would use reflection or type assertion
    // to extract JSON field names from structs or map keys
    return nil
}
```

**Usage Example:**

```go
// Enable validation at client level
client, err := langfuse.New(
    publicKey,
    secretKey,
    langfuse.WithValidation(
        langfuse.RAGEvaluator,
        langfuse.QAEvaluator,
    ),
)

// Create trace - automatic validation on Create()
trace, err := client.NewTrace().
    Name("search-query").
    Input(map[string]interface{}{
        "message": "What is Go?", // Wrong field name!
    }).
    Create()
// Error: trace missing required fields for RAG evaluator: [query, context]

// Fix the structure
trace, err = client.NewTrace().
    Name("search-query").
    Input(&langfuse.RAGInput{
        Query: "What is Go?",
        Context: []string{"Go is a programming language"},
    }).
    Create() // Validation passes

// Manual validation
err = trace.ValidateForEvaluator(langfuse.RAGEvaluator)
if err != nil {
    log.Printf("Warning: %v", err)
}
```

**Advantages:**
- Catches mistakes early
- Works with any input structure (maps or structs)
- Opt-in, doesn't affect existing code
- Helpful error messages

**Disadvantages:**
- Requires reflection for map validation
- May have performance overhead
- Can't catch all issues at compile time

## Recommended Approach

**Implement all four solutions in phases**, as they complement each other:

### Phase 1: Foundation (Typed Structs + Validation)
- Add typed input/output structs (Solution 1)
- Add validation utilities (Solution 4)
- Document evaluator requirements

**Why:** Provides immediate value with minimal API surface. Users can opt-in to type safety while maintaining backwards compatibility.

**Timeline:** 2-3 weeks

### Phase 2: Enhanced Builders (Evaluation Presets)
- Add `NewRAGTrace`, `NewQATrace`, etc. (Solution 3)
- Add specialized context types
- Add convenience update methods

**Why:** Provides the best developer experience for common cases. Builds on Phase 1 types.

**Timeline:** 3-4 weeks

### Phase 3: Advanced Patterns (Context Builders)
- Add fluent context builders (Solution 2)
- Add functional options for evaluation contexts
- Add support for custom evaluator types

**Why:** Enables power users and custom evaluation scenarios. Optional for most users.

**Timeline:** 2-3 weeks

## Detailed Design

### Package Structure

```
langfuse/
├── client.go              # Core client
├── trace.go               # Trace builders
├── ingestion.go           # Existing ingestion code
├── types.go               # Existing types
├── evaluation.go          # NEW: Evaluation types and structs
├── evaluation_builders.go # NEW: Specialized builders
├── validation.go          # NEW: Validation utilities
└── examples/
    └── evaluation/
        ├── rag.go         # RAG example
        ├── qa.go          # Q&A example
        └── summarization.go
```

### Complete Code Examples

#### evaluation.go

```go
package langfuse

// EvaluationType represents supported evaluation scenarios
type EvaluationType string

const (
    EvaluationTypeRAG            EvaluationType = "rag"
    EvaluationTypeQA             EvaluationType = "qa"
    EvaluationTypeSummarization  EvaluationType = "summarization"
    EvaluationTypeClassification EvaluationType = "classification"
    EvaluationTypeToxicity       EvaluationType = "toxicity"
)

// RAGInput represents input for RAG (Retrieval-Augmented Generation) workflows.
// This structure matches Langfuse's RAG evaluator expectations.
//
// Example:
//
//	input := &langfuse.RAGInput{
//	    Query: "What are Go's concurrency features?",
//	    Context: []string{
//	        "Go has built-in goroutines for lightweight concurrency.",
//	        "Channels enable communication between goroutines.",
//	    },
//	    GroundTruth: "Go provides goroutines and channels for concurrency.",
//	}
type RAGInput struct {
    // Query is the user's question or search query (required)
    Query string `json:"query"`

    // Context contains retrieved context chunks from your knowledge base (required)
    Context []string `json:"context"`

    // GroundTruth is the expected correct answer for evaluation (optional)
    GroundTruth string `json:"ground_truth,omitempty"`

    // AdditionalContext allows passing extra metadata
    AdditionalContext map[string]interface{} `json:"additional_context,omitempty"`
}

// RAGOutput represents output from a RAG workflow.
//
// Example:
//
//	output := &langfuse.RAGOutput{
//	    Output: "Go provides goroutines for lightweight concurrency...",
//	    Citations: []string{"golang-docs.txt", "concurrency-guide.md"},
//	    SourceChunks: []int{0, 1}, // Indices of context chunks used
//	}
type RAGOutput struct {
    // Output is the generated response (required)
    Output string `json:"output"`

    // Citations lists source documents used (optional)
    Citations []string `json:"citations,omitempty"`

    // SourceChunks indicates which context chunks were used (optional)
    SourceChunks []int `json:"source_chunks,omitempty"`

    // Confidence is the model's confidence in the answer (optional)
    Confidence float64 `json:"confidence,omitempty"`

    // Metadata allows passing additional metadata
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// QAInput represents input for question-answering workflows.
//
// Example:
//
//	input := &langfuse.QAInput{
//	    Query: "What is the capital of France?",
//	    GroundTruth: "Paris",
//	}
type QAInput struct {
    // Query is the user's question (required)
    Query string `json:"query"`

    // GroundTruth is the expected correct answer for evaluation (optional)
    GroundTruth string `json:"ground_truth,omitempty"`

    // Context provides additional context for the question (optional)
    Context string `json:"context,omitempty"`
}

// QAOutput represents output from a question-answering workflow.
type QAOutput struct {
    // Output is the generated answer (required)
    Output string `json:"output"`

    // Confidence is the model's confidence in the answer (optional)
    Confidence float64 `json:"confidence,omitempty"`

    // Reasoning provides explanation for the answer (optional)
    Reasoning string `json:"reasoning,omitempty"`

    // Metadata allows passing additional metadata
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// SummarizationInput represents input for summarization workflows.
//
// Example:
//
//	input := &langfuse.SummarizationInput{
//	    Input: longArticleText,
//	    MaxLength: 500,
//	    GroundTruth: expertSummary,
//	}
type SummarizationInput struct {
    // Input is the original text to summarize (required)
    Input string `json:"input"`

    // GroundTruth is a reference summary for evaluation (optional)
    GroundTruth string `json:"ground_truth,omitempty"`

    // MaxLength specifies target summary length in words (optional)
    MaxLength int `json:"max_length,omitempty"`

    // Style specifies summary style (e.g., "bullet_points", "paragraph") (optional)
    Style string `json:"style,omitempty"`
}

// SummarizationOutput represents output from summarization workflows.
type SummarizationOutput struct {
    // Output is the generated summary (required)
    Output string `json:"output"`

    // Length is the summary length in words (optional)
    Length int `json:"length,omitempty"`

    // CompressionRatio indicates input:output length ratio (optional)
    CompressionRatio float64 `json:"compression_ratio,omitempty"`

    // Metadata allows passing additional metadata
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ClassificationInput represents input for classification workflows.
type ClassificationInput struct {
    // Input is the text to classify (required)
    Input string `json:"input"`

    // Classes lists possible classification categories (optional)
    Classes []string `json:"classes,omitempty"`

    // GroundTruth is the expected classification for evaluation (optional)
    GroundTruth string `json:"ground_truth,omitempty"`
}

// ClassificationOutput represents output from classification workflows.
type ClassificationOutput struct {
    // Output is the predicted class (required)
    Output string `json:"output"`

    // Confidence is the prediction confidence (optional)
    Confidence float64 `json:"confidence,omitempty"`

    // Scores provides confidence scores for all classes (optional)
    Scores map[string]float64 `json:"scores,omitempty"`

    // Metadata allows passing additional metadata
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}
```

#### evaluation_builders.go

```go
package langfuse

// NewRAGTrace creates a trace pre-configured for RAG evaluation.
//
// Example:
//
//	trace, err := client.NewRAGTrace("product-search").
//	    Query("What are the best wireless headphones?").
//	    Context(
//	        "Sony WH-1000XM5 features industry-leading noise cancellation.",
//	        "Bose QuietComfort 45 offers excellent comfort.",
//	    ).
//	    Create()
func (c *Client) NewRAGTrace(name string) *RAGTraceBuilder {
    return &RAGTraceBuilder{
        TraceBuilder: c.NewTrace().Name(name),
        ragInput:     &RAGInput{},
    }
}

// RAGTraceBuilder provides a fluent interface for creating RAG traces.
type RAGTraceBuilder struct {
    *TraceBuilder
    ragInput *RAGInput
}

// Query sets the user's query/question.
func (b *RAGTraceBuilder) Query(query string) *RAGTraceBuilder {
    b.ragInput.Query = query
    return b
}

// Context adds retrieved context chunks.
func (b *RAGTraceBuilder) Context(chunks ...string) *RAGTraceBuilder {
    b.ragInput.Context = append(b.ragInput.Context, chunks...)
    return b
}

// GroundTruth sets the expected correct answer.
func (b *RAGTraceBuilder) GroundTruth(truth string) *RAGTraceBuilder {
    b.ragInput.GroundTruth = truth
    return b
}

// AdditionalContext sets extra context metadata.
func (b *RAGTraceBuilder) AdditionalContext(ctx map[string]interface{}) *RAGTraceBuilder {
    b.ragInput.AdditionalContext = ctx
    return b
}

// Validate checks if required fields are set.
func (b *RAGTraceBuilder) Validate() error {
    if b.ragInput.Query == "" {
        return NewValidationError("query", "query is required for RAG traces")
    }
    if len(b.ragInput.Context) == 0 {
        return NewValidationError("context", "at least one context chunk is required")
    }
    return b.TraceBuilder.Validate()
}

// Create creates the RAG trace.
func (b *RAGTraceBuilder) Create() (*RAGTraceContext, error) {
    if err := b.Validate(); err != nil {
        return nil, err
    }

    b.TraceBuilder.Input(b.ragInput)

    ctx, err := b.TraceBuilder.Create()
    if err != nil {
        return nil, err
    }

    return &RAGTraceContext{
        TraceContext: ctx,
        ragInput:     b.ragInput,
    }, nil
}

// RAGTraceContext extends TraceContext with RAG-specific methods.
type RAGTraceContext struct {
    *TraceContext
    ragInput  *RAGInput
    ragOutput *RAGOutput
}

// UpdateOutput sets the RAG output with citations.
func (r *RAGTraceContext) UpdateOutput(output string, citations ...string) error {
    r.ragOutput = &RAGOutput{
        Output:    output,
        Citations: citations,
    }
    return r.Update().Output(r.ragOutput).Apply()
}

// UpdateOutputWithMetadata sets the RAG output with full metadata.
func (r *RAGTraceContext) UpdateOutputWithMetadata(output *RAGOutput) error {
    r.ragOutput = output
    return r.Update().Output(r.ragOutput).Apply()
}

// ValidateForEvaluation checks if the trace has all required fields.
func (r *RAGTraceContext) ValidateForEvaluation() error {
    if r.ragInput.Query == "" {
        return NewValidationError("query", "query is required for RAG evaluation")
    }
    if len(r.ragInput.Context) == 0 {
        return NewValidationError("context", "context is required for RAG evaluation")
    }
    if r.ragOutput == nil || r.ragOutput.Output == "" {
        return NewValidationError("output", "output is required for RAG evaluation")
    }
    return nil
}

// GetInput returns the RAG input.
func (r *RAGTraceContext) GetInput() *RAGInput {
    return r.ragInput
}

// GetOutput returns the RAG output.
func (r *RAGTraceContext) GetOutput() *RAGOutput {
    return r.ragOutput
}

// NewQATrace creates a trace pre-configured for Q&A evaluation.
//
// Example:
//
//	trace, err := client.NewQATrace("trivia-bot").
//	    Query("What is the capital of Japan?").
//	    GroundTruth("Tokyo").
//	    Create()
func (c *Client) NewQATrace(name string) *QATraceBuilder {
    return &QATraceBuilder{
        TraceBuilder: c.NewTrace().Name(name),
        qaInput:      &QAInput{},
    }
}

// QATraceBuilder provides a fluent interface for creating Q&A traces.
type QATraceBuilder struct {
    *TraceBuilder
    qaInput *QAInput
}

// Query sets the user's question.
func (b *QATraceBuilder) Query(query string) *QATraceBuilder {
    b.qaInput.Query = query
    return b
}

// GroundTruth sets the expected answer.
func (b *QATraceBuilder) GroundTruth(truth string) *QATraceBuilder {
    b.qaInput.GroundTruth = truth
    return b
}

// Context sets additional context for the question.
func (b *QATraceBuilder) Context(context string) *QATraceBuilder {
    b.qaInput.Context = context
    return b
}

// Validate checks if required fields are set.
func (b *QATraceBuilder) Validate() error {
    if b.qaInput.Query == "" {
        return NewValidationError("query", "query is required for Q&A traces")
    }
    return b.TraceBuilder.Validate()
}

// Create creates the Q&A trace.
func (b *QATraceBuilder) Create() (*QATraceContext, error) {
    if err := b.Validate(); err != nil {
        return nil, err
    }

    b.TraceBuilder.Input(b.qaInput)

    ctx, err := b.TraceBuilder.Create()
    if err != nil {
        return nil, err
    }

    return &QATraceContext{
        TraceContext: ctx,
        qaInput:      b.qaInput,
    }, nil
}

// QATraceContext extends TraceContext with Q&A-specific methods.
type QATraceContext struct {
    *TraceContext
    qaInput  *QAInput
    qaOutput *QAOutput
}

// UpdateOutput sets the Q&A output.
func (q *QATraceContext) UpdateOutput(output string, confidence float64) error {
    q.qaOutput = &QAOutput{
        Output:     output,
        Confidence: confidence,
    }
    return q.Update().Output(q.qaOutput).Apply()
}

// UpdateOutputWithMetadata sets the Q&A output with full metadata.
func (q *QATraceContext) UpdateOutputWithMetadata(output *QAOutput) error {
    q.qaOutput = output
    return q.Update().Output(q.qaOutput).Apply()
}

// ValidateForEvaluation checks if the trace has all required fields.
func (q *QATraceContext) ValidateForEvaluation() error {
    if q.qaInput.Query == "" {
        return NewValidationError("query", "query is required for Q&A evaluation")
    }
    if q.qaOutput == nil || q.qaOutput.Output == "" {
        return NewValidationError("output", "output is required for Q&A evaluation")
    }
    return nil
}

// NewSummarizationTrace creates a trace pre-configured for summarization evaluation.
//
// Example:
//
//	trace, err := client.NewSummarizationTrace("article-summary").
//	    Text(longArticle).
//	    MaxLength(500).
//	    Create()
func (c *Client) NewSummarizationTrace(name string) *SummarizationTraceBuilder {
    return &SummarizationTraceBuilder{
        TraceBuilder:  c.NewTrace().Name(name),
        summaryInput: &SummarizationInput{},
    }
}

// SummarizationTraceBuilder provides a fluent interface for creating summarization traces.
type SummarizationTraceBuilder struct {
    *TraceBuilder
    summaryInput *SummarizationInput
}

// Text sets the original text to summarize.
func (b *SummarizationTraceBuilder) Text(text string) *SummarizationTraceBuilder {
    b.summaryInput.Input = text
    return b
}

// MaxLength sets the target summary length.
func (b *SummarizationTraceBuilder) MaxLength(length int) *SummarizationTraceBuilder {
    b.summaryInput.MaxLength = length
    return b
}

// Style sets the summary style.
func (b *SummarizationTraceBuilder) Style(style string) *SummarizationTraceBuilder {
    b.summaryInput.Style = style
    return b
}

// GroundTruth sets the reference summary.
func (b *SummarizationTraceBuilder) GroundTruth(truth string) *SummarizationTraceBuilder {
    b.summaryInput.GroundTruth = truth
    return b
}

// Validate checks if required fields are set.
func (b *SummarizationTraceBuilder) Validate() error {
    if b.summaryInput.Input == "" {
        return NewValidationError("input", "text to summarize is required")
    }
    return b.TraceBuilder.Validate()
}

// Create creates the summarization trace.
func (b *SummarizationTraceBuilder) Create() (*SummarizationTraceContext, error) {
    if err := b.Validate(); err != nil {
        return nil, err
    }

    b.TraceBuilder.Input(b.summaryInput)

    ctx, err := b.TraceBuilder.Create()
    if err != nil {
        return nil, err
    }

    return &SummarizationTraceContext{
        TraceContext: ctx,
        summaryInput: b.summaryInput,
    }, nil
}

// SummarizationTraceContext extends TraceContext with summarization-specific methods.
type SummarizationTraceContext struct {
    *TraceContext
    summaryInput  *SummarizationInput
    summaryOutput *SummarizationOutput
}

// UpdateOutput sets the summary output.
func (s *SummarizationTraceContext) UpdateOutput(summary string) error {
    s.summaryOutput = &SummarizationOutput{
        Output: summary,
        Length: len(summary),
    }
    return s.Update().Output(s.summaryOutput).Apply()
}

// UpdateOutputWithMetadata sets the summary output with full metadata.
func (s *SummarizationTraceContext) UpdateOutputWithMetadata(output *SummarizationOutput) error {
    s.summaryOutput = output
    return s.Update().Output(s.summaryOutput).Apply()
}

// ValidateForEvaluation checks if the trace has all required fields.
func (s *SummarizationTraceContext) ValidateForEvaluation() error {
    if s.summaryInput.Input == "" {
        return NewValidationError("input", "input text is required for summarization evaluation")
    }
    if s.summaryOutput == nil || s.summaryOutput.Output == "" {
        return NewValidationError("output", "summary output is required for evaluation")
    }
    return nil
}
```

#### validation.go

```go
package langfuse

import (
    "fmt"
    "reflect"
)

// EvaluatorRequirements defines what fields an evaluator expects.
type EvaluatorRequirements struct {
    Name           string
    RequiredFields []string
    OptionalFields []string
}

var (
    // RAGEvaluator defines requirements for RAG evaluations
    RAGEvaluator = EvaluatorRequirements{
        Name:           "RAG",
        RequiredFields: []string{"query", "context", "output"},
        OptionalFields: []string{"ground_truth", "citations", "source_chunks"},
    }

    // QAEvaluator defines requirements for Q&A evaluations
    QAEvaluator = EvaluatorRequirements{
        Name:           "Q&A",
        RequiredFields: []string{"query", "output"},
        OptionalFields: []string{"ground_truth", "confidence"},
    }

    // SummarizationEvaluator defines requirements for summarization evaluations
    SummarizationEvaluator = EvaluatorRequirements{
        Name:           "Summarization",
        RequiredFields: []string{"input", "output"},
        OptionalFields: []string{"ground_truth", "compression_ratio"},
    }

    // ClassificationEvaluator defines requirements for classification evaluations
    ClassificationEvaluator = EvaluatorRequirements{
        Name:           "Classification",
        RequiredFields: []string{"input", "output"},
        OptionalFields: []string{"ground_truth", "confidence", "scores"},
    }
)

// ValidateForEvaluator checks if a trace structure matches evaluator requirements.
func ValidateForEvaluator(input, output interface{}, reqs EvaluatorRequirements) error {
    inputFields := extractFields(input)
    outputFields := extractFields(output)
    allFields := append(inputFields, outputFields...)

    var missing []string
    for _, required := range reqs.RequiredFields {
        if !contains(allFields, required) {
            missing = append(missing, required)
        }
    }

    if len(missing) > 0 {
        return fmt.Errorf(
            "trace missing required fields for %s evaluator: %v (available: %v)",
            reqs.Name, missing, allFields,
        )
    }

    return nil
}

// extractFields extracts field names from a struct or map.
func extractFields(data interface{}) []string {
    if data == nil {
        return nil
    }

    fields := []string{}

    // Handle map[string]interface{}
    if m, ok := data.(map[string]interface{}); ok {
        for k := range m {
            fields = append(fields, k)
        }
        return fields
    }

    // Handle structs using reflection
    v := reflect.ValueOf(data)
    if v.Kind() == reflect.Ptr {
        v = v.Elem()
    }

    if v.Kind() != reflect.Struct {
        return fields
    }

    t := v.Type()
    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)

        // Get JSON tag name
        tag := field.Tag.Get("json")
        if tag == "" || tag == "-" {
            continue
        }

        // Extract field name before comma
        name := tag
        if idx := len(tag); idx > 0 {
            for j, r := range tag {
                if r == ',' {
                    name = tag[:j]
                    break
                }
            }
        }

        // Skip omitempty fields that are zero
        fieldValue := v.Field(i)
        if isOmitEmpty(tag) && fieldValue.IsZero() {
            continue
        }

        fields = append(fields, name)
    }

    return fields
}

// isOmitEmpty checks if a JSON tag includes omitempty.
func isOmitEmpty(tag string) bool {
    for i, r := range tag {
        if r == ',' && i+1 < len(tag) {
            return tag[i+1:] == "omitempty"
        }
    }
    return false
}

// contains checks if a slice contains a string.
func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}

// ValidationWarning represents a non-fatal validation issue.
type ValidationWarning struct {
    Field   string
    Message string
}

// Error implements the error interface.
func (v ValidationWarning) Error() string {
    return fmt.Sprintf("validation warning for field '%s': %s", v.Field, v.Message)
}

// NewValidationWarning creates a new validation warning.
func NewValidationWarning(field, message string) error {
    return ValidationWarning{
        Field:   field,
        Message: message,
    }
}
```

### Usage Examples

#### Example 1: RAG Application

```go
package main

import (
    "context"
    "log"

    langfuse "github.com/jdziat/langfuse-go"
)

func main() {
    client, err := langfuse.New(
        os.Getenv("LANGFUSE_PUBLIC_KEY"),
        os.Getenv("LANGFUSE_SECRET_KEY"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Shutdown(context.Background())

    // Create evaluation-ready RAG trace
    trace, err := client.NewRAGTrace("product-search").
        Query("What are the best wireless headphones under $300?").
        Context(
            "Sony WH-1000XM5: $399, industry-leading noise cancellation",
            "Bose QuietComfort 45: $329, excellent comfort",
            "Sennheiser Momentum 4: $279, best battery life (60hrs)",
        ).
        UserID("user-123").
        Tags([]string{"ecommerce", "search"}).
        Create()
    if err != nil {
        log.Fatal(err)
    }

    // Perform retrieval and generation...
    response := "Based on your budget, the Sennheiser Momentum 4 ($279) " +
        "offers the best value with 60-hour battery life."

    // Update with output
    err = trace.UpdateOutput(response, "product-db", "reviews-db")
    if err != nil {
        log.Printf("Failed to update output: %v", err)
    }

    // Validate before evaluation
    if err := trace.ValidateForEvaluation(); err != nil {
        log.Printf("Warning: %v", err)
    }

    // Score the generation
    trace.Score().
        Name("accuracy").
        NumericValue(0.95).
        Comment("Correctly identified best value option").
        Create()
}
```

#### Example 2: Q&A Bot

```go
func handleQuestion(client *langfuse.Client, question string) {
    trace, err := client.NewQATrace("trivia-bot").
        Query(question).
        UserID("player-456").
        Create()
    if err != nil {
        log.Printf("Failed to create trace: %v", err)
        return
    }

    // Generate answer
    answer, confidence := generateAnswer(question)

    // Update with answer and confidence
    err = trace.UpdateOutput(answer, confidence)
    if err != nil {
        log.Printf("Failed to update output: %v", err)
    }
}
```

#### Example 3: Summarization Service

```go
func summarizeArticle(client *langfuse.Client, article string) {
    trace, err := client.NewSummarizationTrace("news-summary").
        Text(article).
        MaxLength(500).
        Style("paragraph").
        Create()
    if err != nil {
        log.Printf("Failed to create trace: %v", err)
        return
    }

    // Generate summary
    summary := generateSummary(article, 500)

    // Update with rich metadata
    err = trace.UpdateOutputWithMetadata(&langfuse.SummarizationOutput{
        Output:           summary,
        Length:           len(summary),
        CompressionRatio: float64(len(article)) / float64(len(summary)),
        Metadata: map[string]interface{}{
            "model":       "claude-3-sonnet",
            "temperature": 0.7,
        },
    })
    if err != nil {
        log.Printf("Failed to update output: %v", err)
    }
}
```

#### Example 4: Backwards Compatible Usage

```go
// Existing code continues to work unchanged
trace, err := client.NewTrace().
    Name("custom-workflow").
    Input(map[string]interface{}{
        "custom_field": "value",
    }).
    Create()

// But users can now also use typed structs
trace2, err := client.NewTrace().
    Name("rag-workflow").
    Input(&langfuse.RAGInput{
        Query:   "What is Go?",
        Context: []string{"Go is a programming language"},
    }).
    Create()

// And can validate either approach
err = langfuse.ValidateForEvaluator(
    trace2.Input,
    trace2.Output,
    langfuse.RAGEvaluator,
)
```

## Migration Path

### For Existing Users

No migration required! All existing code continues to work:

```go
// This still works
trace, err := client.NewTrace().
    Input(map[string]interface{}{"message": "hello"}).
    Create()
```

### Gradual Adoption

Users can adopt new features incrementally:

**Step 1:** Start using typed structs

```go
trace, err := client.NewTrace().
    Input(&langfuse.RAGInput{
        Query:   "question",
        Context: []string{"context"},
    }).
    Create()
```

**Step 2:** Switch to specialized builders

```go
trace, err := client.NewRAGTrace("name").
    Query("question").
    Context("context").
    Create()
```

**Step 3:** Add validation

```go
if err := trace.ValidateForEvaluation(); err != nil {
    log.Printf("Warning: %v", err)
}
```

## Testing Strategy

### Unit Tests

```go
func TestRAGTraceBuilder(t *testing.T) {
    client := setupTestClient(t)

    trace, err := client.NewRAGTrace("test").
        Query("What is Go?").
        Context("Go is a language").
        Create()

    require.NoError(t, err)
    assert.Equal(t, "What is Go?", trace.GetInput().Query)
}

func TestValidation(t *testing.T) {
    input := &RAGInput{
        Query:   "test",
        Context: []string{"context"},
    }
    output := &RAGOutput{
        Output: "answer",
    }

    err := ValidateForEvaluator(input, output, RAGEvaluator)
    assert.NoError(t, err)
}
```

### Integration Tests

```go
func TestRAGEvaluation(t *testing.T) {
    client := setupRealClient(t)
    defer client.Shutdown(context.Background())

    trace, err := client.NewRAGTrace("integration-test").
        Query("Test question").
        Context("Test context").
        Create()
    require.NoError(t, err)

    err = trace.UpdateOutput("Test answer")
    require.NoError(t, err)

    err = trace.ValidateForEvaluation()
    assert.NoError(t, err)

    client.Flush(context.Background())
}
```

## Documentation Updates

### API Documentation

- Add godoc comments for all new types
- Include usage examples in package docs
- Document evaluator requirements

### User Guide

Create new guide: `docs/evaluation-ready-tracing.md`

Sections:
1. Introduction to evaluation-ready tracing
2. Understanding evaluator requirements
3. Using typed input/output structs
4. Working with specialized builders
5. Validation and debugging
6. Migration from untyped traces
7. Custom evaluation patterns

### Examples

Add comprehensive examples:
- `examples/evaluation/rag/main.go`
- `examples/evaluation/qa/main.go`
- `examples/evaluation/summarization/main.go`
- `examples/evaluation/custom/main.go`

## Performance Considerations

### Memory Impact

- Typed structs: Minimal overhead (same as maps)
- Validation: Only runs when explicitly called
- Builders: No additional allocations vs. existing builders

### CPU Impact

- Reflection in validation: ~1-2μs per validation call
- Struct marshaling: Same as existing map marshaling
- Builder methods: Inline-able, zero overhead

### Benchmarks

```go
func BenchmarkRAGTrace(b *testing.B) {
    client := setupBenchClient(b)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        client.NewRAGTrace("bench").
            Query("test").
            Context("context").
            Create()
    }
}

func BenchmarkValidation(b *testing.B) {
    input := &RAGInput{Query: "test", Context: []string{"ctx"}}
    output := &RAGOutput{Output: "answer"}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        ValidateForEvaluator(input, output, RAGEvaluator)
    }
}
```

## Security Considerations

- No additional security risks
- Validation doesn't execute user code
- Type safety reduces injection risks
- All existing security measures apply

## Alternatives Considered

### 1. JSON Schema Validation

**Approach:** Define JSON schemas for each evaluator type

**Pros:**
- Standard approach
- Tool support

**Cons:**
- Runtime only
- No IDE support
- More dependencies
- Not idiomatic Go

**Verdict:** Rejected - too heavyweight for Go

### 2. Code Generation

**Approach:** Generate builders from evaluator definitions

**Pros:**
- Always in sync
- Compile-time safety

**Cons:**
- Build complexity
- Generated code bloat
- Less readable

**Verdict:** Rejected - premature optimization

### 3. Interface-Based Approach

**Approach:** Define interfaces for evaluation inputs/outputs

**Pros:**
- Flexible
- Testable

**Cons:**
- More abstract
- Harder to use
- Less discoverable

**Verdict:** Rejected - conflicts with simplicity goal

## Open Questions

1. **Should validation be opt-in or opt-out?**
   - **Recommendation:** Opt-in to avoid breaking changes

2. **Should we support custom evaluator definitions?**
   - **Recommendation:** Yes, in Phase 3

3. **How to handle evaluator versioning?**
   - **Recommendation:** Version in evaluator name (e.g., `RAGEvaluatorV2`)

4. **Should we provide evaluation execution in the SDK?**
   - **Recommendation:** No, keep SDK focused on tracing

## Success Metrics

### Developer Experience
- Reduction in evaluation setup errors
- Time to first successful evaluation
- Code readability scores

### Adoption
- % of traces using typed structs
- % using specialized builders
- Community feedback sentiment

### Quality
- Reduction in evaluation failures
- Increase in evaluation usage
- Support ticket reduction

## Timeline

### Phase 1: Foundation (Weeks 1-3)
- Week 1: Implement typed structs
- Week 2: Implement validation utilities
- Week 3: Documentation and tests

### Phase 2: Enhanced Builders (Weeks 4-7)
- Week 4: Implement RAG builder
- Week 5: Implement Q&A and summarization builders
- Week 6: Comprehensive examples
- Week 7: Integration testing

### Phase 3: Advanced Patterns (Weeks 8-10)
- Week 8: Context builders
- Week 9: Custom evaluator support
- Week 10: Final polish and docs

## Appendix

### A. Langfuse Evaluator Field Reference

#### RAG Evaluators

**Required:**
- `query` (string): The user's question
- `context` (string or []string): Retrieved context
- `output` (string): Generated response

**Optional:**
- `ground_truth` (string): Expected answer

#### Q&A Evaluators

**Required:**
- `query` (string): The question
- `output` (string): The answer

**Optional:**
- `ground_truth` (string): Expected answer

#### Summarization Evaluators

**Required:**
- `input` (string): Original text
- `output` (string): Summary

**Optional:**
- `ground_truth` (string): Reference summary

### B. Complete Type Hierarchy

```
TraceBuilder
├── RAGTraceBuilder → RAGTraceContext
├── QATraceBuilder → QATraceContext
└── SummarizationTraceBuilder → SummarizationTraceContext

Input Types
├── RAGInput
├── QAInput
├── SummarizationInput
└── ClassificationInput

Output Types
├── RAGOutput
├── QAOutput
├── SummarizationOutput
└── ClassificationOutput

Validation
├── EvaluatorRequirements
├── RAGEvaluator
├── QAEvaluator
├── SummarizationEvaluator
└── ValidateForEvaluator()
```

### C. Related Work

- Python SDK: Uses decorators and context managers
- TypeScript SDK: Uses type parameters and generics
- Langchain: Uses callbacks and handlers
- OpenTelemetry: Uses attributes and semantic conventions

## Conclusion

This proposal introduces evaluation-ready tracing to the Langfuse Go SDK through a layered approach:

1. **Type safety** via predefined structs
2. **Convenience** via specialized builders
3. **Validation** via opt-in checking
4. **Backwards compatibility** via gradual adoption

The solution is Go-idiomatic, maintains the SDK's zero-dependency philosophy, and provides a superior developer experience for users building evaluated LLM applications.

## References

- [Langfuse Evaluation Docs](https://langfuse.com/docs/scores/model-based-evals)
- [Langfuse Canned Evaluators](https://langfuse.com/docs/scores/model-based-evals/pre-built)
- [Go SDK Repository](https://github.com/jdziat/langfuse-go)
- [Effective Go](https://go.dev/doc/effective_go)
- [Builder Pattern in Go](https://refactoring.guru/design-patterns/builder/go)
