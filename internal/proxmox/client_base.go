package proxmox

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
)

// Client handles communication with Proxmox VE API
type Client struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client
	logger     *logrus.Entry
}

// NewClient creates a new Proxmox VE API client
func NewClient(baseURL, apiToken string, skipSSLVerify bool) *Client {
	var tlsConfig *tls.Config
	if skipSSLVerify {
		// Disable SSL verification for self-signed certificates
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return &Client{
		baseURL:    baseURL,
		apiToken:   apiToken,
		httpClient: httpClient,
		logger:     logrus.WithField("component", "ProxmoxClient"),
	}
}

// doRequest performs an HTTP request to the Proxmox API
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body interface{}) (interface{}, error) {
	urlStr := fmt.Sprintf("%s/api2/json/%s", c.baseURL, endpoint)

	var reqBody io.Reader
	var contentType string

	// Handle query parameters for GET requests
	if method == "GET" && body != nil {
		// For GET requests, convert body to query parameters
		if params, ok := body.(map[string]interface{}); ok {
			q := url.Values{}
			for key, value := range params {
				q.Set(key, fmt.Sprintf("%v", value))
			}
			urlStr = urlStr + "?" + q.Encode()
			body = nil // No body for GET requests with query params
		}
	}

	// Handle JSON body for POST/PUT requests
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
		contentType = "application/json"
	}

	req, err := http.NewRequestWithContext(ctx, method, urlStr, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set authentication header
	req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s", c.apiToken))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return apiResp.Data, nil
}
