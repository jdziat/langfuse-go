package langfuse

// This file exports internal symbols for testing purposes.
// It is only compiled when running tests.

// HTTPClient exports httpClient for testing.
type HTTPClient = httpClient

// NewHTTPClientForTest exports newHTTPClient for testing.
var NewHTTPClientForTest = newHTTPClient

// IngestionEvent exports ingestionEvent for testing.
type IngestionEvent = ingestionEvent

// IngestionRequest exports ingestionRequest for testing.
type IngestionRequest = ingestionRequest

// TraceEventExport exports traceEvent for testing.
type TraceEventExport = traceEvent

// ObservationEventExport exports observationEvent for testing.
type ObservationEventExport = observationEvent

// ScoreEventExport exports scoreEvent for testing.
type ScoreEventExport = scoreEvent

// CombineHooksForTest exports combineHooks for testing.
var CombineHooksForTest = combineHooks

// GenerateIDInternalForTest exports generateIDInternal for testing.
var GenerateIDInternalForTest = generateIDInternal

// IsHexStringForTest exports isHexString for testing.
var IsHexStringForTest = isHexString

// ParseRetryAfterForTest exports parseRetryAfter for testing.
var ParseRetryAfterForTest = parseRetryAfter

// GenerateRequestIDForTest exports generateRequestID for testing.
var GenerateRequestIDForTest = generateRequestID

// Event type constants for testing.
const (
	EventTypeTraceCreateForTest      = eventTypeTraceCreate
	EventTypeTraceUpdateForTest      = eventTypeTraceUpdate
	EventTypeSpanCreateForTest       = eventTypeSpanCreate
	EventTypeSpanUpdateForTest       = eventTypeSpanUpdate
	EventTypeGenerationCreateForTest = eventTypeGenerationCreate
	EventTypeGenerationUpdateForTest = eventTypeGenerationUpdate
	EventTypeEventCreateForTest      = eventTypeEventCreate
	EventTypeScoreCreateForTest      = eventTypeScoreCreate
	EventTypeSDKLogForTest           = eventTypeSDKLog
)

// MaxResponseSizeForTest exports maxResponseSize for testing.
const MaxResponseSizeForTest = maxResponseSize

// RequestForTest exports the request type for testing.
type RequestForTest = request

// RootConfig returns the root-specific config from a Client for testing.
func (c *Client) RootConfig() *Config {
	return c.rootConfig
}

// HandleError calls the internal handleRootError method for testing.
func (c *Client) HandleError(err error) {
	c.handleRootError(err)
}
