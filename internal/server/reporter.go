package server

import "net/http"

// ErrorReporter captures handled errors and panics for external systems.
type ErrorReporter interface {
	CaptureError(error)
	CaptureErrorWithRequest(error, *http.Request)
	CapturePanic(any)
	CapturePanicWithRequest(any, *http.Request)
}

type noopReporter struct{}

func (noopReporter) CaptureError(error)                           {}
func (noopReporter) CaptureErrorWithRequest(error, *http.Request) {}
func (noopReporter) CapturePanic(any)                             {}
func (noopReporter) CapturePanicWithRequest(any, *http.Request)   {}

func reporterOrNoop(reporter ErrorReporter) ErrorReporter {
	if reporter == nil {
		return noopReporter{}
	}
	return reporter
}
