# OpenTelemetry Integration Proposal (Zero External Dependencies)

**Status:** Draft
**Author:** Engineering Team
**Date:** 2025-12-26

## Executive Summary

This proposal outlines an approach for adding OpenTelemetry (OTel) support to the Langfuse Go SDK without introducing external dependencies. The integration enables bidirectional interoperability: exporting Langfuse traces to OTel collectors and importing OTel spans into Langfuse.

## Goals

1. **Zero external dependencies** - Maintain the SDK's current dependency-free design
2. **OTLP/HTTP export** - Send Langfuse traces to OTel collectors via OTLP/HTTP (JSON)
3. **OTel span import** - Accept OTel span data and convert to Langfuse observations
4. **W3C Trace Context** - Propagate trace context across service boundaries
5. **Semantic conventions** - Support LLM-specific semantic conventions (GenAI)

## Non-Goals

- gRPC transport (requires external deps)
- Full OTel SDK compatibility (we are not replacing the OTel SDK)
- Automatic instrumentation
- OTLP/Proto binary encoding (requires protobuf)

---

## 1. Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Langfuse Go SDK                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────────┐  │
│  │   Langfuse   │    │    OTel      │    │   W3C Trace Context  │  │
│  │   Native     │◄──►│   Bridge     │◄──►│   Propagator         │  │
│  │   Types      │    │              │    │                      │  │
│  └──────────────┘    └──────────────┘    └──────────────────────┘  │
│         │                   │                       │               │
│         ▼                   ▼                       ▼               │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────────┐  │
│  │  Langfuse    │    │    OTLP      │    │   HTTP Header        │  │
│  │  Ingestion   │    │   Exporter   │    │   Injection/Extract  │  │
│  │  Endpoint    │    │   (HTTP/JSON)│    │                      │  │
│  └──────────────┘    └──────────────┘    └──────────────────────┘  │
│         │                   │                                       │
└─────────│───────────────────│───────────────────────────────────────┘
          ▼                   ▼
   ┌────────────┐      ┌────────────────┐
   │  Langfuse  │      │ OTel Collector │
   │  Cloud     │      │ (Jaeger, etc.) │
   └────────────┘      └────────────────┘
```

---

## 2. Core Components

### 2.1 OTLP Data Structures (Native Go)

Implement OTLP JSON structures as pure Go types. OTLP/HTTP with JSON encoding requires no protobuf dependency.

```go
// otel/types.go

package otel

import "time"

// Resource represents the entity producing telemetry
type Resource struct {
    Attributes []KeyValue `json:"attributes,omitempty"`
}

// KeyValue represents an attribute key-value pair
type KeyValue struct {
    Key   string     `json:"key"`
    Value AnyValue   `json:"value"`
}

// AnyValue represents a typed attribute value
type AnyValue struct {
    StringValue *string      `json:"stringValue,omitempty"`
    BoolValue   *bool        `json:"boolValue,omitempty"`
    IntValue    *int64       `json:"intValue,omitempty"`
    DoubleValue *float64     `json:"doubleValue,omitempty"`
    ArrayValue  *ArrayValue  `json:"arrayValue,omitempty"`
    KvlistValue *KvlistValue `json:"kvlistValue,omitempty"`
}

// ArrayValue for array attributes
type ArrayValue struct {
    Values []AnyValue `json:"values"`
}

// KvlistValue for nested key-value maps
type KvlistValue struct {
    Values []KeyValue `json:"values"`
}

// InstrumentationScope identifies the instrumentation library
type InstrumentationScope struct {
    Name    string `json:"name"`
    Version string `json:"version,omitempty"`
}

