package langfusetest

import (
	"testing"
	"time"
)

func TestMockMetrics_Counter(t *testing.T) {
	m := NewMockMetrics()

	m.IncrementCounter("events", 1)
	m.IncrementCounter("events", 2)
	m.IncrementCounter("other", 5)

	if got := m.GetCounter("events"); got != 3 {
		t.Errorf("GetCounter(events) = %d, want 3", got)
	}
	if got := m.GetCounter("other"); got != 5 {
		t.Errorf("GetCounter(other) = %d, want 5", got)
	}
	if got := m.GetCounter("missing"); got != 0 {
		t.Errorf("GetCounter(missing) = %d, want 0", got)
	}
}

func TestMockMetrics_Gauge(t *testing.T) {
	m := NewMockMetrics()

	m.SetGauge("queue_size", 10.5)
	m.SetGauge("queue_size", 15.2) // Override

	if got := m.GetGauge("queue_size"); got != 15.2 {
		t.Errorf("GetGauge(queue_size) = %f, want 15.2", got)
	}
	if got := m.GetGauge("missing"); got != 0 {
		t.Errorf("GetGauge(missing) = %f, want 0", got)
	}
}

func TestMockMetrics_Timing(t *testing.T) {
	m := NewMockMetrics()

	// Record using time.Duration
	m.RecordDuration("flush", 100*time.Millisecond)
	m.RecordDuration("flush", 200*time.Millisecond)
	m.RecordDuration("request", 50*time.Millisecond)

	timings := m.GetTimings("flush")
	if len(timings) != 2 {
		t.Errorf("len(GetTimings(flush)) = %d, want 2", len(timings))
	}
	if timings[0] != 100000000 { // 100ms in nanoseconds
		t.Errorf("timing[0] = %d, want 100000000", timings[0])
	}

	requestTimings := m.GetTimings("request")
	if len(requestTimings) != 1 {
		t.Errorf("len(GetTimings(request)) = %d, want 1", len(requestTimings))
	}
	if requestTimings[0] != 50000000 { // 50ms in nanoseconds
		t.Errorf("request timing = %d, want 50000000", requestTimings[0])
	}

	missing := m.GetTimings("missing")
	if len(missing) != 0 {
		t.Errorf("len(GetTimings(missing)) = %d, want 0", len(missing))
	}
}

func TestMockMetrics_Reset(t *testing.T) {
	m := NewMockMetrics()

	m.IncrementCounter("events", 10)
	m.SetGauge("size", 5.5)
	m.RecordDuration("time", time.Millisecond)

	m.Reset()

	if got := m.GetCounter("events"); got != 0 {
		t.Error("Reset should clear counters")
	}
	if got := m.GetGauge("size"); got != 0 {
		t.Error("Reset should clear gauges")
	}
	if got := m.GetTimings("time"); len(got) != 0 {
		t.Error("Reset should clear timings")
	}
}

func TestMockMetrics_ReturnsCopy(t *testing.T) {
	m := NewMockMetrics()

	m.RecordDuration("test", time.Millisecond)
	m.RecordDuration("test", 2*time.Millisecond)

	timings := m.GetTimings("test")
	timings[0] = 999 // Modify returned slice

	// Original should be unchanged (1ms = 1000000ns)
	if m.GetTimings("test")[0] != 1000000 {
		t.Error("GetTimings should return a copy")
	}
}

func TestMockLogger_Messages(t *testing.T) {
	l := NewMockLogger()

	l.Printf("message 1")
	l.Printf("message 2 with %d args", 1)

	if l.MessageCount() != 2 {
		t.Errorf("MessageCount() = %d, want 2", l.MessageCount())
	}

	messages := l.GetMessages()
	if len(messages) != 2 {
		t.Errorf("len(GetMessages()) = %d, want 2", len(messages))
	}
	if messages[0] != "message 1" {
		t.Errorf("messages[0] = %q, want %q", messages[0], "message 1")
	}
}

func TestMockLogger_Reset(t *testing.T) {
	l := NewMockLogger()

	l.Printf("test")
	l.Printf("test")

	l.Reset()

	if l.MessageCount() != 0 {
		t.Error("Reset should clear all messages")
	}
}

func TestMockLogger_ReturnsCopy(t *testing.T) {
	l := NewMockLogger()

	l.Printf("original")

	messages := l.GetMessages()
	messages[0] = "modified"

	// Original should be unchanged
	if l.GetMessages()[0] != "original" {
		t.Error("GetMessages should return a copy")
	}
}
