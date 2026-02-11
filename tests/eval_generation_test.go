package langfuse_test

import (
	"context"
	"testing"

	langfuse "github.com/jdziat/langfuse-go"
	"github.com/jdziat/langfuse-go/langfusetest"
)

func TestEvalGenerationBuilder(t *testing.T) {
	client, _ := langfusetest.NewTestClientWithConfig(t,
		langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
	)
	ctx := context.Background()

	trace, err := client.NewTrace().Name("test-trace").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	gen, err := trace.NewEvalGeneration().
		Name("test-generation").
		Model("gpt-4").
		ForEvaluator(langfuse.EvaluatorFaithfulness, langfuse.EvaluatorHallucination).
		WithQuery("What is Go?").
		WithContext("Go is a programming language...", "Go has goroutines...").
		WithGroundTruth("Go is a language").
		Create(ctx)

	if err != nil {
		t.Fatalf("Failed to create eval generation: %v", err)
	}

	if gen.GenerationID() == "" {
		t.Error("GenerationID should not be empty")
	}

	// Check eval state
	state := gen.GetEvalState()
	if !state.HasContext {
		t.Error("HasContext should be true")
	}
	if !state.HasGroundTruth {
		t.Error("HasGroundTruth should be true")
	}
}

func TestEvalGenerationBuilderForWorkflow(t *testing.T) {
	client, _ := langfusetest.NewTestClientWithConfig(t,
		langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
	)
	ctx := context.Background()

	trace, err := client.NewTrace().Name("test-trace").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	gen, err := trace.NewEvalGeneration().
		Name("qa-generation").
		Model("gpt-4").
		ForWorkflow(langfuse.WorkflowQA).
		WithQuery("What is the capital of France?").
		Create(ctx)

	if err != nil {
		t.Fatalf("Failed to create eval generation: %v", err)
	}

	state := gen.GetEvalState()
	if state.WorkflowType != langfuse.WorkflowQA {
		t.Errorf("WorkflowType = %v, want %v", state.WorkflowType, langfuse.WorkflowQA)
	}
}

func TestEvalGenerationBuilderWithSystemPrompt(t *testing.T) {
	client, _ := langfusetest.NewTestClientWithConfig(t,
		langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
	)
	ctx := context.Background()

	trace, err := client.NewTrace().Name("test-trace").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	gen, err := trace.NewEvalGeneration().
		Name("chat-generation").
		Model("gpt-4").
		WithQuery("Hello").
		WithSystemPrompt("You are a helpful assistant.").
		Create(ctx)

	if err != nil {
		t.Fatalf("Failed to create eval generation: %v", err)
	}

	state := gen.GetEvalState()
	if !state.InputFields["system_prompt"] {
		t.Error("system_prompt should be in input fields")
	}
}

func TestEvalGenerationBuilderWithMessages(t *testing.T) {
	client, _ := langfusetest.NewTestClientWithConfig(t,
		langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
	)
	ctx := context.Background()

	trace, err := client.NewTrace().Name("test-trace").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	messages := []map[string]string{
		{"role": "system", "content": "You are helpful"},
		{"role": "user", "content": "Hello"},
	}

	gen, err := trace.NewEvalGeneration().
		Name("chat-generation").
		Model("gpt-4").
		WithMessages(messages).
		Create(ctx)

	if err != nil {
		t.Fatalf("Failed to create eval generation: %v", err)
	}

	state := gen.GetEvalState()
	if !state.InputFields["messages"] {
		t.Error("messages should be in input fields")
	}
}

func TestEvalGenerationContextIsEvalReady(t *testing.T) {
	client, _ := langfusetest.NewTestClientWithConfig(t,
		langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
	)
	ctx := context.Background()

	trace, err := client.NewTrace().Name("test-trace").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	// Create generation without output - not ready
	gen, err := trace.NewEvalGeneration().
		Name("test-gen").
		Model("gpt-4").
		WithQuery("test").
		Create(ctx)

	if err != nil {
		t.Fatalf("Failed to create eval generation: %v", err)
	}

	if gen.IsEvalReady() {
		t.Error("Should not be eval ready without output")
	}
}

func TestEvalGenerationContextGetCompatibleEvaluators(t *testing.T) {
	client, _ := langfusetest.NewTestClientWithConfig(t,
		langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
	)
	ctx := context.Background()

	trace, err := client.NewTrace().Name("test-trace").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	gen, err := trace.NewEvalGeneration().
		Name("test-gen").
		Model("gpt-4").
		WithQuery("test query").
		WithContext("test context").
		Output("test output").
		Create(ctx)

	if err != nil {
		t.Fatalf("Failed to create eval generation: %v", err)
	}

	evaluators := gen.GetCompatibleEvaluators()

	// Should include faithfulness (context + output) and answer_relevance (query + output)
	if len(evaluators) < 2 {
		t.Errorf("Expected at least 2 compatible evaluators, got %d", len(evaluators))
	}
}

