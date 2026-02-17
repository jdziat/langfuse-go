package langfuse_test

import (
	"testing"

	"github.com/jdziat/langfuse-go"
)

func TestEvalMetadataBuilder(t *testing.T) {
	builder := langfuse.NewEvalMetadataBuilder().
		WithWorkflow(langfuse.WorkflowRAG).
		WithGroundTruth(true).
		WithContext(true).
		WithOutput(true)

	metadata := builder.Build()

	if metadata.Version != langfuse.EvalMetadataVersion {
		t.Errorf("Version = %v, want %v", metadata.Version, langfuse.EvalMetadataVersion)
	}
	if metadata.WorkflowType != langfuse.WorkflowRAG {
		t.Errorf("WorkflowType = %v, want %v", metadata.WorkflowType, langfuse.WorkflowRAG)
	}
	if !metadata.HasGroundTruth {
		t.Error("HasGroundTruth should be true")
	}
	if !metadata.HasContext {
		t.Error("HasContext should be true")
	}
	if !metadata.HasOutput {
		t.Error("HasOutput should be true")
	}
}

func TestEvalMetadataBuilderMarkReady(t *testing.T) {
	builder := langfuse.NewEvalMetadataBuilder().
		WithWorkflow(langfuse.WorkflowRAG).
		WithOutput(true).
		MarkReady()

	metadata := builder.Build()

	if !metadata.Ready {
		t.Error("Ready should be true")
	}
	if metadata.ReadyAt == nil {
		t.Error("ReadyAt should be set")
	}
	if metadata.MissingFields != nil {
		t.Error("MissingFields should be nil when ready")
	}
}

func TestEvalMetadataBuilderBuildAsMap(t *testing.T) {
	builder := langfuse.NewEvalMetadataBuilder().
		WithWorkflow(langfuse.WorkflowQA).
		WithOutput(true)

	result := builder.BuildAsMap()

	if result[langfuse.EvalMetadataKey] == nil {
		t.Errorf("result should have key %s", langfuse.EvalMetadataKey)
	}

	metadata, ok := result[langfuse.EvalMetadataKey].(langfuse.EvalMetadata)
	if !ok {
		t.Fatalf("result[%s] is not EvalMetadata", langfuse.EvalMetadataKey)
	}

	if metadata.WorkflowType != langfuse.WorkflowQA {
		t.Errorf("WorkflowType = %v, want %v", metadata.WorkflowType, langfuse.WorkflowQA)
	}
}

