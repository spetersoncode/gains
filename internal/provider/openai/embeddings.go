package openai

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	ai "github.com/spetersoncode/gains"
)

// Embed generates embeddings for the provided texts using OpenAI's embedding API.
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

	// Build request params
	params := openai.EmbeddingNewParams{
		Model: openai.EmbeddingModel(model.String()),
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: texts,
		},
	}

	// Apply dimensions (only for text-embedding-3-* models)
	if options.Dimensions > 0 {
		params.Dimensions = openai.Int(int64(options.Dimensions))
	}

	// Make API call
	resp, err := c.client.Embeddings.New(ctx, params)
	if err != nil {
		return nil, err
	}

	// Convert response - embeddings are returned in order
	embeddings := make([][]float64, len(resp.Data))
	for i, data := range resp.Data {
		embeddings[i] = data.Embedding
	}

	return &ai.EmbeddingResponse{
		Embeddings: embeddings,
		Usage: ai.Usage{
			InputTokens:  int(resp.Usage.PromptTokens),
			OutputTokens: 0, // Embeddings don't have output tokens
		},
	}, nil
}
