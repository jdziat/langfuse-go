package evaluation

import (
	"encoding/json"
	"reflect"
	"strings"
)

// Input represents a structured input that can be flattened for evaluation.
type Input interface {
	// EvalFields returns the evaluation-relevant fields as a flat map.
	EvalFields() map[string]any
}

// Output represents a structured output that can be flattened for evaluation.
type Output interface {
	// EvalFields returns the evaluation-relevant fields as a flat map.
	EvalFields() map[string]any
}

// InputFlattener flattens structured inputs for evaluation.
type InputFlattener struct {
	mode          Mode
	fieldMappings map[string]string // source field -> target field
}

// NewInputFlattener creates a new input flattener for the given mode.
func NewInputFlattener(mode Mode) *InputFlattener {
	f := &InputFlattener{
		mode:          mode,
		fieldMappings: make(map[string]string),
	}

	// Set up field mappings based on mode
	switch mode {
	case ModeRAGAS:
		// RAGAS uses specific field names
		f.fieldMappings["query"] = "user_input"
		f.fieldMappings["context"] = "retrieved_contexts"
		f.fieldMappings["output"] = "response"
		f.fieldMappings["answer"] = "response"
	case ModeLangfuse:
		// Langfuse uses standard names
		f.fieldMappings["query"] = "input"
		f.fieldMappings["question"] = "input"
	}

	return f
}

// Flatten flattens the input data for evaluation.
func (f *InputFlattener) Flatten(data any) map[string]any {
	if data == nil {
		return nil
	}

	// Check for Input implementation
	if ei, ok := data.(Input); ok {
		result := ei.EvalFields()
		return f.applyMappings(result)
	}

	// Handle map[string]any directly
	if m, ok := data.(map[string]any); ok {
		return f.applyMappings(f.flattenMap(m, ""))
	}

	// Handle structs via reflection
	return f.applyMappings(f.flattenStruct(data, ""))
}

// flattenStruct flattens a struct to a map using json tags.
func (f *InputFlattener) flattenStruct(data any, prefix string) map[string]any {
	result := make(map[string]any)

	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return result
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		// Not a struct, return as-is
		if prefix != "" {
			result[prefix] = data
		}
		return result
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get field name from json tag or use field name
		name := f.getFieldName(field)
		if name == "" || name == "-" {
			continue
		}

		// Build full key with prefix
		key := name
		if prefix != "" {
			key = prefix + "_" + name
		}

		// Skip zero values for omitempty fields
		if f.shouldOmit(field, fieldValue) {
			continue
		}

		// Handle the field value
		value := fieldValue.Interface()

		// Check if nested struct should be flattened
		if f.shouldFlatten(fieldValue) {
			nested := f.flattenStruct(value, key)
			for k, v := range nested {
				result[k] = v
			}
		} else {
			result[key] = value
		}
	}

	return result
}

// flattenMap flattens a map recursively.
func (f *InputFlattener) flattenMap(data map[string]any, prefix string) map[string]any {
	result := make(map[string]any)

	for k, v := range data {
		key := k
		if prefix != "" {
			key = prefix + "_" + k
		}

		// Check if value is a nested map that should be flattened
		if nested, ok := v.(map[string]any); ok {
			for nk, nv := range f.flattenMap(nested, key) {
				result[nk] = nv
			}
		} else {
			result[key] = v
		}
	}

	return result
}

// getFieldName extracts the field name from json tag or struct field name.
func (f *InputFlattener) getFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return strings.ToLower(field.Name)
	}

	// Handle json tag options
	parts := strings.Split(tag, ",")
	name := parts[0]
	if name == "-" {
		return ""
	}
	if name == "" {
		return strings.ToLower(field.Name)
	}
	return name
}

// shouldOmit checks if a field should be omitted (omitempty + zero value).
func (f *InputFlattener) shouldOmit(field reflect.StructField, value reflect.Value) bool {
	tag := field.Tag.Get("json")
	if !strings.Contains(tag, "omitempty") {
		return false
	}
	return value.IsZero()
}

// shouldFlatten determines if a value should be recursively flattened.
func (f *InputFlattener) shouldFlatten(value reflect.Value) bool {
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return false
		}
		value = value.Elem()
	}

	// Flatten structs but not slices, maps, or primitives
	return value.Kind() == reflect.Struct
}

// applyMappings applies field name mappings based on evaluation mode.
func (f *InputFlattener) applyMappings(data map[string]any) map[string]any {
	if len(f.fieldMappings) == 0 {
		return data
	}

	result := make(map[string]any, len(data))
	for k, v := range data {
		if mapped, ok := f.fieldMappings[k]; ok {
			result[mapped] = v
		} else {
			result[k] = v
		}
	}
	return result
}

