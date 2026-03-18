package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/LexSelect/lexselect-api-demo/cli/config"
)

// APIError represents a structured error from the API.
type APIError struct {
	StatusCode int
	Title      string
	Detail     string
	Body       map[string]interface{}
}

func (e *APIError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%d %s: %s", e.StatusCode, e.Title, e.Detail)
	}
	return fmt.Sprintf("%d %s", e.StatusCode, e.Title)
}

// Client wraps HTTP calls to the LexSelect API.
type Client struct {
	cfg        *config.Config
	httpClient *http.Client
}

// New creates a Client from the given config.
func New(cfg *config.Config) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key not set. Use --api-key flag or LEXSELECT_API_KEY env var")
	}
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// Request sends an API request and returns the parsed JSON body.
// Handles rate limiting with automatic retry (up to 3 attempts).
func (c *Client) Request(ctx context.Context, method, path string, body interface{}) (map[string]interface{}, error) {
	const maxRetries = 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		result, retryAfter, err := c.doRequest(ctx, method, path, body)
		if err == nil {
			return result, nil
		}

		apiErr, ok := err.(*APIError)
		if !ok || apiErr.StatusCode != 429 || attempt == maxRetries-1 {
			return nil, err
		}

		wait := time.Duration(retryAfter) * time.Second
		if wait == 0 {
			wait = 2 * time.Second
		}
		fmt.Printf("Rate limited, retrying in %s...\n", wait)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
	}

	return nil, fmt.Errorf("max retries exceeded")
}

func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (map[string]interface{}, int, error) {
	url := c.cfg.APIURL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Version", c.cfg.APIVersion)
	req.Header.Set("User-Agent", fmt.Sprintf("LexSelect-CLI/%s (%s/%s)", config.CLIVersion, runtime.GOOS, runtime.GOARCH))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, 0, fmt.Errorf("failed to parse response: %w", err)
		}
	}

	if resp.StatusCode >= 400 {
		retryAfter := 0
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			retryAfter, _ = strconv.Atoi(ra)
		}

		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			Body:       result,
		}
		if result != nil {
			apiErr.Title, _ = result["title"].(string)
			apiErr.Detail, _ = result["detail"].(string)
		}
		return nil, retryAfter, apiErr
	}

	return result, 0, nil
}

// UploadToS3 uploads file bytes to a presigned S3 URL.
func (c *Client) UploadToS3(ctx context.Context, uploadURL string, data []byte, contentType string) error {
	req, err := http.NewRequestWithContext(ctx, "PUT", uploadURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create S3 request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("S3 upload failed: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("S3 upload returned %d", resp.StatusCode)
	}
	return nil
}
