package gains

// EmbeddingOptions contains configuration for an embedding request.
type EmbeddingOptions struct {
	Model      string
	Dimensions int
	TaskType   EmbeddingTaskType
}

// EmbeddingOption is a functional option for configuring embedding requests.
type EmbeddingOption func(*EmbeddingOptions)

// WithEmbeddingModel sets the model to use for embedding generation.
func WithEmbeddingModel(model string) EmbeddingOption {
	return func(o *EmbeddingOptions) {
		o.Model = model
	}
}

// WithEmbeddingDimensions sets the output dimensions for the embedding vectors.
// Note: Only supported by OpenAI (text-embedding-3-*) and Google.
// For OpenAI, valid values depend on the model (e.g., 256, 512, 1024, 1536, 3072).
func WithEmbeddingDimensions(dims int) EmbeddingOption {
	return func(o *EmbeddingOptions) {
		o.Dimensions = dims
	}
}

// WithEmbeddingTaskType sets the intended task type for embeddings.
// This helps the model optimize the embedding for specific use cases.
// Note: Only supported by Google; ignored by OpenAI.
func WithEmbeddingTaskType(taskType EmbeddingTaskType) EmbeddingOption {
	return func(o *EmbeddingOptions) {
		o.TaskType = taskType
	}
}

// ApplyEmbeddingOptions applies functional options to an EmbeddingOptions struct.
func ApplyEmbeddingOptions(opts ...EmbeddingOption) *EmbeddingOptions {
	o := &EmbeddingOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