// Span represents a single OTel span
type Span struct {
    TraceID                [16]byte   `json:"-"` // Serialized as hex
    SpanID                 [8]byte    `json:"-"` // Serialized as hex
    TraceState             string     `json:"traceState,omitempty"`
    ParentSpanID           [8]byte    `json:"-"` // Serialized as hex
    Name                   string     `json:"name"`
    Kind                   SpanKind   `json:"kind"`
    StartTimeUnixNano      uint64     `json:"startTimeUnixNano"`
    EndTimeUnixNano        uint64     `json:"endTimeUnixNano"`
    Attributes             []KeyValue `json:"attributes,omitempty"`
    Events                 []Event    `json:"events,omitempty"`
    Links                  []Link     `json:"links,omitempty"`
    Status                 Status     `json:"status,omitempty"`
    DroppedAttributesCount uint32     `json:"droppedAttributesCount,omitempty"`
    DroppedEventsCount     uint32     `json:"droppedEventsCount,omitempty"`
    DroppedLinksCount      uint32     `json:"droppedLinksCount,omitempty"`
}

// SpanKind defines the relationship of a span
type SpanKind int32

const (
    SpanKindUnspecified SpanKind = 0
    SpanKindInternal    SpanKind = 1
    SpanKindServer      SpanKind = 2
    SpanKindClient      SpanKind = 3
    SpanKindProducer    SpanKind = 4
    SpanKindConsumer    SpanKind = 5
)

// Event represents a span event
type Event struct {
    TimeUnixNano           uint64     `json:"timeUnixNano"`
    Name                   string     `json:"name"`
    Attributes             []KeyValue `json:"attributes,omitempty"`
    DroppedAttributesCount uint32     `json:"droppedAttributesCount,omitempty"`
}

// Link represents a span link
type Link struct {
    TraceID                [16]byte   `json:"-"`
    SpanID                 [8]byte    `json:"-"`
    TraceState             string     `json:"traceState,omitempty"`
    Attributes             []KeyValue `json:"attributes,omitempty"`
    DroppedAttributesCount uint32     `json:"droppedAttributesCount,omitempty"`
}

// Status represents span status
type Status struct {
    Message string     `json:"message,omitempty"`
    Code    StatusCode `json:"code"`
}

type StatusCode int32

const (
    StatusCodeUnset StatusCode = 0
    StatusCodeOK    StatusCode = 1
    StatusCodeError StatusCode = 2
)

// ExportTraceServiceRequest is the OTLP export request
type ExportTraceServiceRequest struct {
    ResourceSpans []ResourceSpans `json:"resourceSpans"`
}

type ResourceSpans struct {
    Resource   Resource     `json:"resource,omitempty"`
    ScopeSpans []ScopeSpans `json:"scopeSpans"`
}

type ScopeSpans struct {
    Scope InstrumentationScope `json:"scope,omitempty"`
    Spans []Span               `json:"spans"`
}
```

### 2.2 W3C Trace Context Propagator

Implement W3C Trace Context parsing and generation for distributed tracing.

```go
// otel/propagation.go

package otel

import (
    "encoding/hex"
    "fmt"
    "net/http"
    "regexp"
    "strings"
)

const (
    TraceparentHeader = "traceparent"
    TracestateHeader  = "tracestate"
)

// TraceContext holds W3C trace context data
type TraceContext struct {
    TraceID    [16]byte
    SpanID     [8]byte
    TraceFlags byte
    TraceState string
}

// traceparent format: version-traceid-spanid-traceflags
// Example: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
var traceparentRegex = regexp.MustCompile(
    `^([0-9a-f]{2})-([0-9a-f]{32})-([0-9a-f]{16})-([0-9a-f]{2})$`,
)

// ParseTraceparent parses a W3C traceparent header
func ParseTraceparent(value string) (*TraceContext, error) {
    value = strings.TrimSpace(strings.ToLower(value))
    matches := traceparentRegex.FindStringSubmatch(value)
    if matches == nil {
        return nil, fmt.Errorf("invalid traceparent format: %s", value)
    }

    version := matches[1]
    if version == "ff" {
        return nil, fmt.Errorf("invalid version: %s", version)
    }

    tc := &TraceContext{}

    traceIDBytes, _ := hex.DecodeString(matches[2])
    copy(tc.TraceID[:], traceIDBytes)

    spanIDBytes, _ := hex.DecodeString(matches[3])
    copy(tc.SpanID[:], spanIDBytes)

    flagBytes, _ := hex.DecodeString(matches[4])
    tc.TraceFlags = flagBytes[0]

    return tc, nil
}

