package gains

import "context"

// EmbeddingProvider defines the interface for AI embedding providers.
type EmbeddingProvider interface {
	// Embed generates embeddings for the provided texts.
	// Returns an error if texts is empty.
	Embed(ctx context.Context, texts []string, opts ...EmbeddingOption) (*EmbeddingResponse, error)
}

// EmbeddingResponse represents a complete response from an embedding provider.
type EmbeddingResponse struct {
	// Embeddings contains one embedding vector per input text.
	// The order matches the input texts order.
	Embeddings [][]float64
	// Usage contains token usage information.
	Usage Usage
}

// EmbeddingTaskType specifies the intended use case for embeddings.
// This helps the model optimize the embedding for specific tasks.
// Note: Only supported by Google; ignored by OpenAI.
type EmbeddingTaskType string

const (
	// EmbeddingTaskTypeRetrievalQuery optimizes for search queries.
	EmbeddingTaskTypeRetrievalQuery EmbeddingTaskType = "RETRIEVAL_QUERY"
	// EmbeddingTaskTypeRetrievalDocument optimizes for documents to be searched.
	EmbeddingTaskTypeRetrievalDocument EmbeddingTaskType = "RETRIEVAL_DOCUMENT"
	// EmbeddingTaskTypeSemanticSimilarity optimizes for measuring text similarity.
	EmbeddingTaskTypeSemanticSimilarity EmbeddingTaskType = "SEMANTIC_SIMILARITY"
	// EmbeddingTaskTypeClassification optimizes for classification tasks.
	EmbeddingTaskTypeClassification EmbeddingTaskType = "CLASSIFICATION"
	// EmbeddingTaskTypeClustering optimizes for clustering tasks.
	EmbeddingTaskTypeClustering EmbeddingTaskType = "CLUSTERING"
)
