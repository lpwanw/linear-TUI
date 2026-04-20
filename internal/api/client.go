package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const DefaultEndpoint = "https://api.linear.app/graphql"

type Client struct {
	apiKey   string
	endpoint string
	http     *http.Client
}

func NewClient(apiKey string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		apiKey:   apiKey,
		endpoint: DefaultEndpoint,
		http:     httpClient,
	}
}

func (c *Client) WithEndpoint(url string) *Client {
	c.endpoint = url
	return c
}

type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphqlError struct {
	Message    string         `json:"message"`
	Extensions map[string]any `json:"extensions"`
	Path       []any          `json:"path"`
}

type graphqlEnvelope struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphqlError  `json:"errors"`
}

// Query executes a GraphQL operation and unmarshals the top-level `data` object into out.
func (c *Client) Query(ctx context.Context, query string, variables map[string]any, out any) error {
	body, err := json.Marshal(graphqlRequest{Query: query, Variables: variables})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return &NetworkError{Err: err}
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return &NetworkError{Err: err}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Linear typically returns 200, but surface any other status for visibility.
		return &APIError{Code: "HTTP_" + strconv.Itoa(resp.StatusCode), Message: string(raw)}
	}

	var env graphqlEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return &APIError{Message: fmt.Sprintf("decode envelope: %v", err)}
	}
	if len(env.Errors) > 0 {
		return convertGraphQLError(env.Errors[0], resp.Header)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return &APIError{Message: fmt.Sprintf("decode data: %v", err)}
	}
	return nil
}

func convertGraphQLError(e graphqlError, h http.Header) error {
	code, _ := e.Extensions["code"].(string)
	switch code {
	case "UNAUTHENTICATED":
		return ErrUnauthenticated
	case "RATE_LIMIT_EXCEEDED":
		reset := parseResetHeader(h)
		return &RateLimitError{ResetAt: reset}
	default:
		return &APIError{Code: code, Message: e.Message}
	}
}

func parseResetHeader(h http.Header) time.Time {
	v := h.Get("X-RateLimit-Reset")
	if v == "" {
		return time.Time{}
	}
	sec, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(sec, 0)
}