// String returns the traceparent header value
func (tc *TraceContext) String() string {
    return fmt.Sprintf("00-%s-%s-%02x",
        hex.EncodeToString(tc.TraceID[:]),
        hex.EncodeToString(tc.SpanID[:]),
        tc.TraceFlags,
    )
}

// IsSampled returns true if the trace is sampled
func (tc *TraceContext) IsSampled() bool {
    return tc.TraceFlags&0x01 == 0x01
}

// Propagator handles trace context injection and extraction
type Propagator struct{}

// Extract extracts trace context from HTTP headers
func (p *Propagator) Extract(headers http.Header) (*TraceContext, error) {
    traceparent := headers.Get(TraceparentHeader)
    if traceparent == "" {
        return nil, nil
    }

    tc, err := ParseTraceparent(traceparent)
    if err != nil {
        return nil, err
    }

    tc.TraceState = headers.Get(TracestateHeader)
    return tc, nil
}

// Inject injects trace context into HTTP headers
func (p *Propagator) Inject(tc *TraceContext, headers http.Header) {
    if tc == nil {
        return
    }
    headers.Set(TraceparentHeader, tc.String())
    if tc.TraceState != "" {
        headers.Set(TracestateHeader, tc.TraceState)
    }
}
```

### 2.3 Type Bridge (Langfuse ↔ OTel)

Bidirectional conversion between Langfuse and OTel types.

```go
// otel/bridge.go

package otel

import (
    "crypto/rand"
    "encoding/hex"
    "time"
)

// LangfuseObservation represents a Langfuse observation for conversion
type LangfuseObservation struct {
    ID                  string
    TraceID             string
    ParentObservationID string
    Type                string // "SPAN", "GENERATION", "EVENT"
    Name                string
    StartTime           time.Time
    EndTime             *time.Time
    Input               interface{}
    Output              interface{}
    Metadata            map[string]interface{}
    Level               string
    StatusMessage       string
    Model               string // for GENERATION
    ModelParameters     map[string]interface{}
    Usage               *LangfuseUsage
}

type LangfuseUsage struct {
    Input  int64
    Output int64
    Total  int64
}

// ToOTelSpan converts a Langfuse observation to an OTel span
func (obs *LangfuseObservation) ToOTelSpan() (*Span, error) {
    span := &Span{
        Name:              obs.Name,
        Kind:              mapObservationKind(obs.Type),
        StartTimeUnixNano: uint64(obs.StartTime.UnixNano()),
        Attributes:        make([]KeyValue, 0),
    }

    // Generate or parse IDs
    if err := parseOrGenerateTraceID(obs.TraceID, &span.TraceID); err != nil {
        return nil, err
    }
    if err := parseOrGenerateSpanID(obs.ID, &span.SpanID); err != nil {
        return nil, err
    }
    if obs.ParentObservationID != "" {
        parseOrGenerateSpanID(obs.ParentObservationID, &span.ParentSpanID)
    }

    // Set end time
    if obs.EndTime != nil {
        span.EndTimeUnixNano = uint64(obs.EndTime.UnixNano())
    }

    // Convert metadata to attributes
    for k, v := range obs.Metadata {
        span.Attributes = append(span.Attributes, toKeyValue(k, v))
    }

    // Add Langfuse-specific attributes
    span.Attributes = append(span.Attributes,
        stringKV("langfuse.observation.type", obs.Type),
        stringKV("langfuse.trace.id", obs.TraceID),
    )

    // Add GenAI semantic conventions for GENERATION type
    if obs.Type == "GENERATION" {
        span.Attributes = append(span.Attributes,
            stringKV("gen_ai.system", "langfuse"),
        )
        if obs.Model != "" {
            span.Attributes = append(span.Attributes,
                stringKV("gen_ai.request.model", obs.Model),
            )
        }
        if obs.Usage != nil {
            span.Attributes = append(span.Attributes,
                intKV("gen_ai.usage.input_tokens", obs.Usage.Input),
                intKV("gen_ai.usage.output_tokens", obs.Usage.Output),
            )
        }
    }

    // Map level to status
    span.Status = mapLevelToStatus(obs.Level, obs.StatusMessage)

    return span, nil
}

