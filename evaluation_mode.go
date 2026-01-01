package langfuse

import (
	"time"
)

// EvaluationMode controls how traces are structured for LLM-as-a-Judge evaluation.
// When enabled, the SDK automatically flattens and normalizes data to enable
// direct field access in Langfuse evaluators without JSONPath configuration.
type EvaluationMode string

const (
	// EvaluationModeOff disables automatic evaluation structuring (default).
	// Input and output are stored as-is without modification.
	EvaluationModeOff EvaluationMode = ""

	// EvaluationModeAuto automatically structures data for common evaluators.
	// - Flattens nested structures for direct field access
	// - Adds evaluation metadata with compatible evaluator list
	// - Adds evaluation-ready tags for filtering
	EvaluationModeAuto EvaluationMode = "auto"

	// EvaluationModeRAGAS structures data specifically for RAGAS metrics.
	// Uses RAGAS field naming: user_input, retrieved_contexts, response
	EvaluationModeRAGAS EvaluationMode = "ragas"

	// EvaluationModeLangfuse structures data for Langfuse managed evaluators.
	// Uses Langfuse field naming: input, output, context, ground_truth
	EvaluationModeLangfuse EvaluationMode = "langfuse"
)

// WorkflowType represents the type of LLM workflow being traced.
// This helps the SDK understand what evaluation fields to expect and structure.
type WorkflowType string

const (
	// WorkflowRAG is a Retrieval-Augmented Generation workflow.
	// Expected fields: query, context, output, ground_truth (optional)
	WorkflowRAG WorkflowType = "rag"

	// WorkflowQA is a Question-Answering workflow.
	// Expected fields: query, output, ground_truth (optional)
	WorkflowQA WorkflowType = "qa"

	// WorkflowChatCompletion is a standard chat completion workflow.
	// Expected fields: messages, output
	WorkflowChatCompletion WorkflowType = "chat"

	// WorkflowAgentTask is an agent task execution workflow.
	// Expected fields: task, steps, output
	WorkflowAgentTask WorkflowType = "agent"

	// WorkflowChainOfThought is a chain-of-thought reasoning workflow.
	// Expected fields: query, reasoning_steps, output
	WorkflowChainOfThought WorkflowType = "cot"

	// WorkflowSummarization is a text summarization workflow.
	// Expected fields: input, output, ground_truth (optional)
	WorkflowSummarization WorkflowType = "summarization"

	// WorkflowClassification is a text classification workflow.
	// Expected fields: input, classes, output, ground_truth (optional)
	WorkflowClassification WorkflowType = "classification"
)

// EvaluatorType represents built-in evaluator types that the SDK can optimize for.
type EvaluatorType string

const (
	// EvaluatorFaithfulness checks if output is faithful to context.
	// Required: context, output
	EvaluatorFaithfulness EvaluatorType = "faithfulness"

	// EvaluatorAnswerRelevance checks if answer is relevant to query.
	// Required: query, output
	EvaluatorAnswerRelevance EvaluatorType = "answer_relevance"

	// EvaluatorContextPrecision checks if context is precise and relevant.
	// Required: query, context, ground_truth
	EvaluatorContextPrecision EvaluatorType = "context_precision"

	// EvaluatorContextRecall checks if context contains all relevant info.
	// Required: query, context, ground_truth
	EvaluatorContextRecall EvaluatorType = "context_recall"

	// EvaluatorHallucination detects hallucinated content.
	// Required: query, context, output
	EvaluatorHallucination EvaluatorType = "hallucination"

	// EvaluatorToxicity detects toxic content.
	// Required: output
	EvaluatorToxicity EvaluatorType = "toxicity"

	// EvaluatorCorrectness checks answer correctness.
	// Required: query, output, ground_truth
	EvaluatorCorrectness EvaluatorType = "correctness"
)

// EvaluationConfig holds configuration for evaluation mode.
type EvaluationConfig struct {
	// Mode is the evaluation mode to use.
	Mode EvaluationMode

	// DefaultWorkflow is the default workflow type for traces.
	// Can be overridden per-trace.
	DefaultWorkflow WorkflowType

	// TargetEvaluators is the list of evaluators to optimize for.
	// When set, the SDK ensures all required fields are present.
	TargetEvaluators []EvaluatorType

	// AutoValidate enables automatic validation before trace completion.
	// If true, traces are validated against target evaluators.
	AutoValidate bool

	// IncludeMetadata adds evaluation metadata to traces.
	// This includes compatible evaluators, workflow type, and readiness status.
	IncludeMetadata bool

	// IncludeTags adds evaluation-ready tags to traces.
	// Tags include: eval:ready, eval:<workflow>, eval:has-ground-truth
	IncludeTags bool

	// FlattenInput flattens structured input for direct field access.
	// When true, nested fields are hoisted to top level.
	FlattenInput bool

	// FlattenOutput flattens structured output for direct field access.
	FlattenOutput bool
}

// DefaultEvaluationConfig returns a sensible default evaluation configuration.
func DefaultEvaluationConfig() *EvaluationConfig {
	return &EvaluationConfig{
		Mode:            EvaluationModeAuto,
		AutoValidate:    true,
		IncludeMetadata: true,
		IncludeTags:     true,
		FlattenInput:    true,
		FlattenOutput:   true,
	}
}