// FlattenedInput wraps flattened input data with metadata.
type FlattenedInput struct {
	// Fields contains the flattened input fields.
	Fields map[string]any `json:"-"`

	// Original preserves the original structured input.
	Original any `json:"_original,omitempty"`

	// EvalType indicates the evaluation type this was prepared for.
	EvalType WorkflowType `json:"_eval_type,omitempty"`
}

// MarshalJSON implements json.Marshaler to inline Fields.
func (f FlattenedInput) MarshalJSON() ([]byte, error) {
	// Create a copy of fields and add metadata
	result := make(map[string]any, len(f.Fields)+2)
	for k, v := range f.Fields {
		result[k] = v
	}
	if f.Original != nil {
		result["_original"] = f.Original
	}
	if f.EvalType != "" {
		result["_eval_type"] = f.EvalType
	}
	return json.Marshal(result)
}

// FlattenedOutput wraps flattened output data with metadata.
type FlattenedOutput struct {
	// Fields contains the flattened output fields.
	Fields map[string]any `json:"-"`

	// Original preserves the original structured output.
	Original any `json:"_original,omitempty"`
}

// MarshalJSON implements json.Marshaler to inline Fields.
func (f FlattenedOutput) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.Fields)
}

// StandardInput provides a standard evaluation input structure.
type StandardInput struct {
	// Query is the user's question or input.
	Query string `json:"query,omitempty"`

	// Context contains retrieved documents or context chunks.
	Context []string `json:"context,omitempty"`

	// GroundTruth is the expected correct answer for evaluation.
	GroundTruth string `json:"ground_truth,omitempty"`

	// Messages contains chat messages (for chat workflows).
	Messages []map[string]string `json:"messages,omitempty"`

	// SystemPrompt is the system prompt used.
	SystemPrompt string `json:"system_prompt,omitempty"`

	// Additional allows arbitrary additional fields.
	Additional map[string]any `json:"additional,omitempty"`
}

// EvalFields implements Input.
func (s *StandardInput) EvalFields() map[string]any {
	result := make(map[string]any)

	if s.Query != "" {
		result["query"] = s.Query
		result["input"] = s.Query // alias for compatibility
	}
	if len(s.Context) > 0 {
		result["context"] = s.Context
	}
	if s.GroundTruth != "" {
		result["ground_truth"] = s.GroundTruth
	}
	if len(s.Messages) > 0 {
		result["messages"] = s.Messages
	}
	if s.SystemPrompt != "" {
		result["system_prompt"] = s.SystemPrompt
	}
	for k, v := range s.Additional {
		result[k] = v
	}

	return result
}

// StandardOutput provides a standard evaluation output structure.
type StandardOutput struct {
	// Output is the generated response/answer.
	Output string `json:"output"`

	// Citations lists source documents used.
	Citations []string `json:"citations,omitempty"`

	// Confidence is the model's confidence score.
	Confidence float64 `json:"confidence,omitempty"`

	// Reasoning provides explanation for the output.
	Reasoning string `json:"reasoning,omitempty"`

	// ToolCalls lists any tool calls made.
	ToolCalls []map[string]any `json:"tool_calls,omitempty"`

	// Additional allows arbitrary additional fields.
	Additional map[string]any `json:"additional,omitempty"`
}

// EvalFields implements Output.
func (s *StandardOutput) EvalFields() map[string]any {
	result := make(map[string]any)

	result["output"] = s.Output
	if len(s.Citations) > 0 {
		result["citations"] = s.Citations
	}
	if s.Confidence > 0 {
		result["confidence"] = s.Confidence
	}
	if s.Reasoning != "" {
		result["reasoning"] = s.Reasoning
	}
	if len(s.ToolCalls) > 0 {
		result["tool_calls"] = s.ToolCalls
	}
	for k, v := range s.Additional {
		result[k] = v
	}

	return result
}

// PrepareInput prepares input data for evaluation based on config.
func PrepareInput(data any, config *Config) any {
	if config == nil || config.Mode == ModeOff {
		return data
	}

	if !config.FlattenInput {
		return data
	}

	flattener := NewInputFlattener(config.Mode)
	fields := flattener.Flatten(data)

	return FlattenedInput{
		Fields:   fields,
		Original: data,
	}
}

// PrepareOutput prepares output data for evaluation based on config.
func PrepareOutput(data any, config *Config) any {
	if config == nil || config.Mode == ModeOff {
		return data
	}

	if !config.FlattenOutput {
		return data
	}

	// Handle string output specially - wrap in standard structure
	if s, ok := data.(string); ok {
		return FlattenedOutput{
			Fields: map[string]any{"output": s},
		}
	}

	flattener := NewInputFlattener(config.Mode)
	fields := flattener.Flatten(data)

	return FlattenedOutput{
		Fields:   fields,
		Original: data,
	}
}
