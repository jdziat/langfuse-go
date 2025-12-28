# Troubleshooting Guide

This guide helps diagnose and resolve common issues with the Langfuse Go SDK.

## Table of Contents

1. [Events Not Appearing in Langfuse](#events-not-appearing-in-langfuse)
2. [High Memory Usage](#high-memory-usage)
3. [Circuit Breaker Issues](#circuit-breaker-issues)
4. [Connection Problems](#connection-problems)
5. [Performance Issues](#performance-issues)
6. [Debug Logging](#debug-logging)
7. [Error Diagnosis](#error-diagnosis)
8. [Common Error Messages](#common-error-messages)

---

## Events Not Appearing in Langfuse

### Symptom
Traces, spans, or generations are created but don't appear in the Langfuse dashboard.

### Diagnostic Steps

#### 1. Check if Shutdown is Called

The most common cause is not calling `Shutdown()` or `Flush()`:

```go
// PROBLEM: Events may not be sent
client, _ := langfuse.New(pk, sk)
trace, _ := client.NewTrace().Name("test").Create(ctx)
// Program exits - events lost!

// SOLUTION: Always defer shutdown
client, _ := langfuse.New(pk, sk)
defer client.Shutdown(context.Background())
trace, _ := client.NewTrace().Name("test").Create(ctx)
```

#### 2. Enable Debug Logging

```go
client, err := langfuse.New(pk, sk,
    langfuse.WithDebug(true),
)
```

Look for:
- `sending batch` - confirms events are being sent
- `batch sent successfully` - confirms delivery
- Error messages indicating failures

#### 3. Check Async Errors

```go
client, err := langfuse.New(pk, sk,
    langfuse.WithOnAsyncError(func(err *langfuse.AsyncError) {
        log.Printf("Async error [%s]: %v", err.Operation, err.Err)
    }),
)
```

#### 4. Monitor Batch Results

```go
client, err := langfuse.New(pk, sk,
    langfuse.WithOnBatchFlushed(func(result langfuse.BatchResult) {
        log.Printf("Batch: events=%d success=%t errors=%d",
            result.EventCount, result.Success, result.Errors)
    }),
)
```

#### 5. Verify Credentials

```go
// Check credential format
pk := os.Getenv("LANGFUSE_PUBLIC_KEY")
sk := os.Getenv("LANGFUSE_SECRET_KEY")

if !strings.HasPrefix(pk, "pk-") {
    log.Fatal("Invalid public key format")
}
if !strings.HasPrefix(sk, "sk-") {
    log.Fatal("Invalid secret key format")
}
```

#### 6. Test API Connectivity

```go
// Health check
health, err := client.Health(ctx)
if err != nil {
    log.Printf("API health check failed: %v", err)
}
if health.Status != "OK" {
    log.Printf("API reports unhealthy: %s", health.Status)
}
```

### Common Causes

| Cause | Solution |
|-------|----------|
| Missing `Shutdown()` | Add `defer client.Shutdown(ctx)` |
| Wrong credentials | Verify pk/sk from Langfuse settings |
| Wrong region | Use `WithRegion(langfuse.RegionUS)` if in US |
| Network blocked | Check firewall rules for `cloud.langfuse.com` |
| Events dropped | Enable backpressure monitoring |

---

## High Memory Usage

### Symptom
Application memory grows over time when using the SDK.

### Diagnostic Steps

#### 1. Check Queue Size

```go
// Get current backpressure status
status := client.BackpressureStatus()
log.Printf("Queue size: %d/%d (%.1f%%)",
    status.MonitorStats.CurrentSize,
    status.MonitorStats.Capacity,
    status.MonitorStats.PercentFull)
```

#### 2. Monitor Queue State

```go
client, err := langfuse.New(pk, sk,
    langfuse.WithOnBackpressure(func(state langfuse.QueueState) {
        log.Printf("Queue: %d/%d (%.1f%%) level=%s",
            state.Size, state.Capacity, state.PercentFull, state.Level)
    }),
)
```

#### 3. Check for Goroutine Leaks

Enable idle warnings to detect missing shutdown:

```go
client, err := langfuse.New(pk, sk,
    langfuse.WithIdleWarning(5*time.Minute),
)
```

### Solutions

| Cause | Solution |
|-------|----------|
| Queue backing up | Increase `BatchQueueSize` or reduce event volume |
| Events too large | Reduce metadata size, limit input/output |
| Many connections | Tune `MaxIdleConns` settings |
| Goroutine leak | Ensure `Shutdown()` is called |

**Reduce Memory Usage:**

```go
client, err := langfuse.New(pk, sk,
    langfuse.WithBatchSize(50),        // Smaller batches
    langfuse.WithBatchQueueSize(50),   // Smaller queue
    langfuse.WithMaxIdleConns(20),     // Fewer connections
    langfuse.WithDropOnQueueFull(true), // Don't buffer infinitely
)
```

---

## Circuit Breaker Issues

### Symptom
Requests fail with `ErrCircuitOpen` or circuit breaker opens unexpectedly.

### Diagnostic Steps

#### 1. Monitor State Changes

```go
client, err := langfuse.New(pk, sk,
    langfuse.WithCircuitBreaker(langfuse.CircuitBreakerConfig{
        FailureThreshold: 5,
        Timeout:          30 * time.Second,
        OnStateChange: func(from, to langfuse.CircuitState) {
            log.Printf("Circuit: %s -> %s", from, to)
        },
    }),
)
```

#### 2. Check Circuit Status

```go
state := client.CircuitBreakerState()
log.Printf("Circuit state: %s", state.String())
```

#### 3. Understand the Cause

Common triggers for circuit opening:
- Network connectivity issues
- Langfuse API outages
- Rate limiting (429 responses)
- Server errors (5xx responses)

### Solutions

| Issue | Solution |
|-------|----------|
| Premature opening | Increase `FailureThreshold` |
| Slow recovery | Decrease `Timeout` duration |
| Flapping | Increase `SuccessThreshold` |
| Testing interference | Disable for tests |

**Adjust Circuit Breaker Settings:**

```go
client, err := langfuse.New(pk, sk,
    langfuse.WithCircuitBreaker(langfuse.CircuitBreakerConfig{
        FailureThreshold:    10,  // More tolerant of failures
        SuccessThreshold:    3,   // Need more successes to close
        Timeout:             60 * time.Second, // Longer recovery window
        HalfOpenMaxRequests: 5,   // More test requests
    }),
)
```

**Disable for Testing:**

```go
// Don't use circuit breaker in tests
client, err := langfuse.New(pk, sk)
// No WithCircuitBreaker or WithDefaultCircuitBreaker
```

---

## Connection Problems

### Symptom
Requests fail with connection errors, timeouts, or TLS errors.

### Diagnostic Steps

#### 1. Check Network Connectivity

```bash
# Test DNS resolution
nslookup cloud.langfuse.com

# Test HTTPS connectivity
curl -I https://cloud.langfuse.com/api/public/health

# Test with timeout
curl --connect-timeout 5 https://cloud.langfuse.com/api/public/health
```

#### 2. Enable Request Logging

```go
client, err := langfuse.New(pk, sk,
    langfuse.WithHTTPHooks(
        langfuse.LoggingHook(log.Default()),
    ),
)
```

#### 3. Check for Proxy Issues

```go
// Custom transport with proxy
transport := &http.Transport{
    Proxy: http.ProxyFromEnvironment,
}
httpClient := &http.Client{Transport: transport}

client, err := langfuse.New(pk, sk,
    langfuse.WithHTTPClient(httpClient),
)
```

### Solutions

| Error | Cause | Solution |
|-------|-------|----------|
| `dial tcp: lookup failed` | DNS issue | Check DNS resolver |
| `connection refused` | Port blocked | Check firewall for 443 |
| `TLS handshake timeout` | Network latency | Increase timeout |
| `certificate verify failed` | Corporate proxy | Configure custom CA |

**Configure for Corporate Proxy:**

```go
import "crypto/x509"

// Load custom CA
caCert, _ := os.ReadFile("/path/to/corporate-ca.crt")
caCertPool := x509.NewCertPool()
caCertPool.AppendCertsFromPEM(caCert)

transport := &http.Transport{
    Proxy: http.ProxyFromEnvironment,
    TLSClientConfig: &tls.Config{
        RootCAs: caCertPool,
    },
}

client, err := langfuse.New(pk, sk,
    langfuse.WithHTTPClient(&http.Client{Transport: transport}),
)
```

**Increase Timeouts:**

```go
client, err := langfuse.New(pk, sk,
    langfuse.WithTimeout(60*time.Second),
)
```

---

## Performance Issues

### Symptom
High latency, slow response times, or application slowdown.

### Diagnostic Steps

#### 1. Check if SDK is Blocking

The SDK should be non-blocking. If calls are slow:

```go
start := time.Now()
trace, err := client.NewTrace().Name("test").Create(ctx)
elapsed := time.Since(start)

if elapsed > 10*time.Millisecond {
    log.Printf("Trace creation took %v (expected < 10ms)", elapsed)
}
```

#### 2. Monitor Batch Send Times

```go
client, err := langfuse.New(pk, sk,
    langfuse.WithOnBatchFlushed(func(result langfuse.BatchResult) {
        if result.Duration > 5*time.Second {
            log.Printf("Slow batch: %v for %d events",
                result.Duration, result.EventCount)
        }
    }),
)
```

#### 3. Check for Queue Blocking

If `WithBlockOnQueueFull(true)` is set, calls may block:

```go
// This can block if queue is full
trace, err := client.NewTrace().Name("test").Create(ctx)

// Use non-blocking mode instead
client, err := langfuse.New(pk, sk,
    langfuse.WithDropOnQueueFull(true), // Drop instead of block
)
```

### Solutions

| Issue | Solution |
|-------|----------|
| Slow event creation | Should be < 1ms, check for queue full |
| Slow batches | Check network, reduce batch size |
| Queue blocking | Use `WithDropOnQueueFull(true)` |
| Too many retries | Reduce `MaxRetries` |

**Optimize for Low Latency:**

```go
client, err := langfuse.New(pk, sk,
    langfuse.WithBatchSize(50),          // Smaller batches
    langfuse.WithFlushInterval(2*time.Second), // More frequent flushes
    langfuse.WithDropOnQueueFull(true),  // Never block
    langfuse.WithMaxRetries(2),          // Fewer retries
)
```

---

## Debug Logging

### Basic Debug Mode

```go
client, err := langfuse.New(pk, sk,
    langfuse.WithDebug(true),
)
```

This logs to stderr with the `langfuse:` prefix.

### Custom Logger

```go
// Printf-style logger
myLogger := log.New(os.Stdout, "[LANGFUSE] ", log.LstdFlags|log.Lmsgprefix)

client, err := langfuse.New(pk, sk,
    langfuse.WithLogger(langfuse.WrapStdLogger(myLogger)),
)
```

### Structured Logging (slog)

```go
import "log/slog"

handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
})
logger := slog.New(handler)

client, err := langfuse.New(pk, sk,
    langfuse.WithStructuredLogger(langfuse.NewSlogAdapter(logger)),
)
```

### HTTP Request/Response Logging

```go
client, err := langfuse.New(pk, sk,
    langfuse.WithHTTPHooks(
        langfuse.LoggingHook(log.Default()),
    ),
)
```

### Log Levels

The SDK logs at these levels:

| Level | Content |
|-------|---------|
| Debug | Detailed internal operations |
| Info | Normal operations (batch sends, state changes) |
| Warn | Retries, backpressure warnings |
| Error | Failures, drops, circuit breaker triggers |

---

## Error Diagnosis

### Error Types

```go
import "errors"

trace, err := client.NewTrace().Name("test").Create(ctx)
if err != nil {
    // Check specific error types
    if errors.Is(err, langfuse.ErrClientClosed) {
        log.Println("Client was shut down")
    }
    if errors.Is(err, langfuse.ErrCircuitOpen) {
        log.Println("Circuit breaker is open")
    }

    // Check for API errors
    var apiErr *langfuse.APIError
    if errors.As(err, &apiErr) {
        log.Printf("API error: %d %s", apiErr.StatusCode, apiErr.Message)

        if apiErr.IsRateLimited() {
            log.Printf("Rate limited, retry after %v", apiErr.RetryAfter)
        }
    }

    // Check for validation errors
    var valErr *langfuse.ValidationError
    if errors.As(err, &valErr) {
        log.Printf("Validation error on field %q: %s", valErr.Field, valErr.Message)
    }
}
```

### Async Error Handling

```go
client, err := langfuse.New(pk, sk,
    langfuse.WithOnAsyncError(func(err *langfuse.AsyncError) {
        log.Printf("Async error:")
        log.Printf("  Operation: %s", err.Operation)
        log.Printf("  Error: %v", err.Err)
        log.Printf("  Retryable: %t", err.Retryable)
        log.Printf("  Time: %v", err.Time)
        if len(err.EventIDs) > 0 {
            log.Printf("  Affected events: %v", err.EventIDs)
        }
    }),
)
```

### Error Recovery Patterns

```go
// Retry with backoff for transient errors
func createTraceWithRetry(ctx context.Context, client *langfuse.Client) (*langfuse.TraceContext, error) {
    var lastErr error
    for attempt := 0; attempt < 3; attempt++ {
        trace, err := client.NewTrace().Name("test").Create(ctx)
        if err == nil {
            return trace, nil
        }

        lastErr = err

        // Don't retry validation or client errors
        var valErr *langfuse.ValidationError
        if errors.As(err, &valErr) {
            return nil, err
        }
        if errors.Is(err, langfuse.ErrClientClosed) {
            return nil, err
        }

        // Backoff before retry
        time.Sleep(time.Duration(attempt+1) * time.Second)
    }
    return nil, lastErr
}
```

---

## Common Error Messages

### Configuration Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `public key is too short` | Key less than 8 chars | Check LANGFUSE_PUBLIC_KEY |
| `public key should start with "pk-"` | Wrong key format | Verify key from Langfuse |
| `secret key should start with "sk-"` | Wrong key format | Verify key from Langfuse |
| `invalid base URL` | Malformed URL | Check WithBaseURL value |
| `flush interval must be at least 100ms` | Interval too short | Increase flush interval |

### Runtime Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `client is closed` | Used after Shutdown | Create new client |
| `circuit breaker is open` | API failures | Wait for recovery or check network |
| `context deadline exceeded` | Request timeout | Increase timeout or check network |
| `connection refused` | Network issue | Check firewall, DNS |
| `x509: certificate signed by unknown authority` | TLS issue | Configure CA or check proxy |

### API Errors

| Status | Meaning | Solution |
|--------|---------|----------|
| 400 | Bad request | Check event data format |
| 401 | Unauthorized | Verify credentials |
| 403 | Forbidden | Check project permissions |
| 404 | Not found | Verify base URL |
| 429 | Rate limited | Reduce request rate, check RetryAfter |
| 500 | Server error | Retry automatically |
| 502/503/504 | Gateway errors | Retry automatically |

---

## Getting Help

If you're still experiencing issues:

1. **Enable debug logging** and capture the output
2. **Check the circuit breaker status** for API health
3. **Review async errors** for background failures
4. **Test API connectivity** with curl
5. **Check Langfuse status page** for outages

When reporting issues, include:

- Go version (`go version`)
- SDK version (from `go.mod`)
- Debug logs (sanitize credentials)
- Error messages
- Minimal reproduction code

File issues at: https://github.com/jdziat/langfuse-go/issues
