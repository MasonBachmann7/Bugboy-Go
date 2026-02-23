package server

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

//go:embed templates/*.html
var templateFS embed.FS

type route struct {
	Method      string
	Path        string
	Description string
}

type pageData struct {
	Routes []route
}

type errorResponse struct {
	Error     string            `json:"error"`
	RequestID string            `json:"request_id"`
	Timestamp string            `json:"timestamp"`
	Details   map[string]string `json:"details,omitempty"`
}

type successResponse struct {
	Status    string            `json:"status"`
	RequestID string            `json:"request_id"`
	Timestamp string            `json:"timestamp"`
	Details   map[string]string `json:"details,omitempty"`
}

var bugRoutes = []route{
	{Method: http.MethodGet, Path: "/bugs/panic/nil-pointer", Description: "Nil pointer dereference panic"},
	{Method: http.MethodGet, Path: "/bugs/panic/index-out-of-range", Description: "Slice bounds panic"},
	{Method: http.MethodGet, Path: "/bugs/panic/divide-by-zero", Description: "Integer divide-by-zero panic"},
	{Method: http.MethodGet, Path: "/bugs/panic/nil-map-write", Description: "Assignment to entry in nil map panic"},
	{Method: http.MethodGet, Path: "/bugs/error/db-timeout", Description: "Handled timeout wrapped like a DB failure"},
	{Method: http.MethodGet, Path: "/bugs/error/external-api", Description: "Handled upstream API connectivity error"},
	{Method: http.MethodGet, Path: "/bugs/error/json-parse", Description: "Handled JSON parse/type mismatch error"},
	{Method: http.MethodGet, Path: "/bugs/background/panic", Description: "Async goroutine panic (captured and logged)"},
	{Method: http.MethodGet, Path: "/bugs/fatal/unhandled-goroutine-panic", Description: "Unrecovered goroutine panic that terminates the process"},
}

func NewHandler(logger *log.Logger) http.Handler {
	if logger == nil {
		logger = log.Default()
	}
	reporter := reporterOrNoop(nil)

	return newHandler(logger, reporter)
}

func NewHandlerWithReporter(logger *log.Logger, reporter ErrorReporter) http.Handler {
	if logger == nil {
		logger = log.Default()
	}
	reporter = reporterOrNoop(reporter)

	return newHandler(logger, reporter)
}

