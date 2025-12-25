package langfuse

import (
	"context"
	"fmt"
	"net/url"
)

// SessionsClient handles session-related API operations.
type SessionsClient struct {
	client *Client
}

// SessionsListParams represents parameters for listing sessions.
type SessionsListParams struct {
	PaginationParams
	FromTimestamp string
	ToTimestamp   string
}

// SessionsListResponse represents the response from listing sessions.
type SessionsListResponse struct {
	Data []Session    `json:"data"`
	Meta MetaResponse `json:"meta"`
}

// List retrieves a list of sessions.
func (c *SessionsClient) List(ctx context.Context, params *SessionsListParams) (*SessionsListResponse, error) {
	query := url.Values{}
	if params != nil {
		query = params.PaginationParams.ToQuery()
		if params.FromTimestamp != "" {
			query.Set("fromTimestamp", params.FromTimestamp)
		}
		if params.ToTimestamp != "" {
			query.Set("toTimestamp", params.ToTimestamp)
		}
	}

	var result SessionsListResponse
	err := c.client.http.get(ctx, "/sessions", query, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a session by ID.
func (c *SessionsClient) Get(ctx context.Context, sessionID string) (*Session, error) {
	var result Session
	err := c.client.http.get(ctx, fmt.Sprintf("/sessions/%s", sessionID), nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// SessionWithTraces represents a session with its traces.
type SessionWithTraces struct {
	Session
	Traces []Trace `json:"traces"`
}

// GetWithTraces retrieves a session with all its traces.
func (c *SessionsClient) GetWithTraces(ctx context.Context, sessionID string) (*SessionWithTraces, error) {
	// First get the session
	session, err := c.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Then get traces for this session
	tracesResp, err := c.client.Traces().List(ctx, &TracesListParams{
		FilterParams: FilterParams{
			SessionID: sessionID,
		},
	})
	if err != nil {
		return nil, err
	}

	return &SessionWithTraces{
		Session: *session,
		Traces:  tracesResp.Data,
	}, nil
}
