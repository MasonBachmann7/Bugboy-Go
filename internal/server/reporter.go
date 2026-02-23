package server

// ErrorReporter captures handled errors and panics for external systems.
type ErrorReporter interface {
	CaptureError(error)
	CapturePanic(any)
}

type noopReporter struct{}

func (noopReporter) CaptureError(error) {}
func (noopReporter) CapturePanic(any)   {}

func reporterOrNoop(reporter ErrorReporter) ErrorReporter {
	if reporter == nil {
		return noopReporter{}
	}
	return reporter
}
