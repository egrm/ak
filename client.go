package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"
)

type Client struct {
	url     string
	checked bool
}

type ankiRequest struct {
	Action  string `json:"action"`
	Version int    `json:"version"`
	Params  any    `json:"params,omitempty"`
}

type ankiResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *string         `json:"error"`
}

func NewClient() *Client {
	return &Client{url: "http://localhost:8765"}
}

func (c *Client) Call(action string, params any) (json.RawMessage, error) {
	if !c.checked {
		if err := c.ensureRunning(); err != nil {
			return nil, err
		}
		c.checked = true
	}

	req := ankiRequest{Action: action, Version: 6, Params: params}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	resp, err := http.Post(c.url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var ar ankiResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	if ar.Error != nil {
		return nil, fmt.Errorf("%s", *ar.Error)
	}

	return ar.Result, nil
}

func (c *Client) ensureRunning() error {
	if c.ping() {
		return nil
	}

	// Try macOS auto-launch; if it fails (wrong OS, Anki not installed), just skip
	if err := exec.Command("open", "-a", "Anki").Start(); err != nil {
		return fmt.Errorf("AnkiConnect not reachable at %s — start Anki with AnkiConnect plugin enabled", c.url)
	}

	fmt.Fprintln(os.Stderr, "Waiting for Anki to start...")
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)
		if c.ping() {
			return nil
		}
	}

	return fmt.Errorf("Anki launched but AnkiConnect not responding at %s — is the AnkiConnect plugin installed?", c.url)
}

func (c *Client) ping() bool {
	cl := &http.Client{Timeout: 2 * time.Second}
	resp, err := cl.Post(c.url, "application/json",
		bytes.NewReader([]byte(`{"action":"version","version":6}`)))
	if err != nil {
		return false
	}
	resp.Body.Close()
	return true
}