func TestEvalMetadataBuilderGenerateEvalTags(t *testing.T) {
	tests := []struct {
		name        string
		builder     *langfuse.EvalMetadataBuilder
		mustInclude []string
	}{
		{
			name: "ready RAG workflow",
			builder: langfuse.NewEvalMetadataBuilder().
				WithWorkflow(langfuse.WorkflowRAG).
				WithGroundTruth(true).
				MarkReady(),
			mustInclude: []string{langfuse.EvalTagReady, "eval:rag", langfuse.EvalTagGroundTruth},
		},
		{
			name: "not ready workflow",
			builder: langfuse.NewEvalMetadataBuilder().
				WithWorkflow(langfuse.WorkflowQA).
				WithMissingFields("output"),
			mustInclude: []string{langfuse.EvalTagNotReady, "eval:qa"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := tt.builder.GenerateEvalTags()

			for _, must := range tt.mustInclude {
				found := false
				for _, tag := range tags {
					if tag == must {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("tags should include %s, got %v", must, tags)
				}
			}
		})
	}
}

func TestEvalTagForWorkflow(t *testing.T) {
	tests := []struct {
		workflow langfuse.WorkflowType
		expected string
	}{
		{langfuse.WorkflowRAG, "eval:rag"},
		{langfuse.WorkflowQA, "eval:qa"},
		{langfuse.WorkflowSummarization, "eval:summarization"},
	}

	for _, tt := range tests {
		t.Run(string(tt.workflow), func(t *testing.T) {
			result := langfuse.EvalTagForWorkflow(tt.workflow)
			if result != tt.expected {
				t.Errorf("EvalTagForWorkflow(%s) = %s, want %s", tt.workflow, result, tt.expected)
			}
		})
	}
}

func TestEvalTagForEvaluator(t *testing.T) {
	tests := []struct {
		evaluator langfuse.EvaluatorType
		expected  string
	}{
		{langfuse.EvaluatorFaithfulness, "eval:faithfulness"},
		{langfuse.EvaluatorHallucination, "eval:hallucination"},
		{langfuse.EvaluatorToxicity, "eval:toxicity"},
	}

	for _, tt := range tests {
		t.Run(string(tt.evaluator), func(t *testing.T) {
			result := langfuse.EvalTagForEvaluator(tt.evaluator)
			if result != tt.expected {
				t.Errorf("EvalTagForEvaluator(%s) = %s, want %s", tt.evaluator, result, tt.expected)
			}
		})
	}
}

func TestEvalState(t *testing.T) {
	state := langfuse.NewEvalState()
	state.WorkflowType = langfuse.WorkflowRAG

	// Initially not ready
	if state.IsReady() {
		t.Error("state should not be ready initially")
	}

	// Add input fields
	state.UpdateFromInput(map[string]any{
		"query":   "test query",
		"context": []string{"doc1", "doc2"},
	})

	if !state.HasContext {
		t.Error("HasContext should be true after adding context")
	}

	// Still not ready without output
	if state.IsReady() {
		t.Error("state should not be ready without output")
	}

	// Add output
	state.UpdateFromOutput(map[string]any{
		"output": "test response",
	})

	if !state.HasOutput {
		t.Error("HasOutput should be true after adding output")
	}

	// Now should be ready for RAG workflow
	if !state.IsReady() {
		t.Error("state should be ready with query, context, and output")
	}
}

func TestEvalStateGetCompatibleEvaluators(t *testing.T) {
	state := langfuse.NewEvalState()

	// Add fields for various evaluators
	state.UpdateFromInput(map[string]any{
		"query":   "test",
		"context": []string{"doc1"},
	})
	state.UpdateFromOutput(map[string]any{
		"output": "response",
	})

	evaluators := state.GetCompatibleEvaluators()

	// Should include at least Faithfulness, AnswerRelevance, Hallucination, Toxicity
	if len(evaluators) < 4 {
		t.Errorf("expected at least 4 compatible evaluators, got %d", len(evaluators))
	}

	// Check for specific evaluators
	expected := []langfuse.EvaluatorType{
		langfuse.EvaluatorFaithfulness,
		langfuse.EvaluatorAnswerRelevance,
		langfuse.EvaluatorHallucination,
		langfuse.EvaluatorToxicity,
	}

	for _, e := range expected {
		found := false
		for _, got := range evaluators {
			if got == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("evaluators should include %v", e)
		}
	}
}

func TestEvalStateGetMissingFields(t *testing.T) {
	state := langfuse.NewEvalState()
	state.TargetEvaluators = []langfuse.EvaluatorType{langfuse.EvaluatorCorrectness}

	// Correctness requires query, output, ground_truth
	state.UpdateFromInput(map[string]any{
		"query": "test",
	})

	missing := state.GetMissingFields()

	// Should be missing output and ground_truth
	if len(missing) != 2 {
		t.Errorf("expected 2 missing fields, got %d: %v", len(missing), missing)
	}

	expectedMissing := map[string]bool{
		"output":       true,
		"ground_truth": true,
	}

	for _, m := range missing {
		if !expectedMissing[m] {
			t.Errorf("unexpected missing field: %s", m)
		}
	}
}

func TestEvalStateBuildMetadata(t *testing.T) {
	state := langfuse.NewEvalState()
	state.WorkflowType = langfuse.WorkflowRAG

	state.UpdateFromInput(map[string]any{
		"query":        "test",
		"context":      []string{"doc1"},
		"ground_truth": "expected",
	})
	state.UpdateFromOutput(map[string]any{
		"output": "response",
	})

	builder := state.BuildMetadata()
	metadata := builder.Build()

	if metadata.WorkflowType != langfuse.WorkflowRAG {
		t.Errorf("WorkflowType = %v, want %v", metadata.WorkflowType, langfuse.WorkflowRAG)
	}
	if !metadata.HasGroundTruth {
		t.Error("HasGroundTruth should be true")
	}
	if !metadata.HasContext {
		t.Error("HasContext should be true")
	}
	if !metadata.HasOutput {
		t.Error("HasOutput should be true")
	}
	if !metadata.Ready {
		t.Error("Ready should be true")
	}
}

func TestValidateForEvaluator(t *testing.T) {
	tests := []struct {
		name      string
		input     any
		output    any
		evaluator langfuse.EvaluatorType
		wantErr   bool
	}{
		{
			name:      "faithfulness with all fields",
			input:     map[string]any{"context": []string{"doc1"}},
			output:    map[string]any{"output": "response"},
			evaluator: langfuse.EvaluatorFaithfulness,
			wantErr:   false,
		},
		{
			name:      "faithfulness missing context",
			input:     map[string]any{"query": "test"},
			output:    map[string]any{"output": "response"},
			evaluator: langfuse.EvaluatorFaithfulness,
			wantErr:   true,
		},
		{
			name:      "toxicity only needs output",
			input:     nil,
			output:    map[string]any{"output": "response"},
			evaluator: langfuse.EvaluatorToxicity,
			wantErr:   false,
		},
		{
			name:      "correctness needs ground_truth",
			input:     map[string]any{"query": "test"},
			output:    map[string]any{"output": "response"},
			evaluator: langfuse.EvaluatorCorrectness,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := langfuse.ValidateForEvaluator(tt.input, tt.output, tt.evaluator)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateForEvaluator() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateForWorkflow(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		output   any
		workflow langfuse.WorkflowType
		wantErr  bool
	}{
		{
			name: "RAG with all fields",
			input: map[string]any{
				"query":   "test",
				"context": []string{"doc1"},
			},
			output:   map[string]any{"output": "response"},
			workflow: langfuse.WorkflowRAG,
			wantErr:  false,
		},
		{
			name:     "RAG missing context",
			input:    map[string]any{"query": "test"},
			output:   map[string]any{"output": "response"},
			workflow: langfuse.WorkflowRAG,
			wantErr:  true,
		},
		{
			name:     "QA with all fields",
			input:    map[string]any{"query": "test"},
			output:   map[string]any{"output": "response"},
			workflow: langfuse.WorkflowQA,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := langfuse.ValidateForWorkflow(tt.input, tt.output, tt.workflow)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateForWorkflow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Note: Tests for mergeMetadata, mergeTags, and fieldAlias have been removed
// as they test unexported functions. The exported FieldAlias function is tested below.

func TestFieldAlias(t *testing.T) {
	tests := []struct {
		field    string
		expected string
	}{
		{"query", "input"},
		{"input", "query"},
		{"output", "response"},
		{"response", "output"},
		{"context", "retrieved_contexts"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			result := langfuse.FieldAlias(tt.field)
			if result != tt.expected {
				t.Errorf("FieldAlias(%s) = %s, want %s", tt.field, result, tt.expected)
			}
		})
	}
}
