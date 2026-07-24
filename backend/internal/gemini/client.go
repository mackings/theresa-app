package gemini

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

type Client struct {
	genai     *genai.Client
	textModel string
}

func NewClient(ctx context.Context, apiKey, textModel string) (*Client, error) {
	gc, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("gemini client: %w", err)
	}
	return &Client{genai: gc, textModel: textModel}, nil
}

// Genai exposes the underlying SDK client for packages (like gemini/live)
// that need direct access to SDK features this package doesn't wrap itself.
func (c *Client) Genai() *genai.Client {
	return c.genai
}