// RAGASEvaluationConfig returns configuration optimized for RAGAS evaluators.
func RAGASEvaluationConfig() *EvaluationConfig {
	return &EvaluationConfig{
		Mode:            EvaluationModeRAGAS,
		DefaultWorkflow: WorkflowRAG,
		TargetEvaluators: []EvaluatorType{
			EvaluatorFaithfulness,
			EvaluatorAnswerRelevance,
			EvaluatorContextPrecision,
			EvaluatorContextRecall,
		},
		AutoValidate:    true,
		IncludeMetadata: true,
		IncludeTags:     true,
		FlattenInput:    true,
		FlattenOutput:   true,
	}
}

// EvalMetadata contains evaluation-specific metadata added to traces.
type EvalMetadata struct {
	// Version is the evaluation metadata schema version.
	Version string `json:"version"`

	// WorkflowType is the type of workflow being traced.
	WorkflowType WorkflowType `json:"workflow_type,omitempty"`

	// HasGroundTruth indicates if ground truth was provided.
	HasGroundTruth bool `json:"has_ground_truth"`

	// HasContext indicates if context/retrieved documents were provided.
	HasContext bool `json:"has_context"`

	// HasOutput indicates if output has been set.
	HasOutput bool `json:"has_output"`

	// CompatibleEvaluators lists evaluators this trace can be used with.
	CompatibleEvaluators []EvaluatorType `json:"compatible_evaluators,omitempty"`

	// Ready indicates if the trace is ready for evaluation.
	Ready bool `json:"ready"`

	// ReadyAt is when the trace became evaluation-ready.
	ReadyAt *time.Time `json:"ready_at,omitempty"`

	// MissingFields lists fields that are still required.
	MissingFields []string `json:"missing_fields,omitempty"`
}

// EvalMetadataKey is the metadata key used for evaluation metadata.
const EvalMetadataKey = "_langfuse_eval_metadata"

// evaluatorRequirements maps evaluators to their required fields.
var evaluatorRequirements = map[EvaluatorType][]string{
	EvaluatorFaithfulness:     {"context", "output"},
	EvaluatorAnswerRelevance:  {"query", "output"},
	EvaluatorContextPrecision: {"query", "context", "ground_truth"},
	EvaluatorContextRecall:    {"query", "context", "ground_truth"},
	EvaluatorHallucination:    {"query", "context", "output"},
	EvaluatorToxicity:         {"output"},
	EvaluatorCorrectness:      {"query", "output", "ground_truth"},
}

// GetRequiredFields returns the required fields for an evaluator.
func (e EvaluatorType) GetRequiredFields() []string {
	if fields, ok := evaluatorRequirements[e]; ok {
		return fields
	}
	return nil
}

// workflowFields maps workflow types to their expected fields.
var workflowFields = map[WorkflowType]struct {
	required []string
	optional []string
}{
	WorkflowRAG: {
		required: []string{"query", "context", "output"},
		optional: []string{"ground_truth", "citations"},
	},
	WorkflowQA: {
		required: []string{"query", "output"},
		optional: []string{"ground_truth", "context"},
	},
	WorkflowChatCompletion: {
		required: []string{"messages", "output"},
		optional: []string{"system_prompt"},
	},
	WorkflowAgentTask: {
		required: []string{"task", "output"},
		optional: []string{"steps", "tools_used"},
	},
	WorkflowChainOfThought: {
		required: []string{"query", "output"},
		optional: []string{"reasoning_steps", "intermediate_outputs"},
	},
	WorkflowSummarization: {
		required: []string{"input", "output"},
		optional: []string{"ground_truth", "max_length"},
	},
	WorkflowClassification: {
		required: []string{"input", "output"},
		optional: []string{"classes", "ground_truth", "confidence"},
	},
}

// GetRequiredFields returns the required fields for a workflow type.
func (w WorkflowType) GetRequiredFields() []string {
	if fields, ok := workflowFields[w]; ok {
		return fields.required
	}
	return nil
}

// GetOptionalFields returns the optional fields for a workflow type.
func (w WorkflowType) GetOptionalFields() []string {
	if fields, ok := workflowFields[w]; ok {
		return fields.optional
	}
	return nil
}

// GetCompatibleEvaluators returns evaluators compatible with a workflow type.
func (w WorkflowType) GetCompatibleEvaluators() []EvaluatorType {
	switch w {
	case WorkflowRAG:
		return []EvaluatorType{
			EvaluatorFaithfulness,
			EvaluatorAnswerRelevance,
			EvaluatorContextPrecision,
			EvaluatorContextRecall,
			EvaluatorHallucination,
		}
	case WorkflowQA:
		return []EvaluatorType{
			EvaluatorAnswerRelevance,
			EvaluatorCorrectness,
		}
	case WorkflowChatCompletion:
		return []EvaluatorType{
			EvaluatorToxicity,
			EvaluatorAnswerRelevance,
		}
	case WorkflowSummarization:
		return []EvaluatorType{
			EvaluatorFaithfulness,
		}
	case WorkflowClassification:
		return []EvaluatorType{
			EvaluatorCorrectness,
		}
	default:
		return []EvaluatorType{EvaluatorToxicity}
	}
}
