// Package embedding provides an HTTP client for Ollama embeddings.
package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sgxdev/same/internal/config"
)

// Client wraps an HTTP client for Ollama embedding requests.
type Client struct {
	httpClient *http.Client
	baseURL    string
	model      string
}

// NewClient creates a new Ollama embedding client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 60 * time.Second},
		baseURL:    config.OllamaURL(),
		model:      config.EmbeddingModel,
	}
}

type embeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type embeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

// GetEmbedding returns an embedding vector for the given text with a prefix.
// The prefix should be "search_document" for indexing or "search_query" for queries.
func (c *Client) GetEmbedding(text string, prefix string) ([]float32, error) {
	prompt := prefix + ": " + text

	body, err := json.Marshal(embeddingRequest{
		Model:  c.model,
		Prompt: prompt,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/embeddings",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	// If 500 (likely context overflow), retry with truncated text
	if resp.StatusCode == http.StatusInternalServerError && len(text) > 3000 {
		resp.Body.Close()
		truncated := text[:len(text)/2]
		return c.GetEmbedding(truncated, prefix)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding returned")
	}

	return result.Embedding, nil
}

// GetDocumentEmbedding returns an embedding for document indexing.
func (c *Client) GetDocumentEmbedding(text string) ([]float32, error) {
	return c.GetEmbedding(text, "search_document")
}

// GetQueryEmbedding returns an embedding for search queries.
func (c *Client) GetQueryEmbedding(text string) ([]float32, error) {
	return c.GetEmbedding(text, "search_query")
}
