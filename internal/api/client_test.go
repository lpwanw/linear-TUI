package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newFixtureServer(t *testing.T, body string, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Error("missing Authorization header")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

func TestQueryHappyPath(t *testing.T) {
	srv := newFixtureServer(t, `{"data":{"viewer":{"id":"u1","name":"Tay","email":"t@x"}}}`, 200)
	defer srv.Close()
	c := NewClient("lin_api_xxx", nil).WithEndpoint(srv.URL)

	var out struct {
		Viewer struct{ ID, Name, Email string }
	}
	if err := c.Query(context.Background(), QueryViewerAndTeams, nil, &out); err != nil {
		t.Fatal(err)
	}
	if out.Viewer.ID != "u1" || out.Viewer.Name != "Tay" {
		t.Fatalf("viewer = %+v", out.Viewer)
	}
}

func TestQueryUnauthenticated(t *testing.T) {
	srv := newFixtureServer(t, `{"errors":[{"message":"bad token","extensions":{"code":"UNAUTHENTICATED"}}]}`, 200)
	defer srv.Close()
	c := NewClient("lin_api_xxx", nil).WithEndpoint(srv.URL)
	err := c.Query(context.Background(), "query{}", nil, nil)
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("want ErrUnauthenticated, got %v", err)
	}
}

func TestQueryRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Reset", "1700000000")
		_, _ = w.Write([]byte(`{"errors":[{"message":"slow down","extensions":{"code":"RATE_LIMIT_EXCEEDED"}}]}`))
	}))
	defer srv.Close()
	c := NewClient("lin_api_xxx", nil).WithEndpoint(srv.URL)
	err := c.Query(context.Background(), "query{}", nil, nil)
	var rle *RateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("want RateLimitError, got %v", err)
	}
	if rle.ResetAt.IsZero() {
		t.Error("expected non-zero reset time")
	}
}

func TestQueryGenericError(t *testing.T) {
	srv := newFixtureServer(t, `{"errors":[{"message":"bad input","extensions":{"code":"BAD_REQUEST"}}]}`, 200)
	defer srv.Close()
	c := NewClient("lin_api_xxx", nil).WithEndpoint(srv.URL)
	err := c.Query(context.Background(), "query{}", nil, nil)
	var ae *APIError
	if !errors.As(err, &ae) || ae.Code != "BAD_REQUEST" {
		t.Fatalf("want APIError BAD_REQUEST, got %v", err)
	}
}

func TestQueryMalformedJSON(t *testing.T) {
	srv := newFixtureServer(t, `not json`, 200)
	defer srv.Close()
	c := NewClient("lin_api_xxx", nil).WithEndpoint(srv.URL)
	err := c.Query(context.Background(), "query{}", nil, nil)
	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatalf("want APIError, got %v", err)
	}
}

func TestQueryVariablesMarshal(t *testing.T) {
	var got graphqlRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	defer srv.Close()
	c := NewClient("lin_api_xxx", nil).WithEndpoint(srv.URL)
	_ = c.Query(context.Background(), "query X($n: Int!){}", map[string]any{"n": 3}, nil)
	if got.Variables["n"].(float64) != 3 {
		t.Fatalf("variables = %+v", got.Variables)
	}
}
