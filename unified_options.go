package langfuse

import (
	"time"
)

// ============================================================================
// Unified Observation Options
// ============================================================================
//
// These options work across all observation types (Span, Generation, Event).
// They provide a convenient way to set common properties without needing
// type-specific option functions.
//
// Example:
//
//	span, _ := trace.Span(ctx, "preprocessing",
//	    langfuse.Input(data),
//	    langfuse.Metadata(langfuse.M{"step": 1}),
//	    langfuse.Level(langfuse.ObservationLevelDebug))
//
//	gen, _ := trace.Generation(ctx, "llm-call",
//	    langfuse.Input(prompt),
//	    langfuse.Output(response),
//	    langfuse.Metadata(langfuse.M{"model": "gpt-4"}))

// ObservationOption is an option that can be applied to any observation type.
// It implements SpanOption, GenerationOption, and EventOption.
type ObservationOption interface {
	SpanOption
	GenerationOption
	EventOption
}

// Input sets the input for any observation type.
// This is a unified option that works with Span, Generation, and Event.
//
// Example:
//
//	span, _ := trace.Span(ctx, "process", langfuse.Input(data))
//	gen, _ := trace.Generation(ctx, "llm", langfuse.Input(prompt))
func Input(input any) ObservationOption {
	return &unifiedInputOption{input: input}
}

type unifiedInputOption struct {
	input any
}

func (o *unifiedInputOption) apply(c *spanConfig)        { c.input = o.input }
func (o *unifiedInputOption) apply2(c *generationConfig) { c.input = o.input }
func (o *unifiedInputOption) apply3(c *eventConfig)      { c.input = o.input }

// Ensure unifiedInputOption implements all option interfaces
var _ SpanOption = (*unifiedInputOption)(nil)
var _ GenerationOption = (*unifiedInputOption)(nil)
var _ EventOption = (*unifiedInputOption)(nil)

// Output sets the output for any observation type.
// This is a unified option that works with Span, Generation, and Event.
//
// Example:
//
//	span, _ := trace.Span(ctx, "process", langfuse.Output(result))
//	gen, _ := trace.Generation(ctx, "llm", langfuse.Output(response))
func Output(output any) ObservationOption {
	return &unifiedOutputOption{output: output}
}

type unifiedOutputOption struct {
	output any
}

func (o *unifiedOutputOption) apply(c *spanConfig)        { c.output = o.output }
func (o *unifiedOutputOption) apply2(c *generationConfig) { c.output = o.output }
func (o *unifiedOutputOption) apply3(c *eventConfig)      { c.output = o.output }

var _ SpanOption = (*unifiedOutputOption)(nil)
var _ GenerationOption = (*unifiedOutputOption)(nil)
var _ EventOption = (*unifiedOutputOption)(nil)

// ObsMetadata sets the metadata for any observation type.
// This is a unified option that works with Span, Generation, and Event.
// Named ObsMetadata to avoid conflict with the Metadata type alias.
//
// Example:
//
//	span, _ := trace.Span(ctx, "process",
//	    langfuse.ObsMetadata(langfuse.M{"step": 1}))
func ObsMetadata(metadata Metadata) ObservationOption {
	return &unifiedMetadataOption{metadata: metadata}
}

type unifiedMetadataOption struct {
	metadata Metadata
}

func (o *unifiedMetadataOption) apply(c *spanConfig)        { c.metadata = o.metadata }
func (o *unifiedMetadataOption) apply2(c *generationConfig) { c.metadata = o.metadata }
func (o *unifiedMetadataOption) apply3(c *eventConfig)      { c.metadata = o.metadata }

var _ SpanOption = (*unifiedMetadataOption)(nil)
var _ GenerationOption = (*unifiedMetadataOption)(nil)
var _ EventOption = (*unifiedMetadataOption)(nil)

// ObsLevel sets the observation level for any observation type.
// This is a unified option that works with Span, Generation, and Event.
//
// Example:
//
//	span, _ := trace.Span(ctx, "process",
//	    langfuse.ObsLevel(langfuse.ObservationLevelDebug))
func ObsLevel(level ObservationLevel) ObservationOption {
	return &unifiedLevelOption{level: level}
}

type unifiedLevelOption struct {
	level ObservationLevel
}

