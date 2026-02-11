package langfuse_test

import (
	"testing"

	"github.com/jdziat/langfuse-go"
)

func TestEvaluationMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     langfuse.EvaluationMode
		expected string
	}{
		{"off mode", langfuse.EvaluationModeOff, ""},
		{"auto mode", langfuse.EvaluationModeAuto, "auto"},
		{"ragas mode", langfuse.EvaluationModeRAGAS, "ragas"},
		{"langfuse mode", langfuse.EvaluationModeLangfuse, "langfuse"},
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
		workflow langfuse.WorkflowType
		expected string
	}{
		{"rag workflow", langfuse.WorkflowRAG, "rag"},
		{"qa workflow", langfuse.WorkflowQA, "qa"},
		{"chat workflow", langfuse.WorkflowChatCompletion, "chat"},
		{"agent workflow", langfuse.WorkflowAgentTask, "agent"},
		{"cot workflow", langfuse.WorkflowChainOfThought, "cot"},
		{"summarization workflow", langfuse.WorkflowSummarization, "summarization"},
		{"classification workflow", langfuse.WorkflowClassification, "classification"},
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
		workflow langfuse.WorkflowType
		expected []string
	}{
		{langfuse.WorkflowRAG, []string{"query", "context", "output"}},
		{langfuse.WorkflowQA, []string{"query", "output"}},
		{langfuse.WorkflowSummarization, []string{"input", "output"}},
		{langfuse.WorkflowClassification, []string{"input", "output"}},
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
		workflow      langfuse.WorkflowType
		minEvaluators int
		mustInclude   []langfuse.EvaluatorType
	}{
		{langfuse.WorkflowRAG, 4, []langfuse.EvaluatorType{langfuse.EvaluatorFaithfulness, langfuse.EvaluatorHallucination}},
		{langfuse.WorkflowQA, 2, []langfuse.EvaluatorType{langfuse.EvaluatorAnswerRelevance, langfuse.EvaluatorCorrectness}},
		{langfuse.WorkflowChatCompletion, 2, []langfuse.EvaluatorType{langfuse.EvaluatorToxicity}},
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
		evaluator langfuse.EvaluatorType
		expected  []string
	}{
		{langfuse.EvaluatorFaithfulness, []string{"context", "output"}},
		{langfuse.EvaluatorAnswerRelevance, []string{"query", "output"}},
		{langfuse.EvaluatorHallucination, []string{"query", "context", "output"}},
		{langfuse.EvaluatorToxicity, []string{"output"}},
		{langfuse.EvaluatorCorrectness, []string{"query", "output", "ground_truth"}},
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
	config := langfuse.DefaultEvaluationConfig()

	if config.Mode != langfuse.EvaluationModeAuto {
		t.Errorf("Mode = %v, want %v", config.Mode, langfuse.EvaluationModeAuto)
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
	config := langfuse.RAGASEvaluationConfig()

	if config.Mode != langfuse.EvaluationModeRAGAS {
		t.Errorf("Mode = %v, want %v", config.Mode, langfuse.EvaluationModeRAGAS)
	}
	if config.DefaultWorkflow != langfuse.WorkflowRAG {
		t.Errorf("DefaultWorkflow = %v, want %v", config.DefaultWorkflow, langfuse.WorkflowRAG)
	}
	if len(config.TargetEvaluators) != 4 {
		t.Errorf("TargetEvaluators has %d items, want 4", len(config.TargetEvaluators))
	}

	// Check specific evaluators
	expectedEvaluators := map[langfuse.EvaluatorType]bool{
		langfuse.EvaluatorFaithfulness:     true,
		langfuse.EvaluatorAnswerRelevance:  true,
		langfuse.EvaluatorContextPrecision: true,
		langfuse.EvaluatorContextRecall:    true,
	}

	for _, e := range config.TargetEvaluators {
		if !expectedEvaluators[e] {
			t.Errorf("Unexpected evaluator: %v", e)
		}
	}
}