// FromOTelSpan converts an OTel span to Langfuse observation parameters
func FromOTelSpan(span *Span) *LangfuseObservation {
    obs := &LangfuseObservation{
        ID:        hex.EncodeToString(span.SpanID[:]),
        TraceID:   hex.EncodeToString(span.TraceID[:]),
        Name:      span.Name,
        StartTime: time.Unix(0, int64(span.StartTimeUnixNano)),
        Metadata:  make(map[string]interface{}),
        Type:      "SPAN", // Default to SPAN
    }

    if span.EndTimeUnixNano > 0 {
        endTime := time.Unix(0, int64(span.EndTimeUnixNano))
        obs.EndTime = &endTime
    }

    if !isZeroSpanID(span.ParentSpanID) {
        obs.ParentObservationID = hex.EncodeToString(span.ParentSpanID[:])
    }

    // Extract attributes
    for _, attr := range span.Attributes {
        key := attr.Key
        value := extractValue(attr.Value)

        // Check for GenAI conventions to identify GENERATION type
        switch key {
        case "gen_ai.request.model":
            obs.Type = "GENERATION"
            if s, ok := value.(string); ok {
                obs.Model = s
            }
        case "gen_ai.usage.input_tokens":
            if obs.Usage == nil {
                obs.Usage = &LangfuseUsage{}
            }
            if i, ok := value.(int64); ok {
                obs.Usage.Input = i
            }
        case "gen_ai.usage.output_tokens":
            if obs.Usage == nil {
                obs.Usage = &LangfuseUsage{}
            }
            if i, ok := value.(int64); ok {
                obs.Usage.Output = i
            }
        default:
            obs.Metadata[key] = value
        }
    }

    // Map status to level
    obs.Level, obs.StatusMessage = mapStatusToLevel(span.Status)

    return obs
}

// Helper functions

func mapObservationKind(obsType string) SpanKind {
    switch obsType {
    case "GENERATION":
        return SpanKindClient // LLM calls are typically client calls
    case "EVENT":
        return SpanKindInternal
    default:
        return SpanKindInternal
    }
}

func mapLevelToStatus(level, message string) Status {
    switch level {
    case "ERROR":
        return Status{Code: StatusCodeError, Message: message}
    case "WARNING":
        return Status{Code: StatusCodeUnset, Message: message}
    default:
        return Status{Code: StatusCodeOK, Message: message}
    }
}

func mapStatusToLevel(status Status) (level, message string) {
    message = status.Message
    switch status.Code {
    case StatusCodeError:
        level = "ERROR"
    case StatusCodeOK:
        level = "DEFAULT"
    default:
        level = "DEFAULT"
    }
    return
}

func stringKV(key, value string) KeyValue {
    return KeyValue{Key: key, Value: AnyValue{StringValue: &value}}
}

func intKV(key string, value int64) KeyValue {
    return KeyValue{Key: key, Value: AnyValue{IntValue: &value}}
}

func toKeyValue(key string, value interface{}) KeyValue {
    kv := KeyValue{Key: key}
    switch v := value.(type) {
    case string:
        kv.Value.StringValue = &v
    case bool:
        kv.Value.BoolValue = &v
    case int64:
        kv.Value.IntValue = &v
    case int:
        i64 := int64(v)
        kv.Value.IntValue = &i64
    case float64:
        kv.Value.DoubleValue = &v
    default:
        s := fmt.Sprintf("%v", v)
        kv.Value.StringValue = &s
    }
    return kv
}

func extractValue(v AnyValue) interface{} {
    switch {
    case v.StringValue != nil:
        return *v.StringValue
    case v.BoolValue != nil:
        return *v.BoolValue
    case v.IntValue != nil:
        return *v.IntValue
    case v.DoubleValue != nil:
        return *v.DoubleValue
    default:
        return nil
    }
}

func isZeroSpanID(id [8]byte) bool {
    for _, b := range id {
        if b != 0 {
            return false
        }
    }
    return true
}

