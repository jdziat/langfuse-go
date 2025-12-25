package langfuse

import (
	"context"
	"sync"
	"time"
)

// Client is the main Langfuse client.
type Client struct {
	config *Config
	http   *httpClient

	// Batching
	mu            sync.Mutex
	pendingEvents []ingestionEvent
	flushTimer    *time.Timer
	closed        bool

	// Sub-clients
	traces       *TracesClient
	observations *ObservationsClient
	scores       *ScoresClient
	prompts      *PromptsClient
	datasets     *DatasetsClient
	sessions     *SessionsClient
	models       *ModelsClient
}

// New creates a new Langfuse client.
func New(publicKey, secretKey string, opts ...ConfigOption) (*Client, error) {
	cfg := &Config{
		PublicKey: publicKey,
		SecretKey: secretKey,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	cfg.applyDefaults()

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	httpClient := newHTTPClient(cfg)

	c := &Client{
		config:        cfg,
		http:          httpClient,
		pendingEvents: make([]ingestionEvent, 0, cfg.BatchSize),
	}

	// Initialize sub-clients
	c.traces = &TracesClient{client: c}
	c.observations = &ObservationsClient{client: c}
	c.scores = &ScoresClient{client: c}
	c.prompts = &PromptsClient{client: c}
	c.datasets = &DatasetsClient{client: c}
	c.sessions = &SessionsClient{client: c}
	c.models = &ModelsClient{client: c}

	// Start flush timer
	c.flushTimer = time.AfterFunc(cfg.FlushInterval, c.flushLoop)

	return c, nil
}

// flushLoop is called periodically to flush pending events.
func (c *Client) flushLoop() {
	c.Flush(context.Background())

	c.mu.Lock()
	if !c.closed {
		c.flushTimer = time.AfterFunc(c.config.FlushInterval, c.flushLoop)
	}
	c.mu.Unlock()
}

// Traces returns the traces sub-client.
func (c *Client) Traces() *TracesClient {
	return c.traces
}

// Observations returns the observations sub-client.
func (c *Client) Observations() *ObservationsClient {
	return c.observations
}

// Scores returns the scores sub-client.
func (c *Client) Scores() *ScoresClient {
	return c.scores
}

// Prompts returns the prompts sub-client.
func (c *Client) Prompts() *PromptsClient {
	return c.prompts
}

// Datasets returns the datasets sub-client.
func (c *Client) Datasets() *DatasetsClient {
	return c.datasets
}

// Sessions returns the sessions sub-client.
func (c *Client) Sessions() *SessionsClient {
	return c.sessions
}

// Models returns the models sub-client.
func (c *Client) Models() *ModelsClient {
	return c.models
}

// Health checks the health of the Langfuse API.
func (c *Client) Health(ctx context.Context) (*HealthStatus, error) {
	var result HealthStatus
	err := c.http.get(ctx, "/health", nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Flush sends all pending events to the Langfuse API.
func (c *Client) Flush(ctx context.Context) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return ErrClientClosed
	}
	if len(c.pendingEvents) == 0 {
		c.mu.Unlock()
		return nil
	}

	events := c.pendingEvents
	c.pendingEvents = make([]ingestionEvent, 0, c.config.BatchSize)
	c.mu.Unlock()

	return c.sendBatch(ctx, events)
}

// Shutdown flushes pending events and closes the client.
func (c *Client) Shutdown(ctx context.Context) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return ErrClientClosed
	}
	c.closed = true
	if c.flushTimer != nil {
		c.flushTimer.Stop()
	}

	// Flush pending events before closing
	if len(c.pendingEvents) == 0 {
		c.mu.Unlock()
		return nil
	}

	events := c.pendingEvents
	c.pendingEvents = nil
	c.mu.Unlock()

	return c.sendBatch(ctx, events)
}

// Close is an alias for Shutdown.
func (c *Client) Close(ctx context.Context) error {
	return c.Shutdown(ctx)
}

// queueEvent adds an event to the pending queue.
func (c *Client) queueEvent(event ingestionEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrClientClosed
	}

	c.pendingEvents = append(c.pendingEvents, event)

	// Flush if batch is full
	if len(c.pendingEvents) >= c.config.BatchSize {
		events := c.pendingEvents
		c.pendingEvents = make([]ingestionEvent, 0, c.config.BatchSize)

		// Send asynchronously
		go func() {
			c.sendBatch(context.Background(), events)
		}()
	}

	return nil
}

// sendBatch sends a batch of events to the API.
func (c *Client) sendBatch(ctx context.Context, events []ingestionEvent) error {
	if len(events) == 0 {
		return nil
	}

	req := &ingestionRequest{
		Batch: events,
	}

	var result IngestionResult
	err := c.http.post(ctx, "/ingestion", req, &result)
	if err != nil {
		return err
	}

	// Log errors if any
	if c.config.Debug && result.HasErrors() {
		for _, e := range result.Errors {
			// In a real implementation, you might want to use a logger
			_ = e
		}
	}

	return nil
}

// TraceBuilder provides a fluent interface for creating traces.
type TraceBuilder struct {
	client *Client
	trace  *createTraceEvent
}

// NewTrace creates a new trace builder.
func (c *Client) NewTrace() *TraceBuilder {
	return &TraceBuilder{
		client: c,
		trace: &createTraceEvent{
			ID:        generateID(),
			Timestamp: Now(),
		},
	}
}

// ID sets the trace ID.
func (b *TraceBuilder) ID(id string) *TraceBuilder {
	b.trace.ID = id
	return b
}

