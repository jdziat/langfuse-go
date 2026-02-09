package langfuse

import (
	"testing"
)

func TestEvaluationMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     EvaluationMode
		expected string
	}{
		{"off mode", EvaluationModeOff, ""},
		{"auto mode", EvaluationModeAuto, "auto"},
		{"ragas mode", EvaluationModeRAGAS, "ragas"},
		{"langfuse mode", EvaluationModeLangfuse, "langfuse"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.mode) != tt.expected {
				t.Errorf("EvaluationMode = %v, want %v", tt.mode, tt.expected)
			}
		})
	}
}

func TestWorkflowType(t *testing.T) {
	tests := []struct {
		name     string
		workflow WorkflowType
		expected string
	}{
		{"rag workflow", WorkflowRAG, "rag"},
		{"qa workflow", WorkflowQA, "qa"},
		{"chat workflow", WorkflowChatCompletion, "chat"},
		{"agent workflow", WorkflowAgentTask, "agent"},
		{"cot workflow", WorkflowChainOfThought, "cot"},
		{"summarization workflow", WorkflowSummarization, "summarization"},
		{"classification workflow", WorkflowClassification, "classification"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.workflow) != tt.expected {
				t.Errorf("WorkflowType = %v, want %v", tt.workflow, tt.expected)
			}
		})
	}
}

func TestWorkflowTypeGetRequiredFields(t *testing.T) {
	tests := []struct {
		workflow WorkflowType
		expected []string
	}{
		{WorkflowRAG, []string{"query", "context", "output"}},
		{WorkflowQA, []string{"query", "output"}},
		{WorkflowSummarization, []string{"input", "output"}},
		{WorkflowClassification, []string{"input", "output"}},
	}

	for _, tt := range tests {
		t.Run(string(tt.workflow), func(t *testing.T) {
			fields := tt.workflow.GetRequiredFields()
			if len(fields) != len(tt.expected) {
				t.Errorf("GetRequiredFields() returned %d fields, want %d", len(fields), len(tt.expected))
				return
			}
			for i, f := range fields {
				if f != tt.expected[i] {
					t.Errorf("GetRequiredFields()[%d] = %v, want %v", i, f, tt.expected[i])
				}
			}
		})
	}
}

func TestWorkflowTypeGetCompatibleEvaluators(t *testing.T) {
	tests := []struct {
		workflow      WorkflowType
		minEvaluators int
		mustInclude   []EvaluatorType
	}{
		{WorkflowRAG, 4, []EvaluatorType{EvaluatorFaithfulness, EvaluatorHallucination}},
		{WorkflowQA, 2, []EvaluatorType{EvaluatorAnswerRelevance, EvaluatorCorrectness}},
		{WorkflowChatCompletion, 2, []EvaluatorType{EvaluatorToxicity}},
	}

	for _, tt := range tests {
		t.Run(string(tt.workflow), func(t *testing.T) {
			evaluators := tt.workflow.GetCompatibleEvaluators()
			if len(evaluators) < tt.minEvaluators {
				t.Errorf("GetCompatibleEvaluators() returned %d evaluators, want at least %d",
					len(evaluators), tt.minEvaluators)
			}

			for _, must := range tt.mustInclude {
				found := false
				for _, e := range evaluators {
					if e == must {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("GetCompatibleEvaluators() should include %v", must)
				}
			}
		})
	}
}

func TestEvaluatorTypeGetRequiredFields(t *testing.T) {
	tests := []struct {
		evaluator EvaluatorType
		expected  []string
	}{
		{EvaluatorFaithfulness, []string{"context", "output"}},
		{EvaluatorAnswerRelevance, []string{"query", "output"}},
		{EvaluatorHallucination, []string{"query", "context", "output"}},
		{EvaluatorToxicity, []string{"output"}},
		{EvaluatorCorrectness, []string{"query", "output", "ground_truth"}},
	}

	for _, tt := range tests {
		t.Run(string(tt.evaluator), func(t *testing.T) {
			fields := tt.evaluator.GetRequiredFields()
			if len(fields) != len(tt.expected) {
				t.Errorf("GetRequiredFields() returned %d fields, want %d", len(fields), len(tt.expected))
				return
			}
			for i, f := range fields {
				if f != tt.expected[i] {
					t.Errorf("GetRequiredFields()[%d] = %v, want %v", i, f, tt.expected[i])
				}
			}
		})
	}
}

func TestDefaultEvaluationConfig(t *testing.T) {
	config := DefaultEvaluationConfig()

	if config.Mode != EvaluationModeAuto {
		t.Errorf("Mode = %v, want %v", config.Mode, EvaluationModeAuto)
	}
	if !config.AutoValidate {
		t.Error("AutoValidate should be true")
	}
	if !config.IncludeMetadata {
		t.Error("IncludeMetadata should be true")
	}
	if !config.IncludeTags {
		t.Error("IncludeTags should be true")
	}
	if !config.FlattenInput {
		t.Error("FlattenInput should be true")
	}
	if !config.FlattenOutput {
		t.Error("FlattenOutput should be true")
	}
}

func TestRAGASEvaluationConfig(t *testing.T) {
	config := RAGASEvaluationConfig()

	if config.Mode != EvaluationModeRAGAS {
		t.Errorf("Mode = %v, want %v", config.Mode, EvaluationModeRAGAS)
	}
	if config.DefaultWorkflow != WorkflowRAG {
		t.Errorf("DefaultWorkflow = %v, want %v", config.DefaultWorkflow, WorkflowRAG)
	}
	if len(config.TargetEvaluators) != 4 {
		t.Errorf("TargetEvaluators has %d items, want 4", len(config.TargetEvaluators))
	}

	// Check specific evaluators
	expectedEvaluators := map[EvaluatorType]bool{
		EvaluatorFaithfulness:     true,
		EvaluatorAnswerRelevance:  true,
		EvaluatorContextPrecision: true,
		EvaluatorContextRecall:    true,
	}

	for _, e := range config.TargetEvaluators {
		if !expectedEvaluators[e] {
			t.Errorf("Unexpected evaluator: %v", e)
		}
	}
}
