//go:build bugstack

package bugstack

import (
	"fmt"
	"log"
	"os"
	"strings"

	sdk "github.com/MasonBachmann7/bugstack-go"
)

// Client wraps the BugStack Go SDK for server integration.
type Client struct {
	enabled bool
}

// InitFromEnv initializes BugStack when BUGSTACK_API_KEY is present.
func InitFromEnv(logger *log.Logger) *Client {
	if logger == nil {
		logger = log.Default()
	}

	apiKey := strings.TrimSpace(os.Getenv("BUGSTACK_API_KEY"))
	if apiKey == "" {
		logger.Printf("bugstack disabled: BUGSTACK_API_KEY is not set")
		return &Client{enabled: false}
	}

	endpoint := os.Getenv("BUGSTACK_ENDPOINT")
	logger.Printf("bugstack: APIKey length=%d, Endpoint=%q", len(apiKey), endpoint)

	sdk.Init(sdk.Config{
		APIKey:   apiKey,
		Endpoint: endpoint,
		Debug:    true,
	})
	logger.Printf("bugstack initialized (debug enabled)")
	return &Client{enabled: true}
}

func (c *Client) CaptureError(err error) {
	if c == nil || !c.enabled || err == nil {
		return
	}
	sdk.CaptureError(err)
}

func (c *Client) CapturePanic(value any) {
	if c == nil || !c.enabled {
		return
	}
	if err, ok := value.(error); ok {
		sdk.CaptureError(err)
		return
	}
	sdk.CaptureError(fmt.Errorf("panic: %v", value))
}

func (c *Client) Flush() {
	if c == nil || !c.enabled {
		return
	}
	sdk.Flush()
}
