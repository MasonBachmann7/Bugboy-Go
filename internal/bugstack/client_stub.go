//go:build !bugstack

package bugstack

import (
	"log"
	"net/http"
	"os"
	"strings"
)

// Client is a no-op reporter when the real SDK is not built in.
type Client struct{}

// InitFromEnv configures a no-op client in default builds.
func InitFromEnv(logger *log.Logger) *Client {
	if logger == nil {
		logger = log.Default()
	}
	if strings.TrimSpace(os.Getenv("BUGSTACK_API_KEY")) != "" {
		logger.Printf("BUGSTACK_API_KEY is set but bugstack SDK is disabled in this build; run with -tags bugstack after `go get github.com/MasonBachmann7/bugstack-go`")
	}
	return &Client{}
}

func (c *Client) CaptureError(err error) {}

func (c *Client) CaptureErrorWithRequest(err error, r *http.Request) {}

func (c *Client) CapturePanic(value any) {}

func (c *Client) CapturePanicWithRequest(value any, r *http.Request) {}

func (c *Client) Flush() {}
