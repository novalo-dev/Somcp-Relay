// Package llm talks to the local AI through the API most local runners
// share with OpenAI: Ollama, LM Studio, llama.cpp, LocalAI, Jan and vLLM
// all answer /v1/models and /v1/chat/completions. One format covers them
// all, so the relay never needs to know which one is installed.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MaxBody caps what is read from the AI or sent to it: a chat request with
// a long history is large, but never gigabytes.
const MaxBody = 32 << 20

// Client calls one local AI, at the address the user typed.
type Client struct {
	base string
	http *http.Client
}

// Model is one model the AI offers, and whether it passed the tool test.
type Model struct {
	ID string `json:"id"`
	// Tools is nil until tested: a model that cannot call tools is useless
	// to the shop, whose assistant works through its tools.
	Tools *bool `json:"tools,omitempty"`
}

// New checks the address and returns a client for it. The address is the
// one the user typed; "/v1" is added when it is missing, since Ollama and
// LM Studio are usually given as host and port only.
func New(address string) (*Client, error) {
	base, err := Normalize(address)
	if err != nil {
		return nil, err
	}
	return &Client{base: base, http: &http.Client{Timeout: 10 * time.Minute}}, nil
}

// Normalize turns a typed address into the API's base URL.
func Normalize(address string) (string, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return "", errors.New("empty address")
	}
	if !strings.Contains(address, "://") {
		address = "http://" + address
	}
	u, err := url.Parse(address)
	if err != nil {
		return "", err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
	if u.Host == "" {
		return "", errors.New("no host in the address")
	}
	path := strings.TrimRight(u.Path, "/")
	if !strings.HasSuffix(path, "/v1") {
		path += "/v1"
	}
	u.Path = path
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

// Base is the API's base URL, for display and for Ollama's own API.
func (c *Client) Base() string {
	return c.base
}

// Models lists the models the AI offers.
func (c *Client) Models(ctx context.Context) ([]Model, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/models", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var listed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, MaxBody)).Decode(&listed); err != nil {
		return nil, err
	}
	models := make([]Model, 0, len(listed.Data))
	for _, m := range listed.Data {
		if m.ID != "" {
			models = append(models, Model{ID: m.ID})
		}
	}
	return models, nil
}

// TestTools asks a model to call a made-up tool and reports whether it did.
// The shop's assistant does everything through tools: a model that answers
// in prose instead would leave the merchant with nothing done.
func (c *Client) TestTools(ctx context.Context, model string) (bool, error) {
	body := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "You can call tools. Use them when asked."},
			{"role": "user", "content": "What time is it in Paris? Use the tool."},
		},
		"tools": []map[string]any{{
			"type": "function",
			"function": map[string]any{
				"name":        "get_time",
				"description": "Gives the current time in a city.",
				"parameters": map[string]any{
					"type":       "object",
					"properties": map[string]any{"city": map[string]string{"type": "string"}},
					"required":   []string{"city"},
				},
			},
		}},
		"stream": false,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return false, err
	}
	status, answer, err := c.Chat(ctx, raw)
	if err != nil {
		return false, err
	}
	if status != http.StatusOK {
		return false, fmt.Errorf("HTTP %d", status)
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				ToolCalls []struct {
					Function struct {
						Name string `json:"name"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(answer, &parsed); err != nil {
		return false, err
	}
	for _, choice := range parsed.Choices {
		for _, call := range choice.Message.ToolCalls {
			if call.Function.Name == "get_time" {
				return true, nil
			}
		}
	}
	return false, nil
}

// Chat sends a chat completion request as it came and returns the AI's
// status and body as they came. The relay checks the request before, so
// this only carries it.
func (c *Client) Chat(ctx context.Context, request []byte) (int, []byte, error) {
	if len(request) > MaxBody {
		return 0, nil, errors.New("request too large")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/chat/completions", bytes.NewReader(request))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	answer, err := io.ReadAll(io.LimitReader(resp.Body, MaxBody))
	return resp.StatusCode, answer, err
}
