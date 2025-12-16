package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	ai "github.com/spetersoncode/gains"
)

// HTTPToolOption configures the HTTP tool.
type HTTPToolOption func(*httpToolConfig)

type httpToolConfig struct {
	client          *http.Client
	allowedHosts    []string
	blockedHosts    []string
	maxResponseSize int64
	timeout         time.Duration
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c *http.Client) HTTPToolOption {
	return func(cfg *httpToolConfig) {
		cfg.client = c
	}
}

// WithAllowedHosts restricts requests to specific hosts only.
func WithAllowedHosts(hosts ...string) HTTPToolOption {
	return func(cfg *httpToolConfig) {
		cfg.allowedHosts = hosts
	}
}

// WithBlockedHosts blocks requests to specific hosts.
func WithBlockedHosts(hosts ...string) HTTPToolOption {
	return func(cfg *httpToolConfig) {
		cfg.blockedHosts = hosts
	}
}

// WithMaxResponseSize sets the maximum response body size.
// Default is 1MB.
func WithMaxResponseSize(bytes int64) HTTPToolOption {
	return func(cfg *httpToolConfig) {
		cfg.maxResponseSize = bytes
	}
}

// WithHTTPTimeout sets the request timeout.
// Default is 30 seconds.
func WithHTTPTimeout(d time.Duration) HTTPToolOption {
	return func(cfg *httpToolConfig) {
		cfg.timeout = d
	}
}

func applyHTTPOpts(opts []HTTPToolOption) *httpToolConfig {
	cfg := &httpToolConfig{
		maxResponseSize: 1024 * 1024, // 1MB default
		timeout:         30 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.client == nil {
		cfg.client = &http.Client{
			Timeout: cfg.timeout,
		}
	}
	return cfg
}

func (c *httpToolConfig) checkHost(urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return err
	}

	host := u.Hostname()

	// Check blocked hosts
	for _, blocked := range c.blockedHosts {
		if host == blocked || strings.HasSuffix(host, "."+blocked) {
			return fmt.Errorf("host %q is blocked", host)
		}
	}

	// Check allowed hosts (if set)
	if len(c.allowedHosts) > 0 {
		allowed := false
		for _, a := range c.allowedHosts {
			if host == a || strings.HasSuffix(host, "."+a) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("host %q is not in allowed list", host)
		}
	}

	return nil
}

// httpArgs defines arguments for the HTTP request tool.
type httpArgs struct {
	URL     string            `json:"url" desc:"URL to request" required:"true"`
	Method  string            `json:"method" desc:"HTTP method" enum:"GET,POST,PUT,DELETE,PATCH"`
	Headers map[string]string `json:"headers" desc:"Request headers"`
	Body    string            `json:"body" desc:"Request body (for POST/PUT/PATCH)"`
}

// NewHTTPTool creates a tool for making HTTP requests.
func NewHTTPTool(opts ...HTTPToolOption) (ai.Tool, Handler) {
	cfg := applyHTTPOpts(opts)

	schema := MustSchemaFor[httpArgs]()

	t := ai.Tool{
		Name:        "http_request",
		Description: "Make an HTTP request to a URL",
		Parameters:  schema,
	}

	handler := func(ctx context.Context, call ai.ToolCall) (string, error) {
		var args httpArgs
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return "", err
		}

		// Validate host
		if err := cfg.checkHost(args.URL); err != nil {
			return "", err
		}

		// Default to GET
		method := args.Method
		if method == "" {
			method = "GET"
		}

		// Create request body
		var body io.Reader
		if args.Body != "" {
			body = bytes.NewBufferString(args.Body)
		}

		req, err := http.NewRequestWithContext(ctx, method, args.URL, body)
		if err != nil {
			return "", err
		}

		// Set headers
		for k, v := range args.Headers {
			req.Header.Set(k, v)
		}

		// Execute request
		resp, err := cfg.client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		// Read response with size limit
		respBody, err := io.ReadAll(io.LimitReader(resp.Body, cfg.maxResponseSize))
		if err != nil {
			return "", err
		}

		// Build response
		result := struct {
			Status     string            `json:"status"`
			StatusCode int               `json:"status_code"`
			Headers    map[string]string `json:"headers"`
			Body       string            `json:"body"`
			BodySize   int               `json:"body_size"`
		}{
			Status:     resp.Status,
			StatusCode: resp.StatusCode,
			Headers:    make(map[string]string),
			Body:       string(respBody),
			BodySize:   len(respBody),
		}

		// Copy important headers
		for _, h := range []string{"Content-Type", "Content-Length", "Date", "Server"} {
			if v := resp.Header.Get(h); v != "" {
				result.Headers[h] = v
			}
		}

		out, err := json.Marshal(result)
		if err != nil {
			return "", err
		}
		return string(out), nil
	}

	return t, handler
}

// WebTools returns the HTTP request tool.
func WebTools(opts ...HTTPToolOption) []ToolPair {
	t, h := NewHTTPTool(opts...)
	return []ToolPair{{Tool: t, Handler: h}}
}