func newHandler(logger *log.Logger, reporter ErrorReporter) http.Handler {
	recoverPanics := true
	if raw := strings.TrimSpace(strings.ToLower(os.Getenv("BUGBOY_RECOVER_PANICS"))); raw == "false" || raw == "0" {
		recoverPanics = false
	}

	indexTemplate := template.Must(template.ParseFS(templateFS, "templates/index.html"))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := indexTemplate.Execute(w, pageData{Routes: bugRoutes}); err != nil {
			reporter.CaptureErrorWithRequest(err, r)
			logger.Printf("template render failed request_id=%s err=%v", requestIDFromContext(r.Context()), err)
		}
	})

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, successResponse{
			Status:    "ok",
			RequestID: requestIDFromContext(r.Context()),
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		})
	})

	mux.HandleFunc("GET /bugs/panic/nil-pointer", func(w http.ResponseWriter, r *http.Request) {
		var profile *struct{ Name string }
		_ = profile.Name
	})

	mux.HandleFunc("GET /bugs/panic/index-out-of-range", func(w http.ResponseWriter, r *http.Request) {
		values := []string{"alpha", "beta", "gamma"}
		_ = values[9]
	})

	mux.HandleFunc("GET /bugs/panic/divide-by-zero", func(w http.ResponseWriter, r *http.Request) {
		numerator := 42
		denominator := 0
		_ = numerator / denominator
	})

	mux.HandleFunc("GET /bugs/panic/nil-map-write", func(w http.ResponseWriter, r *http.Request) {
		var session map[string]string
		session["tenant"] = "demo"
	})

	mux.HandleFunc("GET /bugs/error/db-timeout", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 40*time.Millisecond)
		defer cancel()

		err := simulateDBQuery(ctx, 120*time.Millisecond)
		if err != nil {
			wrappedErr := fmt.Errorf("fetching release metadata failed: %w", err)
			reporter.CaptureErrorWithRequest(wrappedErr, r)
			logger.Printf("application error request_id=%s route=%s err=%v", requestIDFromContext(r.Context()), r.URL.Path, wrappedErr)
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Error:     "database operation failed",
				RequestID: requestIDFromContext(r.Context()),
				Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
				Details: map[string]string{
					"operation":  "GetLatestRelease",
					"root_cause": wrappedErr.Error(),
				},
			})
			return
		}

		writeJSON(w, http.StatusOK, successResponse{
			Status:    "unexpected-success",
			RequestID: requestIDFromContext(r.Context()),
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		})
	})

	mux.HandleFunc("GET /bugs/error/external-api", func(w http.ResponseWriter, r *http.Request) {
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, "http://127.0.0.1:1/internal-api", nil)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Error:     "failed to build upstream request",
				RequestID: requestIDFromContext(r.Context()),
				Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
				Details:   map[string]string{"root_cause": err.Error()},
			})
			return
		}

		client := &http.Client{Timeout: 200 * time.Millisecond}
		_, err = client.Do(req)
		if err != nil {
			reporter.CaptureErrorWithRequest(err, r)
			writeJSON(w, http.StatusBadGateway, errorResponse{
				Error:     "upstream dependency unavailable",
				RequestID: requestIDFromContext(r.Context()),
				Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
				Details: map[string]string{
					"dependency": "internal-api",
					"root_cause": err.Error(),
				},
			})
			return
		}

		writeJSON(w, http.StatusOK, successResponse{
			Status:    "unexpected-success",
			RequestID: requestIDFromContext(r.Context()),
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		})
	})

	mux.HandleFunc("GET /bugs/error/json-parse", func(w http.ResponseWriter, r *http.Request) {
		type payload struct {
			UserID int    `json:"user_id"`
			Email  string `json:"email"`
		}

		raw := []byte(`{"user_id":"not-an-int","email":"demo@example.com"}`)
		var p payload
		err := json.Unmarshal(raw, &p)
		if err != nil {
			reporter.CaptureErrorWithRequest(err, r)
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error:     "request payload invalid",
				RequestID: requestIDFromContext(r.Context()),
				Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
				Details: map[string]string{
					"root_cause": err.Error(),
				},
			})
			return
		}

		writeJSON(w, http.StatusOK, successResponse{
			Status:    "unexpected-success",
			RequestID: requestIDFromContext(r.Context()),
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		})
	})

	mux.HandleFunc("GET /bugs/background/panic", func(w http.ResponseWriter, r *http.Request) {
		reqID := requestIDFromContext(r.Context())
		route := r.URL.Path
		method := r.Method
		go func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					// Build a minimal request for route context (original r may be stale)
					fakeReq, _ := http.NewRequest(method, route, nil)
					reporter.CapturePanicWithRequest(recovered, fakeReq)
					logger.Printf(
						"background panic recovered request_id=%s panic=%v stack=%s",
						reqID,
						recovered,
						string(debug.Stack()),
					)
				}
			}()

			time.Sleep(25 * time.Millisecond)
			var workers []string
			_ = workers[3]
		}()

		writeJSON(w, http.StatusAccepted, successResponse{
			Status:    "queued",
			RequestID: requestIDFromContext(r.Context()),
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Details: map[string]string{
				"note": "background panic will be logged after response",
			},
		})
	})

	mux.HandleFunc("GET /bugs/fatal/unhandled-goroutine-panic", func(w http.ResponseWriter, r *http.Request) {
		go func() {
			time.Sleep(25 * time.Millisecond)
			var workers []string
			_ = workers[3]
		}()

		writeJSON(w, http.StatusAccepted, successResponse{
			Status:    "queued",
			RequestID: requestIDFromContext(r.Context()),
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Details: map[string]string{
				"note": "process should terminate once goroutine panic triggers",
			},
		})
	})

	var handler http.Handler = mux
	if recoverPanics {
		handler = recoverPanicMiddleware(logger, reporter, handler)
	}
	handler = accessLogMiddleware(logger, handler)
	handler = requestIDMiddleware(handler)
	return handler
}

func simulateDBQuery(ctx context.Context, latency time.Duration) error {
	select {
	case <-time.After(latency):
		return nil
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("db timeout after %s: %w", latency, ctx.Err())
		}
		return fmt.Errorf("db query canceled: %w", ctx.Err())
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, `{"error":"response encoding failed"}`, http.StatusInternalServerError)
	}
}
