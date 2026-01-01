package langfuse

import (
	"fmt"
	"time"
)

// EvalMetadataVersion is the current version of the evaluation metadata schema.
const EvalMetadataVersion = "1.0"

// Evaluation tag prefixes.
const (
	EvalTagPrefix     = "eval:"
	EvalTagReady      = "eval:ready"
	EvalTagNotReady   = "eval:not-ready"
	EvalTagGroundTruth = "eval:has-ground-truth"
)

// EvalTagForWorkflow returns the evaluation tag for a workflow type.
func EvalTagForWorkflow(w WorkflowType) string {
	return EvalTagPrefix + string(w)
}

// EvalTagForEvaluator returns the evaluation tag for an evaluator type.
func EvalTagForEvaluator(e EvaluatorType) string {
	return EvalTagPrefix + string(e)
}

// EvalMetadataBuilder helps build evaluation metadata.
type EvalMetadataBuilder struct {
	metadata EvalMetadata
}

// NewEvalMetadataBuilder creates a new evaluation metadata builder.
func NewEvalMetadataBuilder() *EvalMetadataBuilder {
	return &EvalMetadataBuilder{
		metadata: EvalMetadata{
			Version: EvalMetadataVersion,
		},
	}
}

// WithWorkflow sets the workflow type.
func (b *EvalMetadataBuilder) WithWorkflow(w WorkflowType) *EvalMetadataBuilder {
	b.metadata.WorkflowType = w
	b.metadata.CompatibleEvaluators = w.GetCompatibleEvaluators()
	return b
}

// WithGroundTruth marks that ground truth is present.
func (b *EvalMetadataBuilder) WithGroundTruth(hasGroundTruth bool) *EvalMetadataBuilder {
	b.metadata.HasGroundTruth = hasGroundTruth
	return b
}

// WithContext marks that context is present.
func (b *EvalMetadataBuilder) WithContext(hasContext bool) *EvalMetadataBuilder {
	b.metadata.HasContext = hasContext
	return b
}

// WithOutput marks that output is present.
func (b *EvalMetadataBuilder) WithOutput(hasOutput bool) *EvalMetadataBuilder {
	b.metadata.HasOutput = hasOutput
	return b
}

// WithCompatibleEvaluators sets the compatible evaluators.
func (b *EvalMetadataBuilder) WithCompatibleEvaluators(evaluators ...EvaluatorType) *EvalMetadataBuilder {
	b.metadata.CompatibleEvaluators = evaluators
	return b
}

// WithMissingFields sets the missing fields list.
func (b *EvalMetadataBuilder) WithMissingFields(fields ...string) *EvalMetadataBuilder {
	b.metadata.MissingFields = fields
	return b
}

// MarkReady marks the trace as evaluation-ready.
func (b *EvalMetadataBuilder) MarkReady() *EvalMetadataBuilder {
	b.metadata.Ready = true
	now := time.Now()
	b.metadata.ReadyAt = &now
	b.metadata.MissingFields = nil
	return b
}

// Build returns the built metadata.
func (b *EvalMetadataBuilder) Build() EvalMetadata {
	return b.metadata
}

// BuildAsMap returns the metadata as a map suitable for trace metadata.
func (b *EvalMetadataBuilder) BuildAsMap() map[string]any {
	return map[string]any{
		EvalMetadataKey: b.metadata,
	}
}

// GenerateEvalTags generates evaluation tags based on the current state.
func (b *EvalMetadataBuilder) GenerateEvalTags() []string {
	tags := make([]string, 0, 5)

	// Ready status
	if b.metadata.Ready {
		tags = append(tags, EvalTagReady)
	} else {
		tags = append(tags, EvalTagNotReady)
	}

	// Workflow type
	if b.metadata.WorkflowType != "" {
		tags = append(tags, EvalTagForWorkflow(b.metadata.WorkflowType))
	}

	// Ground truth
	if b.metadata.HasGroundTruth {
		tags = append(tags, EvalTagGroundTruth)
	}

	// Compatible evaluators (limit to avoid too many tags)
	for i, e := range b.metadata.CompatibleEvaluators {
		if i >= 3 { // Max 3 evaluator tags
			break
		}
		tags = append(tags, EvalTagForEvaluator(e))
	}

	return tags
}

// EvalState tracks the current evaluation state for a trace.
type EvalState struct {
	// WorkflowType is the type of workflow.
	WorkflowType WorkflowType

	// InputFields tracks which input fields are present.
	InputFields map[string]bool

	// OutputFields tracks which output fields are present.
	OutputFields map[string]bool

	// HasGroundTruth indicates if ground truth was set.
	HasGroundTruth bool

	// HasContext indicates if context was set.
	HasContext bool

	// HasOutput indicates if output was set.
	HasOutput bool

	// TargetEvaluators are the evaluators being optimized for.
	TargetEvaluators []EvaluatorType
}

// NewEvalState creates a new evaluation state tracker.
func NewEvalState() *EvalState {
	return &EvalState{
		InputFields:  make(map[string]bool),
		OutputFields: make(map[string]bool),
	}
}

// UpdateFromInput updates state from input data.
func (s *EvalState) UpdateFromInput(data any) {
	presence := extractFieldPresence(data)
	for k, v := range presence {
		s.InputFields[k] = v
	}

	// Check for known fields
	if s.InputFields["ground_truth"] {
		s.HasGroundTruth = true
	}
	if s.InputFields["context"] || s.InputFields["retrieved_contexts"] {
		s.HasContext = true
	}
}

