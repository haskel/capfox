package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client for capfox API
type Client struct {
	baseURL  string
	client   *http.Client
	user     string
	password string
}

// NewClient creates a new API client
func NewClient() *Client {
	return &Client{
		baseURL: GetServerURL(),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		user:     user,
		password: password,
	}
}

// Get performs a GET request
func (c *Client) Get(path string) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, 0, err
	}

	return c.do(req)
}

// Post performs a POST request with JSON body
func (c *Client) Post(path string, body any) ([]byte, int, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, 0, err
		}
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, &buf)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	return c.do(req)
}

func (c *Client) do(req *http.Request) ([]byte, int, error) {
	// Add auth if provided
	if c.user != "" && c.password != "" {
		req.SetBasicAuth(c.user, c.password)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response: %w", err)
	}

	return data, resp.StatusCode, nil
}

// Health checks if server is running
func (c *Client) Health() error {
	_, status, err := c.Get("/health")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("server returned status %d", status)
	}
	return nil
}
