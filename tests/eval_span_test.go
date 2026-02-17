package langfuse_test

import (
	"context"
	"testing"
	"time"

	langfuse "github.com/jdziat/langfuse-go"
	"github.com/jdziat/langfuse-go/langfusetest"
)

func TestEvalSpanBuilder(t *testing.T) {
	client, _ := langfusetest.NewTestClientWithConfig(t,
		langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
	)
	ctx := context.Background()

	trace, err := client.NewTrace().Name("test-trace").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	span, err := trace.NewEvalSpan().
		Name("test-span").
		Type(langfuse.EvalSpanRetrieval).
		WithQuery("What is Go?").
		Create(ctx)

	if err != nil {
		t.Fatalf("Failed to create eval span: %v", err)
	}

	if span.ID() == "" {
		t.Error("Span ID should not be empty")
	}

	if span.GetSpanType() != langfuse.EvalSpanRetrieval {
		t.Errorf("SpanType = %v, want %v", span.GetSpanType(), langfuse.EvalSpanRetrieval)
	}
}

func TestNewRetrievalSpan(t *testing.T) {
	client, _ := langfusetest.NewTestClientWithConfig(t,
		langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
	)
	ctx := context.Background()

	trace, err := client.NewTrace().Name("test-trace").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	span, err := trace.NewRetrievalSpan().
		Name("retrieval").
		WithQuery("test query").
		Create(ctx)

	if err != nil {
		t.Fatalf("Failed to create retrieval span: %v", err)
	}

	if span.GetSpanType() != langfuse.EvalSpanRetrieval {
		t.Errorf("SpanType = %v, want %v", span.GetSpanType(), langfuse.EvalSpanRetrieval)
	}

	state := span.GetEvalState()
	if !state.InputFields["query"] {
		t.Error("query should be in input fields")
	}
}

func TestNewToolCallSpan(t *testing.T) {
	client, _ := langfusetest.NewTestClientWithConfig(t,
		langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
	)
	ctx := context.Background()

	trace, err := client.NewTrace().Name("test-trace").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	span, err := trace.NewToolCallSpan().
		Name("tool-call").
		Create(ctx)

	if err != nil {
		t.Fatalf("Failed to create tool call span: %v", err)
	}

	if span.GetSpanType() != langfuse.EvalSpanToolCall {
		t.Errorf("SpanType = %v, want %v", span.GetSpanType(), langfuse.EvalSpanToolCall)
	}
}

func TestEvalSpanBuilderWithContext(t *testing.T) {
	client, _ := langfusetest.NewTestClientWithConfig(t,
		langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
	)
	ctx := context.Background()

	trace, err := client.NewTrace().Name("test-trace").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	span, err := trace.NewEvalSpan().
		Name("test-span").
		WithQuery("query").
		WithContext("doc1", "doc2").
		Create(ctx)

	if err != nil {
		t.Fatalf("Failed to create eval span: %v", err)
	}

	state := span.GetEvalState()
	if !state.HasContext {
		t.Error("HasContext should be true")
	}
	if !state.InputFields["context"] {
		t.Error("context should be in input fields")
	}
}

func TestEvalSpanBuilderChaining(t *testing.T) {
	client, _ := langfusetest.NewTestClientWithConfig(t,
		langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
	)
	ctx := context.Background()

	trace, err := client.NewTrace().Name("test-trace").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	span, err := trace.NewEvalSpan().
		ID("custom-id").
		Name("test-span").
		Type(langfuse.EvalSpanProcessing).
		Input(map[string]any{"key": "value"}).
		Output("output").
		Metadata(langfuse.Metadata{"meta": "data"}).
		Level(langfuse.ObservationLevelDebug).
		Environment("test").
		Create(ctx)

	if err != nil {
		t.Fatalf("Failed to create eval span: %v", err)
	}

	if span.ID() != "custom-id" {
		t.Errorf("ID = %v, want 'custom-id'", span.ID())
	}
}

func TestEvalSpanContextEndWithContext(t *testing.T) {
	client, _ := langfusetest.NewTestClientWithConfig(t,
		langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
	)
	ctx := context.Background()

	trace, err := client.NewTrace().Name("test-trace").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	span, err := trace.NewRetrievalSpan().
		Name("retrieval").
		WithQuery("test query").
		Create(ctx)

	if err != nil {
		t.Fatalf("Failed to create retrieval span: %v", err)
	}

	// End with retrieved context
	err = span.EndWithContext(ctx, "doc1", "doc2", "doc3")
	if err != nil {
		t.Errorf("EndWithContext failed: %v", err)
	}

	state := span.GetEvalState()
	if !state.HasContext {
		t.Error("HasContext should be true after EndWithContext")
	}
}

func TestEvalSpanContextEndWithToolResult(t *testing.T) {
	client, _ := langfusetest.NewTestClientWithConfig(t,
		langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
	)
	ctx := context.Background()

	trace, err := client.NewTrace().Name("test-trace").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	span, err := trace.NewToolCallSpan().
		Name("tool-execution").
		Create(ctx)

	if err != nil {
		t.Fatalf("Failed to create tool span: %v", err)
	}

	result := &langfuse.ToolCallResult{
		ToolName: "search",
		Input:    map[string]any{"query": "test"},
		Output:   []string{"result1", "result2"},
		Success:  true,
		Duration: 100 * time.Millisecond,
	}

	err = span.EndWithToolResult(ctx, result)
	if err != nil {
		t.Errorf("EndWithToolResult failed: %v", err)
	}
}

