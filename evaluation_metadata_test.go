package langfuse

import (
	"reflect"
	"testing"
)

func TestEvalMetadataBuilder(t *testing.T) {
	builder := NewEvalMetadataBuilder().
		WithWorkflow(WorkflowRAG).
		WithGroundTruth(true).
		WithContext(true).
		WithOutput(true)

	metadata := builder.Build()

	if metadata.Version != EvalMetadataVersion {
		t.Errorf("Version = %v, want %v", metadata.Version, EvalMetadataVersion)
	}
	if metadata.WorkflowType != WorkflowRAG {
		t.Errorf("WorkflowType = %v, want %v", metadata.WorkflowType, WorkflowRAG)
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
	builder := NewEvalMetadataBuilder().
		WithWorkflow(WorkflowRAG).
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
	builder := NewEvalMetadataBuilder().
		WithWorkflow(WorkflowQA).
		WithOutput(true)

	result := builder.BuildAsMap()

	if result[EvalMetadataKey] == nil {
		t.Errorf("result should have key %s", EvalMetadataKey)
	}

	metadata, ok := result[EvalMetadataKey].(EvalMetadata)
	if !ok {
		t.Fatalf("result[%s] is not EvalMetadata", EvalMetadataKey)
	}

	if metadata.WorkflowType != WorkflowQA {
		t.Errorf("WorkflowType = %v, want %v", metadata.WorkflowType, WorkflowQA)
	}
}

func TestEvalMetadataBuilderGenerateEvalTags(t *testing.T) {
	tests := []struct {
		name        string
		builder     *EvalMetadataBuilder
		mustInclude []string
	}{
		{
			name: "ready RAG workflow",
			builder: NewEvalMetadataBuilder().
				WithWorkflow(WorkflowRAG).
				WithGroundTruth(true).
				MarkReady(),
			mustInclude: []string{EvalTagReady, "eval:rag", EvalTagGroundTruth},
		},
		{
			name: "not ready workflow",
			builder: NewEvalMetadataBuilder().
				WithWorkflow(WorkflowQA).
				WithMissingFields("output"),
			mustInclude: []string{EvalTagNotReady, "eval:qa"},
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
		workflow WorkflowType
		expected string
	}{
		{WorkflowRAG, "eval:rag"},
		{WorkflowQA, "eval:qa"},
		{WorkflowSummarization, "eval:summarization"},
	}

	for _, tt := range tests {
		t.Run(string(tt.workflow), func(t *testing.T) {
			result := EvalTagForWorkflow(tt.workflow)
			if result != tt.expected {
				t.Errorf("EvalTagForWorkflow(%s) = %s, want %s", tt.workflow, result, tt.expected)
			}
		})
	}
}

func TestEvalTagForEvaluator(t *testing.T) {
	tests := []struct {
		evaluator EvaluatorType
		expected  string
	}{
		{EvaluatorFaithfulness, "eval:faithfulness"},
		{EvaluatorHallucination, "eval:hallucination"},
		{EvaluatorToxicity, "eval:toxicity"},
	}

	for _, tt := range tests {
		t.Run(string(tt.evaluator), func(t *testing.T) {
			result := EvalTagForEvaluator(tt.evaluator)
			if result != tt.expected {
				t.Errorf("EvalTagForEvaluator(%s) = %s, want %s", tt.evaluator, result, tt.expected)
			}
		})
	}
}

func TestEvalState(t *testing.T) {
	state := NewEvalState()
	state.WorkflowType = WorkflowRAG

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
	state := NewEvalState()

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
	expected := []EvaluatorType{
		EvaluatorFaithfulness,
		EvaluatorAnswerRelevance,
		EvaluatorHallucination,
		EvaluatorToxicity,
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
	state := NewEvalState()
	state.TargetEvaluators = []EvaluatorType{EvaluatorCorrectness}

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
	state := NewEvalState()
	state.WorkflowType = WorkflowRAG

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

	if metadata.WorkflowType != WorkflowRAG {
		t.Errorf("WorkflowType = %v, want %v", metadata.WorkflowType, WorkflowRAG)
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
		evaluator EvaluatorType
		wantErr   bool
	}{
		{
			name:      "faithfulness with all fields",
			input:     map[string]any{"context": []string{"doc1"}},
			output:    map[string]any{"output": "response"},
			evaluator: EvaluatorFaithfulness,
			wantErr:   false,
		},
		{
			name:      "faithfulness missing context",
			input:     map[string]any{"query": "test"},
			output:    map[string]any{"output": "response"},
			evaluator: EvaluatorFaithfulness,
			wantErr:   true,
		},
		{
			name:      "toxicity only needs output",
			input:     nil,
			output:    map[string]any{"output": "response"},
			evaluator: EvaluatorToxicity,
			wantErr:   false,
		},
		{
			name:      "correctness needs ground_truth",
			input:     map[string]any{"query": "test"},
			output:    map[string]any{"output": "response"},
			evaluator: EvaluatorCorrectness,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateForEvaluator(tt.input, tt.output, tt.evaluator)
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
		workflow WorkflowType
		wantErr  bool
	}{
		{
			name: "RAG with all fields",
			input: map[string]any{
				"query":   "test",
				"context": []string{"doc1"},
			},
			output:   map[string]any{"output": "response"},
			workflow: WorkflowRAG,
			wantErr:  false,
		},
		{
			name:     "RAG missing context",
			input:    map[string]any{"query": "test"},
			output:   map[string]any{"output": "response"},
			workflow: WorkflowRAG,
			wantErr:  true,
		},
		{
			name:     "QA with all fields",
			input:    map[string]any{"query": "test"},
			output:   map[string]any{"output": "response"},
			workflow: WorkflowQA,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateForWorkflow(tt.input, tt.output, tt.workflow)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateForWorkflow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMergeMetadata(t *testing.T) {
	existing := Metadata{"key1": "value1"}
	evalMeta := map[string]any{EvalMetadataKey: EvalMetadata{Version: "1.0"}}

	result := mergeMetadata(existing, evalMeta)

	if result["key1"] != "value1" {
		t.Errorf("existing key should be preserved")
	}
	if result[EvalMetadataKey] == nil {
		t.Errorf("eval metadata should be added")
	}
}

func TestMergeMetadataNilExisting(t *testing.T) {
	evalMeta := map[string]any{EvalMetadataKey: EvalMetadata{Version: "1.0"}}

	result := mergeMetadata(nil, evalMeta)

	if result[EvalMetadataKey] == nil {
		t.Errorf("eval metadata should be added")
	}
}

func TestMergeTags(t *testing.T) {
	existing := []string{"tag1", "tag2"}
	evalTags := []string{"eval:ready", "tag2", "eval:rag"}

	result := mergeTags(existing, evalTags)

	expected := []string{"tag1", "tag2", "eval:ready", "eval:rag"}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("mergeTags = %v, want %v", result, expected)
	}
}

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
			result := fieldAlias(tt.field)
			if result != tt.expected {
				t.Errorf("fieldAlias(%s) = %s, want %s", tt.field, result, tt.expected)
			}
		})
	}
}
