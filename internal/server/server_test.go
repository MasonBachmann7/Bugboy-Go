package server

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(NewHandler(log.New(io.Discard, "", 0)))
	defer server.Close()

	resp, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, resp.StatusCode)
	}

	if got := resp.Header.Get("X-Request-ID"); got == "" {
		t.Fatalf("expected X-Request-ID header")
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed decoding health payload: %v", err)
	}

	if payload["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", payload["status"])
	}
}

func TestPanicRouteIsRecovered(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(NewHandler(log.New(io.Discard, "", 0)))
	defer server.Close()

	resp, err := http.Get(server.URL + "/bugs/panic/nil-pointer")
	if err != nil {
		t.Fatalf("panic route request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, resp.StatusCode)
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed decoding panic payload: %v", err)
	}

	if payload["error"] == "" {
		t.Fatalf("expected error payload, got %#v", payload)
	}
}

func TestHandledErrorRoutes(t *testing.T) {
	server := httptest.NewServer(NewHandler(log.New(io.Discard, "", 0)))
	defer server.Close()

	tests := []struct {
		name       string
		path       string
		statusCode int
	}{
		{name: "db timeout", path: "/bugs/error/db-timeout", statusCode: http.StatusInternalServerError},
		{name: "external api", path: "/bugs/error/external-api", statusCode: http.StatusBadGateway},
		{name: "json parse", path: "/bugs/error/json-parse", statusCode: http.StatusBadRequest},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			resp, err := http.Get(server.URL + test.path)
			if err != nil {
				t.Fatalf("request failed for %s: %v", test.path, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != test.statusCode {
				t.Fatalf("expected %d, got %d for %s", test.statusCode, resp.StatusCode, test.path)
			}
		})
	}
}