func (o *unifiedLevelOption) apply(c *spanConfig) {
	c.level = o.level
	c.hasLevel = true
}
func (o *unifiedLevelOption) apply2(c *generationConfig) {
	c.level = o.level
	c.hasLevel = true
}
func (o *unifiedLevelOption) apply3(c *eventConfig) {
	c.level = o.level
	c.hasLevel = true
}

var _ SpanOption = (*unifiedLevelOption)(nil)
var _ GenerationOption = (*unifiedLevelOption)(nil)
var _ EventOption = (*unifiedLevelOption)(nil)

// StatusMessage sets the status message for any observation type.
// This is a unified option that works with Span, Generation, and Event.
//
// Example:
//
//	span, _ := trace.Span(ctx, "process",
//	    langfuse.StatusMessage("processing complete"))
func StatusMessage(msg string) ObservationOption {
	return &unifiedStatusMessageOption{msg: msg}
}

type unifiedStatusMessageOption struct {
	msg string
}

func (o *unifiedStatusMessageOption) apply(c *spanConfig)        { c.statusMessage = o.msg }
func (o *unifiedStatusMessageOption) apply2(c *generationConfig) { c.statusMessage = o.msg }
func (o *unifiedStatusMessageOption) apply3(c *eventConfig)      { c.statusMessage = o.msg }

var _ SpanOption = (*unifiedStatusMessageOption)(nil)
var _ GenerationOption = (*unifiedStatusMessageOption)(nil)
var _ EventOption = (*unifiedStatusMessageOption)(nil)

// ObsVersion sets the version for any observation type.
// This is a unified option that works with Span, Generation, and Event.
//
// Example:
//
//	span, _ := trace.Span(ctx, "process", langfuse.ObsVersion("1.0.0"))
func ObsVersion(version string) ObservationOption {
	return &unifiedVersionOption{version: version}
}

type unifiedVersionOption struct {
	version string
}

func (o *unifiedVersionOption) apply(c *spanConfig)        { c.version = o.version }
func (o *unifiedVersionOption) apply2(c *generationConfig) { c.version = o.version }
func (o *unifiedVersionOption) apply3(c *eventConfig)      { c.version = o.version }

var _ SpanOption = (*unifiedVersionOption)(nil)
var _ GenerationOption = (*unifiedVersionOption)(nil)
var _ EventOption = (*unifiedVersionOption)(nil)

// ObsEnvironment sets the environment for any observation type.
// This is a unified option that works with Span, Generation, and Event.
//
// Example:
//
//	span, _ := trace.Span(ctx, "process", langfuse.ObsEnvironment("production"))
func ObsEnvironment(env string) ObservationOption {
	return &unifiedEnvironmentOption{env: env}
}

type unifiedEnvironmentOption struct {
	env string
}

func (o *unifiedEnvironmentOption) apply(c *spanConfig)        { c.environment = o.env }
func (o *unifiedEnvironmentOption) apply2(c *generationConfig) { c.environment = o.env }
func (o *unifiedEnvironmentOption) apply3(c *eventConfig)      { c.environment = o.env }

var _ SpanOption = (*unifiedEnvironmentOption)(nil)
var _ GenerationOption = (*unifiedEnvironmentOption)(nil)
var _ EventOption = (*unifiedEnvironmentOption)(nil)

// ObsStartTime sets the start time for any observation type.
// This is a unified option that works with Span, Generation, and Event.
//
// Example:
//
//	startedAt := time.Now()
//	// ... do work ...
//	span, _ := trace.Span(ctx, "process", langfuse.ObsStartTime(startedAt))
func ObsStartTime(t time.Time) ObservationOption {
	return &unifiedStartTimeOption{t: t}
}

type unifiedStartTimeOption struct {
	t time.Time
}

func (o *unifiedStartTimeOption) apply(c *spanConfig) {
	c.startTime = o.t
	c.hasStartTime = true
}
func (o *unifiedStartTimeOption) apply2(c *generationConfig) {
	c.startTime = o.t
	c.hasStartTime = true
}
func (o *unifiedStartTimeOption) apply3(c *eventConfig) {
	c.startTime = o.t
	c.hasStartTime = true
}

var _ SpanOption = (*unifiedStartTimeOption)(nil)
var _ GenerationOption = (*unifiedStartTimeOption)(nil)
var _ EventOption = (*unifiedStartTimeOption)(nil)
