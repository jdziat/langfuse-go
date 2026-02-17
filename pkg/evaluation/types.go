// Package evaluation provides types and utilities for LLM evaluation workflows.
//
// This package supports structured evaluation with:
//   - Workflow types (RAG, Q&A, summarization, etc.)
//   - Evaluator types (faithfulness, relevance, hallucination, etc.)
//   - Input/output flattening for direct field access in evaluators
//   - Metadata builders for evaluation-ready traces
package evaluation

import (
	"time"
)

// Mode controls how traces are structured for LLM-as-a-Judge evaluation.
type Mode string

const (
	// ModeOff disables automatic evaluation structuring (default).
	ModeOff Mode = ""

	// ModeAuto automatically structures data for common evaluators.
	ModeAuto Mode = "auto"

	// ModeRAGAS structures data specifically for RAGAS metrics.
	ModeRAGAS Mode = "ragas"

	// ModeLangfuse structures data for Langfuse managed evaluators.
	ModeLangfuse Mode = "langfuse"
)

// WorkflowType represents the type of LLM workflow being traced.
type WorkflowType string

const (
	// WorkflowRAG is a Retrieval-Augmented Generation workflow.
	WorkflowRAG WorkflowType = "rag"

	// WorkflowQA is a Question-Answering workflow.
	WorkflowQA WorkflowType = "qa"

	// WorkflowChatCompletion is a standard chat completion workflow.
	WorkflowChatCompletion WorkflowType = "chat"

	// WorkflowAgentTask is an agent task execution workflow.
	WorkflowAgentTask WorkflowType = "agent"

	// WorkflowChainOfThought is a chain-of-thought reasoning workflow.
	WorkflowChainOfThought WorkflowType = "cot"

	// WorkflowSummarization is a text summarization workflow.
	WorkflowSummarization WorkflowType = "summarization"

	// WorkflowClassification is a text classification workflow.
	WorkflowClassification WorkflowType = "classification"
)

// EvaluatorType represents built-in evaluator types.
type EvaluatorType string

const (
	// EvaluatorFaithfulness checks if output is faithful to context.
	EvaluatorFaithfulness EvaluatorType = "faithfulness"

	// EvaluatorAnswerRelevance checks if answer is relevant to query.
	EvaluatorAnswerRelevance EvaluatorType = "answer_relevance"

	// EvaluatorContextPrecision checks if context is precise and relevant.
	EvaluatorContextPrecision EvaluatorType = "context_precision"

	// EvaluatorContextRecall checks if context contains all relevant info.
	EvaluatorContextRecall EvaluatorType = "context_recall"

	// EvaluatorHallucination detects hallucinated content.
	EvaluatorHallucination EvaluatorType = "hallucination"

	// EvaluatorToxicity detects toxic content.
	EvaluatorToxicity EvaluatorType = "toxicity"

	// EvaluatorCorrectness checks answer correctness.
	EvaluatorCorrectness EvaluatorType = "correctness"
)

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

// Config holds configuration for evaluation mode.
type Config struct {
	// Mode is the evaluation mode to use.
	Mode Mode

	// DefaultWorkflow is the default workflow type for traces.
	DefaultWorkflow WorkflowType

	// TargetEvaluators is the list of evaluators to optimize for.
	TargetEvaluators []EvaluatorType

	// AutoValidate enables automatic validation before trace completion.
	AutoValidate bool

	// IncludeMetadata adds evaluation metadata to traces.
	IncludeMetadata bool

	// IncludeTags adds evaluation-ready tags to traces.
	IncludeTags bool

	// FlattenInput flattens structured input for direct field access.
	FlattenInput bool

	// FlattenOutput flattens structured output for direct field access.
	FlattenOutput bool
}

// DefaultConfig returns a sensible default evaluation configuration.
func DefaultConfig() *Config {
	return &Config{
		Mode:            ModeAuto,
		AutoValidate:    true,
		IncludeMetadata: true,
		IncludeTags:     true,
		FlattenInput:    true,
		FlattenOutput:   true,
	}
}

// RAGASConfig returns configuration optimized for RAGAS evaluators.
func RAGASConfig() *Config {
	return &Config{
		Mode:            ModeRAGAS,
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

// Metadata contains evaluation-specific metadata added to traces.
type Metadata struct {
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

// MetadataKey is the metadata key used for evaluation metadata.
const MetadataKey = "_langfuse_eval_metadata"

// MetadataVersion is the current version of the evaluation metadata schema.
const MetadataVersion = "1.0"

// Tag prefixes and constants.
const (
	TagPrefix      = "eval:"
	TagReady       = "eval:ready"
	TagNotReady    = "eval:not-ready"
	TagGroundTruth = "eval:has-ground-truth"
)

// TagForWorkflow returns the evaluation tag for a workflow type.
func TagForWorkflow(w WorkflowType) string {
	return TagPrefix + string(w)
}

// TagForEvaluator returns the evaluation tag for an evaluator type.
func TagForEvaluator(e EvaluatorType) string {
	return TagPrefix + string(e)
}

// SpanType represents the type of evaluation span.
type SpanType string

const (
	// SpanRetrieval is a retrieval/search span.
	SpanRetrieval SpanType = "retrieval"

	// SpanProcessing is a data processing span.
	SpanProcessing SpanType = "processing"

	// SpanToolCall is a tool call span.
	SpanToolCall SpanType = "tool_call"

	// SpanReasoning is a reasoning/thinking span.
	SpanReasoning SpanType = "reasoning"
)
