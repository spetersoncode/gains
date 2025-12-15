package gains

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// testEmbeddingModel is a simple Model implementation for testing.
type testEmbeddingModel string

func (m testEmbeddingModel) String() string { return string(m) }

func TestApplyEmbeddingOptions(t *testing.T) {
	t.Run("returns empty options when no options provided", func(t *testing.T) {
		opts := ApplyEmbeddingOptions()
		assert.NotNil(t, opts)
		assert.Nil(t, opts.Model)
		assert.Zero(t, opts.Dimensions)
		assert.Empty(t, opts.TaskType)
	})

	t.Run("applies multiple options", func(t *testing.T) {
		opts := ApplyEmbeddingOptions(
			WithEmbeddingModel(testEmbeddingModel("text-embedding-3-large")),
			WithEmbeddingDimensions(1024),
			WithEmbeddingTaskType(EmbeddingTaskTypeRetrievalQuery),
		)

		assert.Equal(t, "text-embedding-3-large", opts.Model.String())
		assert.Equal(t, 1024, opts.Dimensions)
		assert.Equal(t, EmbeddingTaskTypeRetrievalQuery, opts.TaskType)
	})
}

func TestWithEmbeddingModel(t *testing.T) {
	tests := []struct {
		name     string
		model    testEmbeddingModel
		expected string
	}{
		{"sets OpenAI model", "text-embedding-3-small", "text-embedding-3-small"},
		{"sets OpenAI large model", "text-embedding-3-large", "text-embedding-3-large"},
		{"sets Google model", "text-embedding-004", "text-embedding-004"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ApplyEmbeddingOptions(WithEmbeddingModel(tt.model))
			assert.Equal(t, tt.expected, opts.Model.String())
		})
	}
}

func TestWithEmbeddingDimensions(t *testing.T) {
	tests := []struct {
		name     string
		dims     int
		expected int
	}{
		{"sets 256 dimensions", 256, 256},
		{"sets 512 dimensions", 512, 512},
		{"sets 1024 dimensions", 1024, 1024},
		{"sets 1536 dimensions", 1536, 1536},
		{"sets 3072 dimensions", 3072, 3072},
		{"handles zero", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ApplyEmbeddingOptions(WithEmbeddingDimensions(tt.dims))
			assert.Equal(t, tt.expected, opts.Dimensions)
		})
	}
}

func TestWithEmbeddingTaskType(t *testing.T) {
	tests := []struct {
		name     string
		taskType EmbeddingTaskType
		expected EmbeddingTaskType
	}{
		{"sets retrieval query", EmbeddingTaskTypeRetrievalQuery, EmbeddingTaskTypeRetrievalQuery},
		{"sets retrieval document", EmbeddingTaskTypeRetrievalDocument, EmbeddingTaskTypeRetrievalDocument},
		{"sets semantic similarity", EmbeddingTaskTypeSemanticSimilarity, EmbeddingTaskTypeSemanticSimilarity},
		{"sets classification", EmbeddingTaskTypeClassification, EmbeddingTaskTypeClassification},
		{"sets clustering", EmbeddingTaskTypeClustering, EmbeddingTaskTypeClustering},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ApplyEmbeddingOptions(WithEmbeddingTaskType(tt.taskType))
			assert.Equal(t, tt.expected, opts.TaskType)
		})
	}
}

func TestEmbeddingTaskTypeConstants(t *testing.T) {
	assert.Equal(t, EmbeddingTaskType("RETRIEVAL_QUERY"), EmbeddingTaskTypeRetrievalQuery)
	assert.Equal(t, EmbeddingTaskType("RETRIEVAL_DOCUMENT"), EmbeddingTaskTypeRetrievalDocument)
	assert.Equal(t, EmbeddingTaskType("SEMANTIC_SIMILARITY"), EmbeddingTaskTypeSemanticSimilarity)
	assert.Equal(t, EmbeddingTaskType("CLASSIFICATION"), EmbeddingTaskTypeClassification)
	assert.Equal(t, EmbeddingTaskType("CLUSTERING"), EmbeddingTaskTypeClustering)
}

func TestEmbeddingOptionsOverride(t *testing.T) {
	t.Run("later option overrides earlier", func(t *testing.T) {
		opts := ApplyEmbeddingOptions(
			WithEmbeddingModel(testEmbeddingModel("first-model")),
			WithEmbeddingDimensions(256),
			WithEmbeddingModel(testEmbeddingModel("second-model")),
			WithEmbeddingDimensions(512),
		)
		assert.Equal(t, "second-model", opts.Model.String())
		assert.Equal(t, 512, opts.Dimensions)
	})
}

func TestEmbeddingResponseStruct(t *testing.T) {
	t.Run("creates response with embeddings", func(t *testing.T) {
		resp := EmbeddingResponse{
			Embeddings: [][]float64{
				{0.1, 0.2, 0.3},
				{0.4, 0.5, 0.6},
			},
			Usage: Usage{
				InputTokens:  20,
				OutputTokens: 0,
			},
		}

		assert.Len(t, resp.Embeddings, 2)
		assert.Equal(t, []float64{0.1, 0.2, 0.3}, resp.Embeddings[0])
		assert.Equal(t, []float64{0.4, 0.5, 0.6}, resp.Embeddings[1])
		assert.Equal(t, 20, resp.Usage.InputTokens)
	})
}
