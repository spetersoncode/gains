package google

import (
	"context"
	"fmt"

	ai "github.com/spetersoncode/gains"
	"google.golang.org/genai"
)

// Embed generates embeddings for the provided texts using Google's embedding API.
func (c *Client) Embed(ctx context.Context, texts []string, opts ...ai.EmbeddingOption) (*ai.EmbeddingResponse, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("%w: at least one text is required for embedding", ai.ErrEmptyInput)
	}

	options := ai.ApplyEmbeddingOptions(opts...)

	// Determine model
	model := DefaultEmbeddingModel
	if options.Model != nil {
		model = EmbeddingModel(options.Model.String())
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
	resp, err := c.client.Models.EmbedContent(ctx, model.String(), contents, config)
	if err != nil {
		return nil, wrapError(err)
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

	return &ai.EmbeddingResponse{
		Embeddings: embeddings,
		Usage: ai.Usage{
			// Google doesn't return token usage for embeddings
			InputTokens:  0,
			OutputTokens: 0,
		},
	}, nil
}
