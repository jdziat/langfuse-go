package types

// ObservationType represents the type of observation.
type ObservationType string

const (
	ObservationTypeSpan       ObservationType = "SPAN"
	ObservationTypeGeneration ObservationType = "GENERATION"
	ObservationTypeEvent      ObservationType = "EVENT"
)

// String returns the string representation of the observation type.
func (o ObservationType) String() string { return string(o) }

// ObservationLevel represents the severity level of an observation.
type ObservationLevel string

const (
	ObservationLevelDebug   ObservationLevel = "DEBUG"
	ObservationLevelDefault ObservationLevel = "DEFAULT"
	ObservationLevelWarning ObservationLevel = "WARNING"
	ObservationLevelError   ObservationLevel = "ERROR"
)

// String returns the string representation of the observation level.
func (l ObservationLevel) String() string { return string(l) }

// ScoreDataType represents the data type of a score.
type ScoreDataType string

const (
	ScoreDataTypeNumeric     ScoreDataType = "NUMERIC"
	ScoreDataTypeCategorical ScoreDataType = "CATEGORICAL"
	ScoreDataTypeBoolean     ScoreDataType = "BOOLEAN"
)

// String returns the string representation of the score data type.
func (s ScoreDataType) String() string { return string(s) }

// ScoreSource represents the source of a score.
type ScoreSource string

const (
	ScoreSourceAPI        ScoreSource = "API"
	ScoreSourceAnnotation ScoreSource = "ANNOTATION"
	ScoreSourceEval       ScoreSource = "EVAL"
)

// String returns the string representation of the score source.
func (s ScoreSource) String() string { return string(s) }

// PromptType represents the type of a prompt.
type PromptType string

const (
	PromptTypeText PromptType = "text"
	PromptTypeChat PromptType = "chat"
)

// String returns the string representation of the prompt type.
func (p PromptType) String() string { return string(p) }
