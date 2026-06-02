package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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

func userAgent() string {
	return fmt.Sprintf("LexSelect-CLI/%s (%s/%s)", config.CLIVersion, runtime.GOOS, runtime.GOARCH)
}

// apiErrorFrom builds an APIError from a status code and a parsed problem+json body.
func apiErrorFrom(status int, body map[string]interface{}) *APIError {
	e := &APIError{StatusCode: status, Body: body}
	if body != nil {
		e.Title, _ = body["title"].(string)
		e.Detail, _ = body["detail"].(string)
	}
	return e
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
	req.Header.Set("User-Agent", userAgent())

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

		return nil, retryAfter, apiErrorFrom(resp.StatusCode, result)
	}

	return result, 0, nil
}

// UploadMultipart performs the single-request upload (POST /documents/upload):
// the server obtains the presigned URL, PUTs to S3, computes the hash, completes,
// and triggers processing. The `name` field MUST be written before the file part —
// the gateway parses the stream and needs the name before the file arrives.
func (c *Client) UploadMultipart(ctx context.Context, name, parentID, contentType string, data []byte) (map[string]interface{}, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("name", name)
	if parentID != "" {
		_ = w.WriteField("parent_id", parentID)
	}
	if contentType != "" {
		_ = w.WriteField("content_type", contentType)
	}
	part, err := w.CreateFormFile("file", name)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write file part: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize multipart body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.cfg.APIURL+"/documents/upload", &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("X-API-Version", c.cfg.APIVersion)
	req.Header.Set("User-Agent", userAgent())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
	}
	if resp.StatusCode >= 400 {
		return nil, apiErrorFrom(resp.StatusCode, result)
	}
	return result, nil
}

// RequestRaw performs a GET and returns the raw response body plus its
// Content-Type. Used for endpoints that return a non-JSON body (e.g. /render,
// which returns Markdown or HTML).
func (c *Client) RequestRaw(ctx context.Context, method, path string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.cfg.APIURL+path, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	req.Header.Set("X-API-Version", c.cfg.APIVersion)
	req.Header.Set("User-Agent", userAgent())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		var result map[string]interface{}
		_ = json.Unmarshal(body, &result)
		return nil, "", apiErrorFrom(resp.StatusCode, result)
	}
	return body, resp.Header.Get("Content-Type"), nil
}
