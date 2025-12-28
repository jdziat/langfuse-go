package langfusetest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/jdziat/langfuse-go"
)

func TestMockServer_RecordsRequests(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	// Make a request
	resp, err := http.Post(ms.URL+"/api/public/ingestion", "application/json", bytes.NewReader([]byte(`{"test": "data"}`)))
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	resp.Body.Close()

	if ms.RequestCount() != 1 {
		t.Errorf("RequestCount() = %d, want 1", ms.RequestCount())
	}

	req := ms.LastRequest()
	if req == nil {
		t.Fatal("LastRequest() returned nil")
	}
	if req.Method != "POST" {
		t.Errorf("Method = %q, want %q", req.Method, "POST")
	}
	if req.Path != "/api/public/ingestion" {
		t.Errorf("Path = %q, want %q", req.Path, "/api/public/ingestion")
	}
	if req.ContentType != "application/json" {
		t.Errorf("ContentType = %q, want %q", req.ContentType, "application/json")
	}
}

func TestMockServer_Reset(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	// Make a request
	resp, err := http.Post(ms.URL+"/test", "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	resp.Body.Close()

	if ms.RequestCount() != 1 {
		t.Errorf("RequestCount() = %d before reset, want 1", ms.RequestCount())
	}

	ms.Reset()

	if ms.RequestCount() != 0 {
		t.Errorf("RequestCount() = %d after reset, want 0", ms.RequestCount())
	}
}

func TestMockServer_RequestAt(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	// Make multiple requests
	paths := []string{"/first", "/second", "/third"}
	for _, path := range paths {
		resp, err := http.Get(ms.URL + path)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		resp.Body.Close()
	}

	// Test RequestAt
	for i, path := range paths {
		req := ms.RequestAt(i)
		if req == nil {
			t.Errorf("RequestAt(%d) returned nil", i)
			continue
		}
		if req.Path != path {
			t.Errorf("RequestAt(%d).Path = %q, want %q", i, req.Path, path)
		}
	}

	// Test out of bounds
	if ms.RequestAt(-1) != nil {
		t.Error("RequestAt(-1) should return nil")
	}
	if ms.RequestAt(100) != nil {
		t.Error("RequestAt(100) should return nil")
	}
}

func TestMockServer_RespondWithSuccess(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()
	ms.RespondWithSuccess()

	resp, err := http.Get(ms.URL + "/test")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result langfuse.IngestionResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result.Successes) != 1 {
		t.Errorf("Successes count = %d, want 1", len(result.Successes))
	}
}

func TestMockServer_RespondWithError(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()
	ms.RespondWithError(http.StatusBadRequest, "Invalid request")

	resp, err := http.Get(ms.URL + "/test")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte("Invalid request")) {
		t.Errorf("Response body should contain error message")
	}
}

func TestMockServer_RespondWithRateLimit(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()
	ms.RespondWithRateLimit(60)

	resp, err := http.Get(ms.URL + "/test")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusTooManyRequests)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["retry_after"] != float64(60) {
		t.Errorf("retry_after = %v, want 60", result["retry_after"])
	}
}

func TestMockServer_RespondWithUnauthorized(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()
	ms.RespondWithUnauthorized()

	resp, err := http.Get(ms.URL + "/test")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestMockServer_RespondWithServerError(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()
	ms.RespondWithServerError()

	resp, err := http.Get(ms.URL + "/test")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

func TestMockServer_RespondWithPartialSuccess(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()
	ms.RespondWithPartialSuccess([]string{"id1", "id2"}, []string{"id3"})

	resp, err := http.Get(ms.URL + "/test")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result langfuse.IngestionResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result.Successes) != 2 {
		t.Errorf("Successes count = %d, want 2", len(result.Successes))
	}
	if len(result.Errors) != 1 {
		t.Errorf("Errors count = %d, want 1", len(result.Errors))
	}
}

func TestMockServer_RespondWith(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	customBody := map[string]string{"custom": "response"}
	ms.RespondWith(http.StatusAccepted, customBody)

	resp, err := http.Get(ms.URL + "/test")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusAccepted)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["custom"] != "response" {
		t.Errorf("Response body = %v, want custom response", result)
	}
}

func TestMockServer_HasRequestWithPath(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	// Make requests
	resp1, _ := http.Get(ms.URL + "/api/ingestion")
	resp1.Body.Close()
	resp2, _ := http.Get(ms.URL + "/api/prompts")
	resp2.Body.Close()

	if !ms.HasRequestWithPath("/api/ingestion") {
		t.Error("HasRequestWithPath should return true for /api/ingestion")
	}
	if !ms.HasRequestWithPath("/api/prompts") {
		t.Error("HasRequestWithPath should return true for /api/prompts")
	}
	if ms.HasRequestWithPath("/api/missing") {
		t.Error("HasRequestWithPath should return false for /api/missing")
	}
}

func TestMockServer_RequestsWithPath(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	// Make multiple requests to same path
	for i := 0; i < 3; i++ {
		resp, _ := http.Get(ms.URL + "/api/ingestion")
		resp.Body.Close()
	}
	resp, _ := http.Get(ms.URL + "/api/other")
	resp.Body.Close()

	reqs := ms.RequestsWithPath("/api/ingestion")
	if len(reqs) != 3 {
		t.Errorf("RequestsWithPath returned %d requests, want 3", len(reqs))
	}

	other := ms.RequestsWithPath("/api/other")
	if len(other) != 1 {
		t.Errorf("RequestsWithPath returned %d requests for /api/other, want 1", len(other))
	}

	missing := ms.RequestsWithPath("/api/missing")
	if len(missing) != 0 {
		t.Errorf("RequestsWithPath returned %d requests for missing path, want 0", len(missing))
	}
}

func TestMockServer_Requests(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	// Make requests
	resp1, _ := http.Get(ms.URL + "/first")
	resp1.Body.Close()
	resp2, _ := http.Get(ms.URL + "/second")
	resp2.Body.Close()

	reqs := ms.Requests()
	if len(reqs) != 2 {
		t.Errorf("Requests() returned %d requests, want 2", len(reqs))
	}

	// Verify it returns a copy (modifying returned slice doesn't affect internal state)
	reqs[0] = nil
	if ms.RequestAt(0) == nil {
		t.Error("Modifying returned slice should not affect internal state")
	}
}

func TestMockServer_SetResponseFunc(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	callCount := 0
	ms.SetResponseFunc(func(r *http.Request) (int, any) {
		callCount++
		return http.StatusTeapot, map[string]int{"count": callCount}
	})

	// First request
	resp1, err := http.Get(ms.URL + "/test")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp1.Body.Close()

	if resp1.StatusCode != http.StatusTeapot {
		t.Errorf("StatusCode = %d, want %d", resp1.StatusCode, http.StatusTeapot)
	}

	var result1 map[string]int
	json.NewDecoder(resp1.Body).Decode(&result1)
	if result1["count"] != 1 {
		t.Errorf("count = %d, want 1", result1["count"])
	}

	// Second request
	resp2, err := http.Get(ms.URL + "/test")
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	defer resp2.Body.Close()

	var result2 map[string]int
	json.NewDecoder(resp2.Body).Decode(&result2)
	if result2["count"] != 2 {
		t.Errorf("count = %d, want 2", result2["count"])
	}
}

func TestMockServer_DefaultResponse(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	// Default response should be success
	resp, err := http.Get(ms.URL + "/test")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result langfuse.IngestionResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result.Successes) == 0 {
		t.Error("Default response should have successes")
	}
}
