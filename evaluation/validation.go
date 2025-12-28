package evaluation

import (
	"fmt"
	"reflect"
	"strings"
)

// EvaluatorRequirements defines what fields an evaluator expects.
type EvaluatorRequirements struct {
	// Name is the evaluator name for error messages
	Name string

	// RequiredFields are fields that must be present
	RequiredFields []string

	// OptionalFields are fields that can enhance evaluation
	OptionalFields []string

	// Description explains what this evaluator does
	Description string
}

var (
	// RAGEvaluator defines requirements for RAG evaluations.
	// RAG evaluators assess retrieval quality and answer accuracy.
	RAGEvaluator = EvaluatorRequirements{
		Name:           "RAG",
		RequiredFields: []string{"query", "context", "output"},
		OptionalFields: []string{"ground_truth", "citations", "source_chunks"},
		Description:    "Evaluates retrieval-augmented generation quality",
	}

	// ContextRelevanceEvaluator checks if retrieved context is relevant.
	ContextRelevanceEvaluator = EvaluatorRequirements{
		Name:           "Context Relevance",
		RequiredFields: []string{"query", "context"},
		OptionalFields: []string{"output"},
		Description:    "Evaluates if retrieved context is relevant to the query",
	}

	// ContextCorrectnessEvaluator checks factual alignment with ground truth.
	ContextCorrectnessEvaluator = EvaluatorRequirements{
		Name:           "Context Correctness",
		RequiredFields: []string{"query", "context", "ground_truth"},
		OptionalFields: []string{"output"},
		Description:    "Evaluates factual alignment between context and ground truth",
	}

	// QAEvaluator defines requirements for Q&A evaluations.
	QAEvaluator = EvaluatorRequirements{
		Name:           "Q&A",
		RequiredFields: []string{"query", "output"},
		OptionalFields: []string{"ground_truth", "confidence"},
		Description:    "Evaluates question-answering accuracy",
	}

	// SummarizationEvaluator defines requirements for summarization evaluations.
	SummarizationEvaluator = EvaluatorRequirements{
		Name:           "Summarization",
		RequiredFields: []string{"input", "output"},
		OptionalFields: []string{"ground_truth", "compression_ratio"},
		Description:    "Evaluates summary quality and completeness",
	}

	// ClassificationEvaluator defines requirements for classification evaluations.
	ClassificationEvaluator = EvaluatorRequirements{
		Name:           "Classification",
		RequiredFields: []string{"input", "output"},
		OptionalFields: []string{"ground_truth", "confidence", "scores"},
		Description:    "Evaluates classification accuracy",
	}

	// ToxicityEvaluator defines requirements for toxicity detection.
	ToxicityEvaluator = EvaluatorRequirements{
		Name:           "Toxicity",
		RequiredFields: []string{"output"},
		OptionalFields: []string{"input", "toxicity_score", "categories"},
		Description:    "Evaluates content for toxic language",
	}

	// HallucinationEvaluator defines requirements for hallucination detection.
	HallucinationEvaluator = EvaluatorRequirements{
		Name:           "Hallucination",
		RequiredFields: []string{"query", "context", "output"},
		OptionalFields: []string{"hallucination_score"},
		Description:    "Detects hallucinations in generated content",
	}

	// FaithfulnessEvaluator checks if output is faithful to context.
	FaithfulnessEvaluator = EvaluatorRequirements{
		Name:           "Faithfulness",
		RequiredFields: []string{"context", "output"},
		OptionalFields: []string{"query"},
		Description:    "Evaluates if the output is faithful to the provided context",
	}

	// AnswerRelevanceEvaluator checks if answer is relevant to query.
	AnswerRelevanceEvaluator = EvaluatorRequirements{
		Name:           "Answer Relevance",
		RequiredFields: []string{"query", "output"},
		OptionalFields: []string{"context"},
		Description:    "Evaluates if the answer is relevant to the query",
	}
)