// UpdateFromOutput updates state from output data.
func (s *EvalState) UpdateFromOutput(data any) {
	presence := extractFieldPresence(data)
	for k, v := range presence {
		s.OutputFields[k] = v
	}

	if s.OutputFields["output"] || s.OutputFields["response"] {
		s.HasOutput = true
	}
}

// AllFields returns all fields (input + output).
func (s *EvalState) AllFields() map[string]bool {
	return mergeFieldPresence(s.InputFields, s.OutputFields)
}

// GetMissingFields returns fields required by target evaluators but not present.
func (s *EvalState) GetMissingFields() []string {
	allFields := s.AllFields()
	missing := make([]string, 0)

	for _, e := range s.TargetEvaluators {
		required := e.GetRequiredFields()
		for _, f := range required {
			if !allFields[f] && !allFields[fieldAlias(f)] {
				// Check if already in missing list
				found := false
				for _, m := range missing {
					if m == f {
						found = true
						break
					}
				}
				if !found {
					missing = append(missing, f)
				}
			}
		}
	}

	return missing
}

// GetCompatibleEvaluators returns evaluators that can work with current fields.
func (s *EvalState) GetCompatibleEvaluators() []EvaluatorType {
	allFields := s.AllFields()
	compatible := make([]EvaluatorType, 0)

	// Check each evaluator
	evaluators := []EvaluatorType{
		EvaluatorFaithfulness,
		EvaluatorAnswerRelevance,
		EvaluatorContextPrecision,
		EvaluatorContextRecall,
		EvaluatorHallucination,
		EvaluatorToxicity,
		EvaluatorCorrectness,
	}

	for _, e := range evaluators {
		required := e.GetRequiredFields()
		hasAll := true
		for _, f := range required {
			if !allFields[f] && !allFields[fieldAlias(f)] {
				hasAll = false
				break
			}
		}
		if hasAll {
			compatible = append(compatible, e)
		}
	}

	return compatible
}

// IsReady checks if the trace is ready for evaluation.
func (s *EvalState) IsReady() bool {
	// Must have output at minimum
	if !s.HasOutput {
		return false
	}

	// If targeting specific evaluators, check requirements
	if len(s.TargetEvaluators) > 0 {
		return len(s.GetMissingFields()) == 0
	}

	// For workflow-based, check workflow requirements
	if s.WorkflowType != "" {
		required := s.WorkflowType.GetRequiredFields()
		allFields := s.AllFields()
		for _, f := range required {
			if !allFields[f] && !allFields[fieldAlias(f)] {
				return false
			}
		}
	}

	return true
}

// BuildMetadata builds evaluation metadata from current state.
func (s *EvalState) BuildMetadata() *EvalMetadataBuilder {
	builder := NewEvalMetadataBuilder().
		WithWorkflow(s.WorkflowType).
		WithGroundTruth(s.HasGroundTruth).
		WithContext(s.HasContext).
		WithOutput(s.HasOutput).
		WithCompatibleEvaluators(s.GetCompatibleEvaluators()...)

	if s.IsReady() {
		builder.MarkReady()
	} else {
		builder.WithMissingFields(s.GetMissingFields()...)
	}

	return builder
}

// fieldAlias returns an alias for a field name (for compatibility).
func fieldAlias(field string) string {
	aliases := map[string]string{
		"query":              "input",
		"input":              "query",
		"output":             "response",
		"response":           "output",
		"context":            "retrieved_contexts",
		"retrieved_contexts": "context",
		"user_input":         "query",
	}
	return aliases[field]
}

// ValidateForEvaluator checks if data has all required fields for an evaluator.
func ValidateForEvaluator(input, output any, evaluator EvaluatorType) error {
	inputPresence := extractFieldPresence(input)
	outputPresence := extractFieldPresence(output)
	allFields := mergeFieldPresence(inputPresence, outputPresence)

	required := evaluator.GetRequiredFields()
	var missing []string

	for _, f := range required {
		if !allFields[f] && !allFields[fieldAlias(f)] {
			missing = append(missing, f)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required fields for %s evaluator: %v", evaluator, missing)
	}

	return nil
}

// ValidateForWorkflow checks if data has all required fields for a workflow.
func ValidateForWorkflow(input, output any, workflow WorkflowType) error {
	inputPresence := extractFieldPresence(input)
	outputPresence := extractFieldPresence(output)
	allFields := mergeFieldPresence(inputPresence, outputPresence)

	required := workflow.GetRequiredFields()
	var missing []string

	for _, f := range required {
		if !allFields[f] && !allFields[fieldAlias(f)] {
			missing = append(missing, f)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required fields for %s workflow: %v", workflow, missing)
	}

	return nil
}

// mergeMetadata merges evaluation metadata into existing metadata.
func mergeMetadata(existing Metadata, evalMeta map[string]any) Metadata {
	if existing == nil {
		existing = make(Metadata)
	}

	for k, v := range evalMeta {
		existing[k] = v
	}

	return existing
}

// mergeTags merges evaluation tags into existing tags (avoiding duplicates).
func mergeTags(existing []string, evalTags []string) []string {
	seen := make(map[string]bool)
	for _, t := range existing {
		seen[t] = true
	}

	result := make([]string, len(existing))
	copy(result, existing)

	for _, t := range evalTags {
		if !seen[t] {
			result = append(result, t)
			seen[t] = true
		}
	}

	return result
}