// Name sets the trace name.
func (b *TraceBuilder) Name(name string) *TraceBuilder {
	b.trace.Name = name
	return b
}

// UserID sets the user ID.
func (b *TraceBuilder) UserID(userID string) *TraceBuilder {
	b.trace.UserID = userID
	return b
}

// SessionID sets the session ID.
func (b *TraceBuilder) SessionID(sessionID string) *TraceBuilder {
	b.trace.SessionID = sessionID
	return b
}

// Input sets the trace input.
func (b *TraceBuilder) Input(input interface{}) *TraceBuilder {
	b.trace.Input = input
	return b
}

// Output sets the trace output.
func (b *TraceBuilder) Output(output interface{}) *TraceBuilder {
	b.trace.Output = output
	return b
}

// Metadata sets the trace metadata.
func (b *TraceBuilder) Metadata(metadata map[string]interface{}) *TraceBuilder {
	b.trace.Metadata = metadata
	return b
}

// Tags sets the trace tags.
func (b *TraceBuilder) Tags(tags []string) *TraceBuilder {
	b.trace.Tags = tags
	return b
}

// Release sets the release version.
func (b *TraceBuilder) Release(release string) *TraceBuilder {
	b.trace.Release = release
	return b
}

// Version sets the version.
func (b *TraceBuilder) Version(version string) *TraceBuilder {
	b.trace.Version = version
	return b
}

// Public sets whether the trace is public.
func (b *TraceBuilder) Public(public bool) *TraceBuilder {
	b.trace.Public = public
	return b
}

// Environment sets the environment.
func (b *TraceBuilder) Environment(env string) *TraceBuilder {
	b.trace.Environment = env
	return b
}

// Create creates the trace and returns a TraceContext for adding observations.
func (b *TraceBuilder) Create() (*TraceContext, error) {
	event := ingestionEvent{
		ID:        generateID(),
		Type:      eventTypeTraceCreate,
		Timestamp: Now(),
		Body:      b.trace,
	}

	if err := b.client.queueEvent(event); err != nil {
		return nil, err
	}

	return &TraceContext{
		client:  b.client,
		traceID: b.trace.ID,
	}, nil
}

// TraceContext provides context for a trace and allows adding observations.
type TraceContext struct {
	client  *Client
	traceID string
}

// ID returns the trace ID.
func (t *TraceContext) ID() string {
	return t.traceID
}

// Update updates the trace.
func (t *TraceContext) Update() *TraceUpdateBuilder {
	return &TraceUpdateBuilder{
		ctx: t,
		update: &updateTraceEvent{
			ID: t.traceID,
		},
	}
}

// Span creates a new span in this trace.
func (t *TraceContext) Span() *SpanBuilder {
	return &SpanBuilder{
		ctx: t,
		span: &createSpanEvent{
			ID:        generateID(),
			TraceID:   t.traceID,
			StartTime: Now(),
		},
	}
}

// Generation creates a new generation in this trace.
func (t *TraceContext) Generation() *GenerationBuilder {
	return &GenerationBuilder{
		ctx: t,
		gen: &createGenerationEvent{
			ID:        generateID(),
			TraceID:   t.traceID,
			StartTime: Now(),
		},
	}
}

// Event creates a new event in this trace.
func (t *TraceContext) Event() *EventBuilder {
	return &EventBuilder{
		ctx: t,
		event: &createEventEvent{
			ID:        generateID(),
			TraceID:   t.traceID,
			StartTime: Now(),
		},
	}
}

// Score creates a new score for this trace.
func (t *TraceContext) Score() *ScoreBuilder {
	return &ScoreBuilder{
		ctx: t,
		score: &createScoreEvent{
			TraceID: t.traceID,
		},
	}
}

// TraceUpdateBuilder provides a fluent interface for updating traces.
type TraceUpdateBuilder struct {
	ctx    *TraceContext
	update *updateTraceEvent
}

// Name sets the trace name.
func (b *TraceUpdateBuilder) Name(name string) *TraceUpdateBuilder {
	b.update.Name = name
	return b
}

// UserID sets the user ID.
func (b *TraceUpdateBuilder) UserID(userID string) *TraceUpdateBuilder {
	b.update.UserID = userID
	return b
}

// SessionID sets the session ID.
func (b *TraceUpdateBuilder) SessionID(sessionID string) *TraceUpdateBuilder {
	b.update.SessionID = sessionID
	return b
}

// Input sets the trace input.
func (b *TraceUpdateBuilder) Input(input interface{}) *TraceUpdateBuilder {
	b.update.Input = input
	return b
}

// Output sets the trace output.
func (b *TraceUpdateBuilder) Output(output interface{}) *TraceUpdateBuilder {
	b.update.Output = output
	return b
}

// Metadata sets the trace metadata.
func (b *TraceUpdateBuilder) Metadata(metadata map[string]interface{}) *TraceUpdateBuilder {
	b.update.Metadata = metadata
	return b
}

// Tags sets the trace tags.
func (b *TraceUpdateBuilder) Tags(tags []string) *TraceUpdateBuilder {
	b.update.Tags = tags
	return b
}

// Public sets whether the trace is public.
func (b *TraceUpdateBuilder) Public(public bool) *TraceUpdateBuilder {
	b.update.Public = public
	return b
}

// Apply applies the update.
func (b *TraceUpdateBuilder) Apply() error {
	event := ingestionEvent{
		ID:        generateID(),
		Type:      eventTypeTraceUpdate,
		Timestamp: Now(),
		Body:      b.update,
	}

	return b.ctx.client.queueEvent(event)
}