func parseOrGenerateTraceID(id string, out *[16]byte) error {
    if id == "" {
        rand.Read(out[:])
        return nil
    }
    // Try parsing as hex (OTel format)
    if len(id) == 32 {
        bytes, err := hex.DecodeString(id)
        if err == nil {
            copy(out[:], bytes)
            return nil
        }
    }
    // Use deterministic hash of Langfuse ID
    hash := deterministicHash(id, 16)
    copy(out[:], hash)
    return nil
}

func parseOrGenerateSpanID(id string, out *[8]byte) error {
    if id == "" {
        rand.Read(out[:])
        return nil
    }
    if len(id) == 16 {
        bytes, err := hex.DecodeString(id)
        if err == nil {
            copy(out[:], bytes)
            return nil
        }
    }
    hash := deterministicHash(id, 8)
    copy(out[:], hash)
    return nil
}

// deterministicHash creates a deterministic hash of a string
func deterministicHash(s string, length int) []byte {
    // Simple deterministic hash (FNV-1a based)
    result := make([]byte, length)
    hash := uint64(14695981039346656037)
    for _, c := range s {
        hash ^= uint64(c)
        hash *= 1099511628211
    }
    for i := 0; i < length; i++ {
        result[i] = byte(hash >> (i * 8))
    }
    return result
}
```

### 2.4 OTLP/HTTP Exporter

Export Langfuse traces to OTel collectors using OTLP/HTTP with JSON.

```go
// otel/exporter.go

package otel

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

// OTLPExporter exports spans to an OTLP collector
type OTLPExporter struct {
    endpoint    string
    headers     map[string]string
    httpClient  *http.Client
    serviceName string
    serviceVersion string
}

// OTLPExporterOption configures the exporter
type OTLPExporterOption func(*OTLPExporter)

// NewOTLPExporter creates a new OTLP exporter
func NewOTLPExporter(endpoint string, opts ...OTLPExporterOption) *OTLPExporter {
    e := &OTLPExporter{
        endpoint:   endpoint,
        headers:    make(map[string]string),
        httpClient: &http.Client{Timeout: 30 * time.Second},
        serviceName: "langfuse-sdk",
        serviceVersion: "1.0.0",
    }
    for _, opt := range opts {
        opt(e)
    }
    return e
}

// WithHeaders sets custom headers (e.g., for authentication)
func WithHeaders(headers map[string]string) OTLPExporterOption {
    return func(e *OTLPExporter) {
        for k, v := range headers {
            e.headers[k] = v
        }
    }
}

// WithServiceName sets the service name for resource attributes
func WithServiceName(name string) OTLPExporterOption {
    return func(e *OTLPExporter) {
        e.serviceName = name
    }
}

// WithServiceVersion sets the service version
func WithServiceVersion(version string) OTLPExporterOption {
    return func(e *OTLPExporter) {
        e.serviceVersion = version
    }
}

