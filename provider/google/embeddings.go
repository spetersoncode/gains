package google

import (
	"context"
	"fmt"

	"github.com/spetersoncode/gains"
	"google.golang.org/genai"
)

// Embed generates embeddings for the provided texts using Google's embedding API.
func (c *Client) Embed(ctx context.Context, texts []string, opts ...gains.EmbeddingOption) (*gains.EmbeddingResponse, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("at least one text is required for embedding")
	}

	options := gains.ApplyEmbeddingOptions(opts...)

	// Determine model
	model := DefaultEmbeddingModel
	if options.Model != "" {
		model = options.Model
	}

	// Build embed content config
	config := &genai.EmbedContentConfig{}

	// Apply dimensions
	if options.Dimensions > 0 {
		dims := int32(options.Dimensions)
		config.OutputDimensionality = &dims
	}

	// Apply task type
	if options.TaskType != "" {
		config.TaskType = string(options.TaskType)
	}

	// Build contents from texts
	contents := make([]*genai.Content, len(texts))
	for i, text := range texts {
		contents[i] = &genai.Content{
			Parts: []*genai.Part{{Text: text}},
		}
	}

	// Make API call
	resp, err := c.client.Models.EmbedContent(ctx, model, contents, config)
	if err != nil {
		return nil, err
	}

	// Convert response
	embeddings := make([][]float64, len(resp.Embeddings))
	for i, emb := range resp.Embeddings {
		// Convert []float32 to []float64
		embeddings[i] = make([]float64, len(emb.Values))
		for j, v := range emb.Values {
			embeddings[i][j] = float64(v)
		}
	}

	return &gains.EmbeddingResponse{
		Embeddings: embeddings,
		Usage: gains.Usage{
			// Google doesn't return token usage for embeddings
			InputTokens:  0,
			OutputTokens: 0,
		},
	}, nil
}
