# Production Deployment Guide

This guide provides recommendations for deploying the Langfuse Go SDK in production environments.

## Table of Contents

1. [Resource Requirements](#resource-requirements)
2. [Configuration Profiles](#configuration-profiles)
3. [Monitoring and Observability](#monitoring-and-observability)
4. [Failure Scenarios and Recovery](#failure-scenarios-and-recovery)
5. [Performance Tuning](#performance-tuning)
6. [Security Considerations](#security-considerations)

---

## Resource Requirements

### Memory

The SDK's memory usage depends primarily on:

| Component | Memory Impact | Scaling Factor |
|-----------|---------------|----------------|
| Event Queue | ~1KB per event | `BatchQueueSize × BatchSize × 1KB` |
| Pending Events | ~1KB per event | Up to `BatchSize` events |
| HTTP Connections | ~10KB per connection | `MaxIdleConns × 10KB` |
| Async Error Buffer | ~100B per error | `AsyncErrorConfig.BufferSize × 100B` |

**Memory Estimation Formula:**

```
Base: ~5MB
+ (BatchQueueSize × BatchSize × 1KB)  # Event queue
+ (MaxIdleConns × 10KB)               # Connection pool
```

**Example Calculations:**

| Profile | BatchQueueSize | BatchSize | MaxIdleConns | Estimated Memory |
|---------|----------------|-----------|--------------|------------------|
| Default | 100 | 100 | 100 | ~15MB |
| High Throughput | 500 | 500 | 200 | ~255MB |
| Low Memory | 50 | 50 | 20 | ~7MB |

### CPU

The SDK is primarily I/O-bound. CPU usage is minimal:

- **Idle**: < 1% (periodic flush timer)
- **Active ingestion**: 1-3% per 1000 events/second
- **JSON serialization**: Main CPU consumer during high throughput

### Network

- **Connections**: Up to `MaxIdleConns` concurrent connections
- **Bandwidth**: ~500 bytes per event average (varies with metadata size)
- **Latency sensitivity**: The SDK handles latency gracefully via async batching

---

## Configuration Profiles

### Low Volume (< 100 events/minute)

```go
client, err := langfuse.New(publicKey, secretKey,
    langfuse.WithRegion(langfuse.RegionUS),
    langfuse.WithBatchSize(50),
    langfuse.WithFlushInterval(10*time.Second),
    langfuse.WithBatchQueueSize(50),
)
```

**Characteristics:**
- Smaller batches, less memory
- Longer flush interval to maximize batching
- Suitable for development, staging, or low-traffic services

### Medium Volume (100-1000 events/minute)

```go
client, err := langfuse.New(publicKey, secretKey,
    langfuse.WithRegion(langfuse.RegionUS),
    langfuse.WithBatchSize(100),
    langfuse.WithFlushInterval(5*time.Second),
    langfuse.WithBatchQueueSize(100),
    langfuse.WithDefaultCircuitBreaker(),
)
```

**Characteristics:**
- Default configuration works well
- Circuit breaker enabled for resilience
- Suitable for most production workloads

### High Volume (> 1000 events/minute)

```go
client, err := langfuse.New(publicKey, secretKey,
    langfuse.WithRegion(langfuse.RegionUS),
    langfuse.WithBatchSize(500),
    langfuse.WithFlushInterval(10*time.Second),
    langfuse.WithBatchQueueSize(500),
    langfuse.WithMaxIdleConns(200),
    langfuse.WithMaxIdleConnsPerHost(50),
    langfuse.WithDefaultCircuitBreaker(),
    langfuse.WithOnBackpressure(func(state langfuse.QueueState) {
        if state.Level >= langfuse.BackpressureCritical {
            metrics.Increment("langfuse.backpressure.critical")
        }
    }),
)
```

Or use the pre-configured profile:

```go
cfg := langfuse.HighThroughputConfig(publicKey, secretKey)
client, err := langfuse.NewWithConfig(cfg)
```

**Characteristics:**
- Large batches for efficiency
- Backpressure monitoring
- More connections for parallel requests
- Suitable for high-traffic production systems

---

## Monitoring and Observability

### Key Metrics to Monitor

The SDK exposes internal metrics via the `Metrics` interface. Enable them with:

```go
client, err := langfuse.New(publicKey, secretKey,
    langfuse.WithMetrics(myMetricsImpl),
    langfuse.WithMetricsRecorder(),
)
```

**Critical Metrics:**

| Metric | Description | Alert Threshold |
|--------|-------------|-----------------|
| `langfuse.queue.size` | Current queue depth | > 80% capacity |
| `langfuse.queue.dropped` | Events dropped | Any non-zero |
| `langfuse.batch.failures` | Failed batch sends | > 5% of batches |
| `langfuse.circuit.state` | Circuit breaker state | 1 (half-open) or 2 (open) |
| `langfuse.http.5xx` | Server error responses | > 1% of requests |

**Informational Metrics:**

| Metric | Description | Purpose |
|--------|-------------|---------|
| `langfuse.events.queued` | Total events queued | Track throughput |
| `langfuse.events.sent` | Total events sent | Track delivery |
| `langfuse.batch.duration_ms` | Batch send latency | Performance baseline |
| `langfuse.http.2xx` | Successful responses | Verify delivery |

### Implementing a Metrics Adapter

```go
type PrometheusMetrics struct {
    counters   map[string]prometheus.Counter
    gauges     map[string]prometheus.Gauge
    histograms map[string]prometheus.Histogram
}

func (p *PrometheusMetrics) IncrementCounter(name string, value int64) {
    if counter, ok := p.counters[name]; ok {
        counter.Add(float64(value))
    }
}

func (p *PrometheusMetrics) SetGauge(name string, value float64) {
    if gauge, ok := p.gauges[name]; ok {
        gauge.Set(value)
    }
}

func (p *PrometheusMetrics) RecordDuration(name string, d time.Duration) {
    if hist, ok := p.histograms[name]; ok {
        hist.Observe(d.Seconds())
    }
}
```

### Structured Logging

Enable structured logging for production:

```go
import "log/slog"

client, err := langfuse.New(publicKey, secretKey,
    langfuse.WithStructuredLogger(langfuse.NewSlogAdapter(
        slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
            Level: slog.LevelInfo,
        })),
    )),
)
```

### Batch Result Monitoring

Monitor batch outcomes:

```go
client, err := langfuse.New(publicKey, secretKey,
    langfuse.WithOnBatchFlushed(func(result langfuse.BatchResult) {
        if !result.Success {
            slog.Error("batch failed",
                "error", result.Error,
                "events", result.EventCount,
                "duration", result.Duration,
            )
        } else {
            slog.Info("batch sent",
                "events", result.EventCount,
                "successes", result.Successes,
                "errors", result.Errors,
                "duration", result.Duration,
            )
        }
    }),
)
```

### Async Error Monitoring

Monitor background operation failures:

```go
client, err := langfuse.New(publicKey, secretKey,
    langfuse.WithOnAsyncError(func(err *langfuse.AsyncError) {
        slog.Error("async error",
            "operation", err.Operation,
            "error", err.Err,
            "retryable", err.Retryable,
        )

        // Alert on critical errors
        if !err.Retryable {
            alerting.Notify("langfuse-critical", err.Error())
        }
    }),
)
```

---

## Failure Scenarios and Recovery

### Circuit Breaker Behavior

The circuit breaker protects your application from cascading failures:

```go
client, err := langfuse.New(publicKey, secretKey,
    langfuse.WithCircuitBreaker(langfuse.CircuitBreakerConfig{
        FailureThreshold:    5,   // Open after 5 consecutive failures
        SuccessThreshold:    2,   // Close after 2 successes in half-open
        Timeout:             30 * time.Second, // Stay open for 30s
        HalfOpenMaxRequests: 3,   // Allow 3 test requests in half-open
        OnStateChange: func(from, to langfuse.CircuitState) {
            slog.Warn("circuit breaker state change",
                "from", from.String(),
                "to", to.String(),
            )
        },
    }),
)
```

**States:**

| State | Behavior | Transition |
|-------|----------|------------|
| Closed | Normal operation | Opens after `FailureThreshold` failures |
| Open | All requests fail fast | Transitions to Half-Open after `Timeout` |
| Half-Open | Limited test requests | Closes after `SuccessThreshold` successes |

### Queue Backpressure

When the event queue fills up:

```go
client, err := langfuse.New(publicKey, secretKey,
    langfuse.WithBackpressureThreshold(langfuse.BackpressureThreshold{
        WarningPercent:  50.0,  // Log warning at 50% full
        CriticalPercent: 80.0,  // Alert at 80% full
        OverflowPercent: 95.0,  // Events may be dropped at 95%
    }),
    langfuse.WithOnBackpressure(func(state langfuse.QueueState) {
        switch state.Level {
        case langfuse.BackpressureWarning:
            slog.Warn("queue filling up", "percent", state.PercentFull)
        case langfuse.BackpressureCritical:
            slog.Error("queue nearly full", "percent", state.PercentFull)
            metrics.Increment("langfuse.backpressure.critical")
        case langfuse.BackpressureOverflow:
            slog.Error("queue overflow - events may be dropped")
            alerting.Notify("langfuse-overflow", "Queue at capacity")
        }
    }),
    // Optional: Block instead of dropping events
    langfuse.WithBlockOnQueueFull(true),
)
```

### Retry Behavior

The SDK uses exponential backoff for retries:

```go
// Default behavior:
// - 3 retries with exponential backoff
// - Initial delay: 1 second
// - Maximum delay: 30 seconds
// - Jitter applied to prevent thundering herd

// Custom retry strategy:
client, err := langfuse.New(publicKey, secretKey,
    langfuse.WithMaxRetries(5),
    langfuse.WithRetryDelay(2*time.Second),
)
```

### Graceful Shutdown

Always shut down the client properly:

```go
// In main.go
func main() {
    client, err := langfuse.New(publicKey, secretKey)
    if err != nil {
        log.Fatal(err)
    }

    // Handle shutdown signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigChan
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        if err := client.Shutdown(ctx); err != nil {
            slog.Error("shutdown failed", "error", err)
        }
        os.Exit(0)
    }()

    // ... rest of application
}
```

### Network Failures

The SDK handles network failures gracefully:

| Scenario | SDK Behavior | Recovery |
|----------|--------------|----------|
| Connection refused | Retry with backoff | Automatic |
| DNS resolution failure | Retry with backoff | Automatic |
| TLS handshake failure | Retry with backoff | Automatic |
| Request timeout | Retry with backoff | Automatic |
| 5xx responses | Retry with backoff | Automatic |
| 429 rate limit | Respect Retry-After header | Automatic |
| 4xx client errors | No retry (permanent failure) | Fix request |

---

## Performance Tuning

### Batch Size Optimization

| Scenario | Recommended BatchSize | Rationale |
|----------|----------------------|-----------|
| Low latency needed | 50-100 | Smaller batches = faster delivery |
| High throughput | 500-1000 | Larger batches = fewer HTTP requests |
| Mixed workload | 100-200 | Balance between latency and efficiency |

### Connection Pool Sizing

```go
// For high concurrency:
client, err := langfuse.New(publicKey, secretKey,
    langfuse.WithMaxIdleConns(200),       // Total across all hosts
    langfuse.WithMaxIdleConnsPerHost(50), // Per host (Langfuse API)
    langfuse.WithIdleConnTimeout(90*time.Second),
)
```

**Guidelines:**

| Metric | Calculation |
|--------|-------------|
| MaxIdleConnsPerHost | `ceil(events_per_second / batch_size / 10)` |
| MaxIdleConns | `MaxIdleConnsPerHost × number_of_hosts` |

### Flush Interval Tradeoffs

| Interval | Pros | Cons |
|----------|------|------|
| 1-2s | Low latency, real-time visibility | More HTTP requests |
| 5s (default) | Good balance | Moderate latency |
| 10-30s | Maximum batching efficiency | Higher latency |

### Memory Optimization

For memory-constrained environments:

```go
client, err := langfuse.New(publicKey, secretKey,
    langfuse.WithBatchSize(50),
    langfuse.WithBatchQueueSize(50),
    langfuse.WithMaxIdleConns(20),
    langfuse.WithMaxIdleConnsPerHost(5),
    langfuse.WithDropOnQueueFull(true), // Don't buffer when full
)
```

---

## Security Considerations

### Credential Management

**Never hardcode credentials:**

```go
// BAD: Hardcoded credentials
client, err := langfuse.New("pk-xxx", "sk-xxx")

// GOOD: Environment variables
client, err := langfuse.New(
    os.Getenv("LANGFUSE_PUBLIC_KEY"),
    os.Getenv("LANGFUSE_SECRET_KEY"),
)

// GOOD: Secret manager
publicKey, _ := secretManager.Get("langfuse-public-key")
secretKey, _ := secretManager.Get("langfuse-secret-key")
client, err := langfuse.New(publicKey, secretKey)
```

### Credential Validation

The SDK validates credential format:

- Public keys must start with `pk-`
- Secret keys must start with `sk-`
- Both must be at least 8 characters

### Logging Safety

The SDK automatically masks credentials in logs:

```go
// Safe to log
slog.Info("client config", "config", client.Config().String())
// Output: Config{PublicKey: "pk-***", SecretKey: "sk-***", ...}
```

### Network Security

- All communication uses HTTPS
- TLS 1.2+ enforced
- Certificate validation enabled by default

For custom TLS configuration:

```go
transport := &http.Transport{
    TLSClientConfig: &tls.Config{
        MinVersion: tls.VersionTLS13,
    },
}
httpClient := &http.Client{Transport: transport}

client, err := langfuse.New(publicKey, secretKey,
    langfuse.WithHTTPClient(httpClient),
)
```

### Data Sensitivity

Consider what data you send to Langfuse:

- **Input/Output**: May contain PII - consider redaction
- **Metadata**: Avoid sensitive identifiers
- **User IDs**: Use anonymized or hashed IDs if needed

```go
// Example: Hashing user IDs
import "crypto/sha256"

func hashUserID(userID string) string {
    h := sha256.Sum256([]byte(userID + salt))
    return fmt.Sprintf("%x", h[:8])
}

trace, _ := client.NewTrace().
    UserID(hashUserID(realUserID)).
    Create(ctx)
```

---

## Checklist for Production Deployment

- [ ] Configure appropriate batch size and queue size for expected volume
- [ ] Enable circuit breaker for resilience
- [ ] Set up backpressure monitoring and alerting
- [ ] Implement metrics collection
- [ ] Configure structured logging
- [ ] Handle async errors with callbacks
- [ ] Implement graceful shutdown with signal handling
- [ ] Use environment variables or secret manager for credentials
- [ ] Review data sensitivity and redact PII if needed
- [ ] Test failure scenarios (network failures, API outages)
- [ ] Set appropriate timeouts for your SLAs
- [ ] Document your configuration choices

---

## Quick Reference: Default Values

| Setting | Default | Range |
|---------|---------|-------|
| BatchSize | 100 | 1-10000 |
| FlushInterval | 5s | >=100ms |
| BatchQueueSize | 100 | >=1 |
| Timeout | 30s | 0-10m |
| MaxRetries | 3 | 0-100 |
| RetryDelay | 1s | - |
| ShutdownTimeout | 35s | >=1s |
| MaxIdleConns | 100 | - |
| MaxIdleConnsPerHost | 10 | - |
| IdleConnTimeout | 90s | - |