// Export sends spans to the OTLP collector
func (e *OTLPExporter) Export(ctx context.Context, spans []*Span) error {
    if len(spans) == 0 {
        return nil
    }

    req := e.buildRequest(spans)
    body, err := json.Marshal(req)
    if err != nil {
        return fmt.Errorf("marshal OTLP request: %w", err)
    }

    httpReq, err := http.NewRequestWithContext(ctx, "POST", e.endpoint, bytes.NewReader(body))
    if err != nil {
        return fmt.Errorf("create HTTP request: %w", err)
    }

    httpReq.Header.Set("Content-Type", "application/json")
    for k, v := range e.headers {
        httpReq.Header.Set(k, v)
    }

    resp, err := e.httpClient.Do(httpReq)
    if err != nil {
        return fmt.Errorf("send OTLP request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("OTLP export failed: status=%d body=%s", resp.StatusCode, string(body))
    }

    return nil
}

func (e *OTLPExporter) buildRequest(spans []*Span) *ExportTraceServiceRequest {
    // Build resource attributes
    resource := Resource{
        Attributes: []KeyValue{
            stringKV("service.name", e.serviceName),
            stringKV("service.version", e.serviceVersion),
            stringKV("telemetry.sdk.name", "langfuse-go"),
            stringKV("telemetry.sdk.language", "go"),
        },
    }

    // Convert spans
    otelSpans := make([]Span, len(spans))
    for i, s := range spans {
        otelSpans[i] = *s
    }

    return &ExportTraceServiceRequest{
        ResourceSpans: []ResourceSpans{
            {
                Resource: resource,
                ScopeSpans: []ScopeSpans{
                    {
                        Scope: InstrumentationScope{
                            Name:    "langfuse-go-sdk",
                            Version: "1.0.0",
                        },
                        Spans: otelSpans,
                    },
                },
            },
        },
    }
}

// Shutdown performs cleanup
func (e *OTLPExporter) Shutdown(ctx context.Context) error {
    return nil
}
```

---

## 3. Client Integration

### 3.1 Configuration Options

```go
// config.go additions

// OTelConfig holds OpenTelemetry configuration
type OTelConfig struct {
    // Enable OTel export alongside Langfuse export
    Enabled bool

    // OTLP endpoint (e.g., "http://localhost:4318/v1/traces")
    Endpoint string

    // Custom headers for OTLP requests
    Headers map[string]string

    // Service identification
    ServiceName    string
    ServiceVersion string

    // Export options
    ExportBatchSize  int           // Default: 100
    ExportInterval   time.Duration // Default: 5s

    // Propagation
    PropagateContext bool // Enable W3C trace context propagation
}

// WithOTelExport enables OTel export
func WithOTelExport(config OTelConfig) ConfigOption {
    return func(c *Config) {
        c.OTel = &config
    }
}

// WithOTelEndpoint is a convenience method for basic OTLP setup
func WithOTelEndpoint(endpoint string) ConfigOption {
    return func(c *Config) {
        if c.OTel == nil {
            c.OTel = &OTelConfig{}
        }
        c.OTel.Enabled = true
        c.OTel.Endpoint = endpoint
    }
}
```

### 3.2 Client Methods for Context Propagation

```go
// client.go additions

// TraceContextFromHeaders extracts W3C trace context from HTTP headers
// Use this when receiving requests with upstream trace context
func (c *Client) TraceContextFromHeaders(headers http.Header) *otel.TraceContext {
    propagator := &otel.Propagator{}
    tc, _ := propagator.Extract(headers)
    return tc
}

// NewTraceWithContext creates a trace linked to an existing OTel context
func (c *Client) NewTraceWithContext(tc *otel.TraceContext) *TraceBuilder {
    builder := c.Traces().Create()
    if tc != nil {
        // Set trace ID to match OTel trace
        builder.WithID(hex.EncodeToString(tc.TraceID[:]))
        // Store parent span for linking
        builder.WithMetadata(map[string]interface{}{
            "otel.parent_span_id": hex.EncodeToString(tc.SpanID[:]),
            "otel.trace_state":    tc.TraceState,
        })
    }
    return builder
}

// InjectTraceContext injects the current trace context into HTTP headers
// Use this when making outbound HTTP requests
func (ctx *TraceContext) InjectHeaders(headers http.Header) {
    if ctx.traceContext != nil {
        propagator := &otel.Propagator{}
        propagator.Inject(ctx.traceContext, headers)
    }
}
```

### 3.3 Dual Export Pipeline

```go
// client.go modifications

func (c *Client) startProcessors() {
    // ... existing code ...

    // Start OTel exporter if configured
    if c.config.OTel != nil && c.config.OTel.Enabled {
        c.otelExporter = otel.NewOTLPExporter(
            c.config.OTel.Endpoint,
            otel.WithHeaders(c.config.OTel.Headers),
            otel.WithServiceName(c.config.OTel.ServiceName),
            otel.WithServiceVersion(c.config.OTel.ServiceVersion),
        )
        go c.otelExportLoop()
    }
}

func (c *Client) otelExportLoop() {
    interval := c.config.OTel.ExportInterval
    if interval == 0 {
        interval = 5 * time.Second
    }
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            c.exportToOTel()
        case <-c.shutdown:
            c.exportToOTel() // Final export
            return
        }
    }
}

