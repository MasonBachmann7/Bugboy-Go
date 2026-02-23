package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"sync/atomic"
	"time"
)

type contextKey string

const requestIDKey contextKey = "request_id"

var requestCounter uint64

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := fmt.Sprintf("req-%d", atomic.AddUint64(&requestCounter, 1))
		w.Header().Set("X-Request-ID", requestID)

		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func accessLogMiddleware(logger *log.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Printf(
			"request_id=%s method=%s path=%s remote_addr=%s duration_ms=%d",
			requestIDFromContext(r.Context()),
			r.Method,
			r.URL.Path,
			r.RemoteAddr,
			time.Since(start).Milliseconds(),
		)
	})
}

func recoverPanicMiddleware(logger *log.Logger, reporter ErrorReporter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				reporter.CapturePanic(recovered)
				requestID := requestIDFromContext(r.Context())
				logger.Printf(
					"panic recovered request_id=%s method=%s path=%s panic=%v stack=%s",
					requestID,
					r.Method,
					r.URL.Path,
					recovered,
					string(debug.Stack()),
				)
				writeJSON(w, http.StatusInternalServerError, errorResponse{
					Error:     "panic recovered in request handler",
					RequestID: requestID,
					Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
					Details: map[string]string{
						"panic": fmt.Sprint(recovered),
					},
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func requestIDFromContext(ctx context.Context) string {
	value, ok := ctx.Value(requestIDKey).(string)
	if !ok || value == "" {
		return "unknown"
	}
	return value
}
