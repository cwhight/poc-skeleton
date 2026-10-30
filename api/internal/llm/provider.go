// Package llm is the vendor-neutral LLM seam.
//
// Provider is the swappable interface; a concrete impl is selected by
// LLM_PROVIDER at startup (New(name, key, model)). NO provider SDK leaks beyond
// this package, and user-facing copy never names the model — the model you
// launch on is rarely the one you keep. Stream emits tokens over a channel so
// the handler can write SSE incrementally.
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Provider is the swappable seam.
type Provider interface {
	// Stream sends system+input to the model and emits output tokens until done.
	// The token channel closes when the stream ends; the error channel carries
	// at most one terminal error (also closed when done).
	Stream(ctx context.Context, systemPrompt, input string) (<-chan string, <-chan error)
}

// httpProvider is a generic SSE-style streaming HTTP client for a chat/generate
// API. It is intentionally vendor-neutral: the wire shape (a JSON body, Bearer
// auth, "data: " framed tokens, "[DONE]" sentinel) is configured, not hardwired
// to any one vendor. Swap the endpoint/body via env to target a real provider.
type httpProvider struct {
	apiKey   string
	model    string
	endpoint string
	client   *http.Client
}

// New selects the impl by name (e.g. "gemini", "openai", "anthropic"). For now
// every recognised (non-empty) name maps to the generic streaming HTTP client;
// add per-vendor request/response shaping here as needed without leaking it out
// of this package.
func New(name, apiKey, model string) (Provider, error) {
	if name == "" {
		return nil, errors.New("llm: provider name is required (set LLM_PROVIDER)")
	}
	if apiKey == "" {
		return nil, errors.New("llm: api key is required (set LLM_API_KEY)")
	}

	endpoint := os.Getenv("LLM_ENDPOINT")
	if endpoint == "" {
		// Placeholder; a real POC sets LLM_ENDPOINT via Terraform.
		endpoint = "https://api.example.com/v1/stream"
	}

	return &httpProvider{
		apiKey:   apiKey,
		model:    model,
		endpoint: endpoint,
		client:   &http.Client{}, // NO client timeout — this is a long stream.
	}, nil
}

func (p *httpProvider) Stream(ctx context.Context, systemPrompt, input string) (<-chan string, <-chan error) {
	out := make(chan string)
	errCh := make(chan error, 1)

	go func() {
		defer close(out)
		defer close(errCh)

		body, err := json.Marshal(map[string]any{
			"model":  p.model,
			"system": systemPrompt,
			"input":  input,
			"stream": true,
		})
		if err != nil {
			errCh <- fmt.Errorf("llm: marshal request: %w", err)
			return
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
		if err != nil {
			errCh <- fmt.Errorf("llm: build request: %w", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
		req.Header.Set("Accept", "text/event-stream")

		resp, err := p.client.Do(req)
		if err != nil {
			errCh <- fmt.Errorf("llm: request failed: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			errCh <- fmt.Errorf("llm: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(snippet)))
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		// Allow long token lines (default 64KB cap can overflow on large chunks).
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "[DONE]" {
				return
			}

			var chunk struct {
				Token string `json:"token"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				errCh <- fmt.Errorf("llm: decode chunk: %w", err)
				return
			}
			if chunk.Token == "" {
				continue
			}

			select {
			case out <- chunk.Token:
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}
		}

		if err := scanner.Err(); err != nil {
			errCh <- fmt.Errorf("llm: read stream: %w", err)
		}
	}()

	return out, errCh
}