func (c *Client) exportToOTel() {
    c.otelMu.Lock()
    spans := c.pendingOTelSpans
    c.pendingOTelSpans = nil
    c.otelMu.Unlock()

    if len(spans) == 0 {
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := c.otelExporter.Export(ctx, spans); err != nil {
        if c.config.Logger != nil {
            c.config.Logger.Printf("OTel export error: %v", err)
        }
    }
}
```

---

## 4. GenAI Semantic Conventions

Support OpenTelemetry's emerging GenAI semantic conventions for LLM observability.

### 4.1 Attribute Mapping

| Langfuse Field | OTel GenAI Attribute |
|----------------|---------------------|
| `Model` | `gen_ai.request.model` |
| `ModelParameters.temperature` | `gen_ai.request.temperature` |
| `ModelParameters.max_tokens` | `gen_ai.request.max_tokens` |
| `ModelParameters.top_p` | `gen_ai.request.top_p` |
| `Usage.Input` | `gen_ai.usage.input_tokens` |
| `Usage.Output` | `gen_ai.usage.output_tokens` |
| `CompletionStartTime` | `gen_ai.response.first_token_time` |
| `Input` (prompt) | Span event: `gen_ai.prompt` |
| `Output` (completion) | Span event: `gen_ai.completion` |

### 4.2 Span Events for Prompts/Completions

```go
// For GENERATION observations, add span events
func (obs *LangfuseObservation) addGenAIEvents(span *Span) {
    if obs.Type != "GENERATION" {
        return
    }

    // Prompt event
    if obs.Input != nil {
        promptEvent := Event{
            Name:         "gen_ai.prompt",
            TimeUnixNano: span.StartTimeUnixNano,
            Attributes: []KeyValue{
                stringKV("gen_ai.prompt.content", fmt.Sprintf("%v", obs.Input)),
            },
        }
        span.Events = append(span.Events, promptEvent)
    }

    // Completion event
    if obs.Output != nil && span.EndTimeUnixNano > 0 {
        completionEvent := Event{
            Name:         "gen_ai.completion",
            TimeUnixNano: span.EndTimeUnixNano,
            Attributes: []KeyValue{
                stringKV("gen_ai.completion.content", fmt.Sprintf("%v", obs.Output)),
            },
        }
        span.Events = append(span.Events, completionEvent)
    }
}
```

---

## 5. Usage Examples

### 5.1 Basic OTel Export

```go
client, _ := langfuse.NewClient(
    langfuse.WithPublicKey("pk-xxx"),
    langfuse.WithSecretKey("sk-xxx"),
    langfuse.WithOTelEndpoint("http://localhost:4318/v1/traces"),
)
defer client.Shutdown(ctx)

// Create traces as normal - they export to both Langfuse and OTel
trace := client.Traces().Create().
    WithName("my-request").
    Create()

gen := trace.Generation().
    WithName("openai-call").
    WithModel("gpt-4").
    Create()

gen.EndWithUsage(output, langfuse.Usage{Input: 100, Output: 50})
```

### 5.2 Distributed Tracing (Incoming Request)

```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    // Extract upstream trace context
    tc := client.TraceContextFromHeaders(r.Header)

    // Create trace linked to upstream
    trace := client.NewTraceWithContext(tc).
        WithName("api-request").
        Create()
    defer trace.Update().WithOutput(response).Apply()

    // ... handle request
}
```

### 5.3 Distributed Tracing (Outgoing Request)

```go
trace := client.Traces().Create().WithName("my-service").Create()

// Make downstream call with trace context
req, _ := http.NewRequest("GET", "http://downstream/api", nil)
trace.InjectHeaders(req.Header)
resp, _ := http.DefaultClient.Do(req)
```

### 5.4 OTel Collector Configuration

```yaml
# otel-collector-config.yaml
receivers:
  otlp:
    protocols:
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 5s