func TestEvalSpanContextNewEvalGeneration(t *testing.T) {
	client, _ := langfusetest.NewTestClientWithConfig(t,
		langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
	)
	ctx := context.Background()

	trace, err := client.NewTrace().Name("test-trace").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	// Create retrieval span with context
	span, err := trace.NewRetrievalSpan().
		Name("retrieval").
		WithQuery("test query").
		WithContext("retrieved doc").
		Create(ctx)

	if err != nil {
		t.Fatalf("Failed to create retrieval span: %v", err)
	}

	// Create child generation - should inherit context
	gen := span.NewEvalGeneration()

	if gen == nil {
		t.Fatal("NewEvalGeneration returned nil")
	}

	// The generation should have context awareness from parent
	genCtx, err := gen.Name("llm-call").Model("gpt-4").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create generation from span: %v", err)
	}

	if genCtx.GetEvalState().HasContext != span.GetEvalState().HasContext {
		t.Error("Child generation should inherit context awareness from parent span")
	}
}

func TestEvalSpanContextNewRetrievalSpan(t *testing.T) {
	client, _ := langfusetest.NewTestClientWithConfig(t,
		langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
	)
	ctx := context.Background()

	trace, err := client.NewTrace().Name("test-trace").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	parentSpan, err := trace.NewEvalSpan().
		Name("parent").
		Create(ctx)

	if err != nil {
		t.Fatalf("Failed to create parent span: %v", err)
	}

	childSpan := parentSpan.NewRetrievalSpan()

	if childSpan == nil {
		t.Fatal("NewRetrievalSpan returned nil")
	}

	childCtx, err := childSpan.Name("child-retrieval").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create child retrieval span: %v", err)
	}

	if childCtx.GetSpanType() != langfuse.EvalSpanRetrieval {
		t.Errorf("Child span type = %v, want %v", childCtx.GetSpanType(), langfuse.EvalSpanRetrieval)
	}
}

func TestRetrievalOutputEvalFields(t *testing.T) {
	output := &langfuse.RetrievalOutput{
		Documents:    []string{"doc1", "doc2"},
		NumDocuments: 2,
		Query:        "test query",
		Scores:       []float64{0.9, 0.8},
		Source:       "vector-db",
	}

	fields := output.EvalFields()

	if len(fields["context"].([]string)) != 2 {
		t.Errorf("context should have 2 documents")
	}
	if fields["num_documents"] != 2 {
		t.Errorf("num_documents = %v, want 2", fields["num_documents"])
	}
	if fields["query"] != "test query" {
		t.Errorf("query = %v, want 'test query'", fields["query"])
	}
	if fields["source"] != "vector-db" {
		t.Errorf("source = %v, want 'vector-db'", fields["source"])
	}
}

func TestToolCallResultEvalFields(t *testing.T) {
	result := &langfuse.ToolCallResult{
		ToolName: "calculator",
		Input:    "2+2",
		Output:   "4",
		Success:  true,
		Error:    "",
		Duration: 50 * time.Millisecond,
	}

	fields := result.EvalFields()

	if fields["tool_name"] != "calculator" {
		t.Errorf("tool_name = %v, want 'calculator'", fields["tool_name"])
	}
	if fields["success"] != true {
		t.Errorf("success = %v, want true", fields["success"])
	}
	if fields["tool_input"] != "2+2" {
		t.Errorf("tool_input = %v, want '2+2'", fields["tool_input"])
	}
	if fields["tool_output"] != "4" {
		t.Errorf("tool_output = %v, want '4'", fields["tool_output"])
	}
	if fields["duration_ms"] != int64(50) {
		t.Errorf("duration_ms = %v, want 50", fields["duration_ms"])
	}
}

func TestToolCallResultEvalFieldsWithError(t *testing.T) {
	result := &langfuse.ToolCallResult{
		ToolName: "api-call",
		Input:    "request",
		Success:  false,
		Error:    "connection timeout",
	}

	fields := result.EvalFields()

	if fields["success"] != false {
		t.Errorf("success = %v, want false", fields["success"])
	}
	if fields["error"] != "connection timeout" {
		t.Errorf("error = %v, want 'connection timeout'", fields["error"])
	}
}

func TestEvalSpanTypes(t *testing.T) {
	tests := []struct {
		spanType langfuse.EvalSpanType
		expected string
	}{
		{langfuse.EvalSpanRetrieval, "retrieval"},
		{langfuse.EvalSpanProcessing, "processing"},
		{langfuse.EvalSpanToolCall, "tool_call"},
		{langfuse.EvalSpanReasoning, "reasoning"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.spanType) != tt.expected {
				t.Errorf("EvalSpanType = %v, want %v", tt.spanType, tt.expected)
			}
		})
	}
}