// ValidateFor checks if input and output structures match evaluator requirements.
//
// Example:
//
//	input := &evaluation.RAGInput{Query: "test", Context: []string{"ctx"}}
//	output := &evaluation.RAGOutput{Output: "answer"}
//	err := evaluation.ValidateFor(input, output, evaluation.RAGEvaluator)
func ValidateFor(input, output any, reqs EvaluatorRequirements) error {
	inputFields := extractFields(input)
	outputFields := extractFields(output)
	allFields := mergeFields(inputFields, outputFields)

	var missing []string
	for _, required := range reqs.RequiredFields {
		if !containsField(allFields, required) {
			missing = append(missing, required)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf(
			"trace missing required fields for %s evaluator: %v (found: %v)",
			reqs.Name, missing, allFields,
		)
	}

	return nil
}

// ValidateInput validates that an input structure has the required fields.
func ValidateInput(input any, reqs EvaluatorRequirements) error {
	fields := extractFields(input)

	// Check which required fields should be in input (not output-specific)
	inputRequiredFields := filterInputFields(reqs.RequiredFields)

	var missing []string
	for _, required := range inputRequiredFields {
		if !containsField(fields, required) {
			missing = append(missing, required)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf(
			"input missing required fields for %s evaluator: %v (found: %v)",
			reqs.Name, missing, fields,
		)
	}

	return nil
}

// ValidateOutput validates that an output structure has the required fields.
func ValidateOutput(output any, reqs EvaluatorRequirements) error {
	fields := extractFields(output)

	// Check if output field is required and present
	if containsField(reqs.RequiredFields, "output") && !containsField(fields, "output") {
		return fmt.Errorf(
			"output missing 'output' field required for %s evaluator (found: %v)",
			reqs.Name, fields,
		)
	}

	return nil
}

// extractFields extracts field names from a struct or map.
func extractFields(data any) []string {
	if data == nil {
		return nil
	}

	fields := []string{}

	// Handle map[string]any
	if m, ok := data.(map[string]any); ok {
		for k, v := range m {
			// Only include non-nil, non-zero values
			if !isZeroValue(v) {
				fields = append(fields, k)
			}
		}
		return fields
	}

	// Handle structs using reflection
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return fields
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fields
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Get JSON tag name
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}

		// Extract field name before comma
		name := extractJSONName(tag)
		if name == "" {
			continue
		}

		// Skip fields that have zero values (validation cares about actual values)
		fieldValue := v.Field(i)
		if isZeroReflectValue(fieldValue) {
			continue
		}

		fields = append(fields, name)
	}

	return fields
}

// extractJSONName extracts the field name from a JSON tag.
func extractJSONName(tag string) string {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx]
	}
	return tag
}

// isZeroValue checks if a value is zero/nil.
func isZeroValue(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	return isZeroReflectValue(rv)
}

// isZeroReflectValue checks if a reflect.Value is zero.
func isZeroReflectValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Invalid:
		return true
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return v.IsNil()
	default:
		return v.IsZero()
	}
}

// containsField checks if a slice contains a string.
func containsField(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// mergeFields merges two field slices, removing duplicates.
func mergeFields(a, b []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, s := range a {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	return result
}

// filterInputFields returns fields that are typically in input (not output-specific).
func filterInputFields(fields []string) []string {
	outputFields := map[string]bool{
		"output":              true,
		"citations":           true,
		"source_chunks":       true,
		"confidence":          true,
		"reasoning":           true,
		"compression_ratio":   true,
		"scores":              true,
		"toxicity_score":      true,
		"hallucination_score": true,
	}

	var inputFields []string
	for _, f := range fields {
		if !outputFields[f] {
			inputFields = append(inputFields, f)
		}
	}
	return inputFields
}

// ValidationResult contains detailed validation results.
type ValidationResult struct {
	Valid         bool
	MissingFields []string
	PresentFields []string
	Warnings      []string
	EvaluatorName string
}

// ValidateDetailed performs detailed validation and returns a result struct.
func ValidateDetailed(input, output any, reqs EvaluatorRequirements) *ValidationResult {
	result := &ValidationResult{
		Valid:         true,
		EvaluatorName: reqs.Name,
	}

	inputFields := extractFields(input)
	outputFields := extractFields(output)
	allFields := mergeFields(inputFields, outputFields)
	result.PresentFields = allFields

	// Check required fields
	for _, required := range reqs.RequiredFields {
		if !containsField(allFields, required) {
			result.Valid = false
			result.MissingFields = append(result.MissingFields, required)
		}
	}

	// Check optional fields for warnings
	for _, optional := range reqs.OptionalFields {
		if !containsField(allFields, optional) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("optional field '%s' not provided - evaluation may be less accurate", optional))
		}
	}

	return result
}

// Error returns an error if validation failed, nil otherwise.
func (r *ValidationResult) Error() error {
	if r.Valid {
		return nil
	}
	return fmt.Errorf(
		"validation failed for %s evaluator: missing required fields %v",
		r.EvaluatorName, r.MissingFields,
	)
}