exporters:
  jaeger:
    endpoint: "jaeger:14250"
  prometheus:
    endpoint: "0.0.0.0:8889"

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [jaeger]
```

---

## 6. Implementation Phases

### Phase 1: Core Types & Propagation
- OTLP JSON data structures
- W3C Trace Context parser/generator
- Unit tests for serialization

### Phase 2: Type Bridge
- Langfuse → OTel conversion
- OTel → Langfuse conversion
- GenAI semantic conventions mapping

### Phase 3: OTLP Exporter
- HTTP/JSON exporter implementation
- Batching and retry logic
- Integration with client lifecycle

### Phase 4: Client Integration
- Configuration options
- Dual export pipeline
- Context propagation helpers

### Phase 5: Testing & Documentation
- Integration tests with OTel collector
- Examples and migration guide
- Performance benchmarks

---

## 7. Alternatives Considered

### 7.1 Use OTLP/Proto (Protobuf)

**Pros:** More efficient encoding, official format
**Cons:** Requires protobuf dependency (google.golang.org/protobuf)
**Decision:** Rejected to maintain zero-dependency goal

### 7.2 gRPC Transport

**Pros:** Streaming, better for high-volume
**Cons:** Requires grpc-go (~10MB+ binary size increase)
**Decision:** Rejected; HTTP/JSON sufficient for most use cases

### 7.3 Full OTel SDK Integration

**Pros:** Complete compatibility
**Cons:** Heavy dependency, complex API surface
**Decision:** Rejected; bridge approach provides 80/20 value

---

## 8. Testing Strategy

```go
// otel/bridge_test.go

func TestLangfuseToOTelConversion(t *testing.T) {
    obs := &LangfuseObservation{
        ID:        "obs-123",
        TraceID:   "trace-456",
        Type:      "GENERATION",
        Name:      "gpt-4-call",
        Model:     "gpt-4",
        StartTime: time.Now(),
        Usage:     &LangfuseUsage{Input: 100, Output: 50},
    }

    span, err := obs.ToOTelSpan()
    require.NoError(t, err)

    assert.Equal(t, "gpt-4-call", span.Name)
    assert.Equal(t, SpanKindClient, span.Kind)

    // Verify GenAI attributes
    modelAttr := findAttribute(span.Attributes, "gen_ai.request.model")
    assert.Equal(t, "gpt-4", *modelAttr.Value.StringValue)
}

func TestTraceparentParsing(t *testing.T) {
    cases := []struct {
        input   string
        valid   bool
        sampled bool
    }{
        {"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", true, true},
        {"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-00", true, false},
        {"invalid", false, false},
    }

    for _, tc := range cases {
        ctx, err := ParseTraceparent(tc.input)
        if tc.valid {
            require.NoError(t, err)
            assert.Equal(t, tc.sampled, ctx.IsSampled())
        } else {
            require.Error(t, err)
        }
    }
}
```

---

## 9. Open Questions

1. **ID Mapping Strategy**: Should we use deterministic hashing of Langfuse UUIDs to OTel trace/span IDs, or generate new random IDs and store the mapping in attributes?

2. **Event vs Span for Langfuse Events**: Should Langfuse `EVENT` observations become OTel span events or zero-duration spans?

3. **Score Export**: Should Langfuse scores be exported as OTel span attributes, metrics, or not at all?

4. **Bidirectional Sync**: Should we support importing OTel spans into Langfuse via HTTP handler, or is export-only sufficient?

---

## 10. Appendix: OTLP JSON Wire Format

Example OTLP/HTTP JSON request:

```json
{
  "resourceSpans": [{
    "resource": {
      "attributes": [
        {"key": "service.name", "value": {"stringValue": "my-service"}},
        {"key": "service.version", "value": {"stringValue": "1.0.0"}}
      ]
    },
    "scopeSpans": [{
      "scope": {
        "name": "langfuse-go-sdk",
        "version": "1.0.0"
      },
      "spans": [{
        "traceId": "4bf92f3577b34da6a3ce929d0e0e4736",
        "spanId": "00f067aa0ba902b7",
        "parentSpanId": "",
        "name": "gpt-4-completion",
        "kind": 3,
        "startTimeUnixNano": "1703520000000000000",
        "endTimeUnixNano": "1703520001500000000",
        "attributes": [
          {"key": "gen_ai.request.model", "value": {"stringValue": "gpt-4"}},
          {"key": "gen_ai.usage.input_tokens", "value": {"intValue": "150"}},
          {"key": "gen_ai.usage.output_tokens", "value": {"intValue": "89"}}
        ],
        "status": {"code": 1}
      }]
    }]
  }]
}
```
