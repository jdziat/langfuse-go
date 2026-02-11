package evaluation

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// State tracks the current evaluation state for a trace.
type State struct {
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

// NewState creates a new evaluation state tracker.
func NewState() *State {
	return &State{
		InputFields:  make(map[string]bool),
		OutputFields: make(map[string]bool),
	}
}

// UpdateFromInput updates state from input data.
func (s *State) UpdateFromInput(data any) {
	presence := ExtractFieldPresence(data)
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
func (s *State) UpdateFromOutput(data any) {
	presence := ExtractFieldPresence(data)
	for k, v := range presence {
		s.OutputFields[k] = v
	}

	if s.OutputFields["output"] || s.OutputFields["response"] {
		s.HasOutput = true
	}
}

// AllFields returns all fields (input + output).
func (s *State) AllFields() map[string]bool {
	return MergeFieldPresence(s.InputFields, s.OutputFields)
}

// GetMissingFields returns fields required by target evaluators but not present.
func (s *State) GetMissingFields() []string {
	allFields := s.AllFields()
	missing := make([]string, 0)

	for _, e := range s.TargetEvaluators {
		required := e.GetRequiredFields()
		for _, f := range required {
			if !allFields[f] && !allFields[FieldAlias(f)] {
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
func (s *State) GetCompatibleEvaluators() []EvaluatorType {
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
			if !allFields[f] && !allFields[FieldAlias(f)] {
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
func (s *State) IsReady() bool {
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
			if !allFields[f] && !allFields[FieldAlias(f)] {
				return false
			}
		}
	}

	return true
}

// BuildMetadata builds evaluation metadata from current state.
func (s *State) BuildMetadata() *MetadataBuilder {
	builder := NewMetadataBuilder().
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

// FieldAlias returns an alias for a field name (for compatibility).
func FieldAlias(field string) string {
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

// ExtractFieldPresence checks which evaluation fields are present in data.
func ExtractFieldPresence(data any) map[string]bool {
	presence := make(map[string]bool)

	if data == nil {
		return presence
	}

	// Handle Input/Output interfaces
	if ei, ok := data.(Input); ok {
		for k := range ei.EvalFields() {
			presence[k] = true
		}
		return presence
	}
	if eo, ok := data.(Output); ok {
		for k := range eo.EvalFields() {
			presence[k] = true
		}
		return presence
	}

	// Handle map
	if m, ok := data.(map[string]any); ok {
		for k := range m {
			presence[k] = true
		}
		return presence
	}

	// Handle simple string (treated as output)
	if s, ok := data.(string); ok && s != "" {
		presence["output"] = true
		presence["response"] = true
		return presence
	}

	// Handle FlattenedInput/FlattenedOutput
	if fi, ok := data.(FlattenedInput); ok {
		for k := range fi.Fields {
			presence[k] = true
		}
		return presence
	}
	if fo, ok := data.(FlattenedOutput); ok {
		for k := range fo.Fields {
			presence[k] = true
		}
		return presence
	}

	// Handle structs via reflection
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return presence
		}
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}

			name := field.Tag.Get("json")
			if name == "" || name == "-" {
				name = strings.ToLower(field.Name)
			} else {
				name = strings.Split(name, ",")[0]
			}

			fieldValue := v.Field(i)
			if !fieldValue.IsZero() {
				presence[name] = true
			}
		}
	}

	return presence
}

// MergeFieldPresence merges two presence maps.
func MergeFieldPresence(a, b map[string]bool) map[string]bool {
	result := make(map[string]bool, len(a)+len(b))
	for k, v := range a {
		result[k] = v
	}
	for k, v := range b {
		result[k] = v
	}
	return result
}

// MetadataBuilder helps build evaluation metadata.
type MetadataBuilder struct {
	metadata Metadata
}

// NewMetadataBuilder creates a new evaluation metadata builder.
func NewMetadataBuilder() *MetadataBuilder {
	return &MetadataBuilder{
		metadata: Metadata{
			Version: MetadataVersion,
		},
	}
}

// WithWorkflow sets the workflow type.
func (b *MetadataBuilder) WithWorkflow(w WorkflowType) *MetadataBuilder {
	b.metadata.WorkflowType = w
	b.metadata.CompatibleEvaluators = w.GetCompatibleEvaluators()
	return b
}

// WithGroundTruth marks that ground truth is present.
func (b *MetadataBuilder) WithGroundTruth(hasGroundTruth bool) *MetadataBuilder {
	b.metadata.HasGroundTruth = hasGroundTruth
	return b
}

// WithContext marks that context is present.
func (b *MetadataBuilder) WithContext(hasContext bool) *MetadataBuilder {
	b.metadata.HasContext = hasContext
	return b
}

// WithOutput marks that output is present.
func (b *MetadataBuilder) WithOutput(hasOutput bool) *MetadataBuilder {
	b.metadata.HasOutput = hasOutput
	return b
}

// WithCompatibleEvaluators sets the compatible evaluators.
func (b *MetadataBuilder) WithCompatibleEvaluators(evaluators ...EvaluatorType) *MetadataBuilder {
	b.metadata.CompatibleEvaluators = evaluators
	return b
}

// WithMissingFields sets the missing fields list.
func (b *MetadataBuilder) WithMissingFields(fields ...string) *MetadataBuilder {
	b.metadata.MissingFields = fields
	return b
}

// MarkReady marks the trace as evaluation-ready.
func (b *MetadataBuilder) MarkReady() *MetadataBuilder {
	b.metadata.Ready = true
	now := time.Now()
	b.metadata.ReadyAt = &now
	b.metadata.MissingFields = nil
	return b
}

// Build returns the built metadata.
func (b *MetadataBuilder) Build() Metadata {
	return b.metadata
}

// BuildAsMap returns the metadata as a map suitable for trace metadata.
func (b *MetadataBuilder) BuildAsMap() map[string]any {
	return map[string]any{
		MetadataKey: b.metadata,
	}
}

// GenerateEvalTags generates evaluation tags based on the current state.
func (b *MetadataBuilder) GenerateEvalTags() []string {
	tags := make([]string, 0, 5)

	// Ready status
	if b.metadata.Ready {
		tags = append(tags, TagReady)
	} else {
		tags = append(tags, TagNotReady)
	}

	// Workflow type
	if b.metadata.WorkflowType != "" {
		tags = append(tags, TagForWorkflow(b.metadata.WorkflowType))
	}

	// Ground truth
	if b.metadata.HasGroundTruth {
		tags = append(tags, TagGroundTruth)
	}

	// Compatible evaluators (limit to avoid too many tags)
	for i, e := range b.metadata.CompatibleEvaluators {
		if i >= 3 { // Max 3 evaluator tags
			break
		}
		tags = append(tags, TagForEvaluator(e))
	}

	return tags
}

// ValidateForEvaluator checks if data has all required fields for an evaluator.
func ValidateForEvaluator(input, output any, evaluator EvaluatorType) error {
	inputPresence := ExtractFieldPresence(input)
	outputPresence := ExtractFieldPresence(output)
	allFields := MergeFieldPresence(inputPresence, outputPresence)

	required := evaluator.GetRequiredFields()
	var missing []string

	for _, f := range required {
		if !allFields[f] && !allFields[FieldAlias(f)] {
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
	inputPresence := ExtractFieldPresence(input)
	outputPresence := ExtractFieldPresence(output)
	allFields := MergeFieldPresence(inputPresence, outputPresence)

	required := workflow.GetRequiredFields()
	var missing []string

	for _, f := range required {
		if !allFields[f] && !allFields[FieldAlias(f)] {
			missing = append(missing, f)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required fields for %s workflow: %v", workflow, missing)
	}

	return nil
}