func TestEvalGenerationContextValidateForEvaluator(t *testing.T) {
	client, _ := langfusetest.NewTestClientWithConfig(t,
		langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
	)
	ctx := context.Background()

	trace, err := client.NewTrace().Name("test-trace").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	// Create generation with context and output (enough for faithfulness)
	gen, err := trace.NewEvalGeneration().
		Name("test-gen").
		Model("gpt-4").
		WithContext("test context").
		Output("test output").
		Create(ctx)

	if err != nil {
		t.Fatalf("Failed to create eval generation: %v", err)
	}

	// Should pass faithfulness (requires context + output)
	err = gen.ValidateForEvaluator(langfuse.EvaluatorFaithfulness)
	if err != nil {
		t.Errorf("ValidateForEvaluator(Faithfulness) should pass: %v", err)
	}

	// Should fail correctness (requires ground_truth)
	err = gen.ValidateForEvaluator(langfuse.EvaluatorCorrectness)
	if err == nil {
		t.Error("ValidateForEvaluator(Correctness) should fail without ground_truth")
	}
}

func TestEvalGenerationContextCompleteWithEvaluation(t *testing.T) {
	client, _ := langfusetest.NewTestClientWithConfig(t,
		langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
	)
	ctx := context.Background()

	trace, err := client.NewTrace().Name("test-trace").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	gen, err := trace.NewEvalGeneration().
		Name("test-gen").
		Model("gpt-4").
		WithQuery("What is Go?").
		WithContext("Go is a programming language").
		Create(ctx)

	if err != nil {
		t.Fatalf("Failed to create eval generation: %v", err)
	}

	// Complete with evaluation result
	result := gen.CompleteWithEvaluation(ctx, &langfuse.EvalGenerationResult{
		Output:       "Go is a statically typed language...",
		InputTokens:  100,
		OutputTokens: 50,
		Model:        "gpt-4",
		Confidence:   0.95,
		Citations:    []string{"go-docs.md"},
	})

	if result.Error != nil {
		t.Errorf("CompleteWithEvaluation failed: %v", result.Error)
	}

	// After completion, should be eval ready
	if !gen.IsEvalReady() {
		t.Error("Should be eval ready after completion with output")
	}
}

func TestEvalGenerationResultEvalFields(t *testing.T) {
	result := &langfuse.EvalGenerationResult{
		Output:     "test output",
		Citations:  []string{"doc1.md"},
		Confidence: 0.9,
		Reasoning:  "because...",
	}

	fields := result.EvalFields()

	if fields["output"] != "test output" {
		t.Errorf("output = %v, want 'test output'", fields["output"])
	}
	if fields["confidence"] != 0.9 {
		t.Errorf("confidence = %v, want 0.9", fields["confidence"])
	}
	if fields["reasoning"] != "because..." {
		t.Errorf("reasoning = %v, want 'because...'", fields["reasoning"])
	}
}

func TestEvalGenerationResultToStandardOutput(t *testing.T) {
	result := &langfuse.EvalGenerationResult{
		Output:     "test output",
		Citations:  []string{"doc1.md"},
		Confidence: 0.9,
	}

	stdOutput := result.ToStandardOutput()

	if stdOutput.Output != "test output" {
		t.Errorf("Output = %v, want 'test output'", stdOutput.Output)
	}
	if len(stdOutput.Citations) != 1 || stdOutput.Citations[0] != "doc1.md" {
		t.Errorf("Citations = %v, want [doc1.md]", stdOutput.Citations)
	}
	if stdOutput.Confidence != 0.9 {
		t.Errorf("Confidence = %v, want 0.9", stdOutput.Confidence)
	}
}

func TestEvalGenerationBuilderChaining(t *testing.T) {
	client, _ := langfusetest.NewTestClientWithConfig(t,
		langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
	)
	ctx := context.Background()

	trace, err := client.NewTrace().Name("test-trace").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	// Test all chainable methods
	gen, err := trace.NewEvalGeneration().
		ID("custom-id").
		Name("test-gen").
		Model("gpt-4").
		ModelParameters(langfuse.Metadata{"temperature": 0.7}).
		Input(map[string]any{"custom": "input"}).
		Output("custom output").
		Metadata(langfuse.Metadata{"key": "value"}).
		Level(langfuse.ObservationLevelDebug).
		PromptName("test-prompt").
		PromptVersion(1).
		Environment("test").
		ForEvaluator(langfuse.EvaluatorToxicity).
		ForWorkflow(langfuse.WorkflowChatCompletion).
		Create(ctx)

	if err != nil {
		t.Fatalf("Failed to create eval generation with all options: %v", err)
	}

	if gen.GenerationID() != "custom-id" {
		t.Errorf("GenerationID = %v, want 'custom-id'", gen.GenerationID())
	}
}
