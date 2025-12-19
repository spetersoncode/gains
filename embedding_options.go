package gains

// EmbeddingOptions contains configuration for an embedding request.
type EmbeddingOptions struct {
	Model       Model
	Dimensions  int
	TaskType    EmbeddingTaskType
	RetryConfig *RetryConfig // Per-call retry config override (nil = use client default)
}

// EmbeddingOption is a functional option for configuring embedding requests.
type EmbeddingOption func(*EmbeddingOptions)

// WithEmbeddingModel sets the model to use for embedding generation.
func WithEmbeddingModel(model Model) EmbeddingOption {
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

// WithEmbeddingRetry overrides the client's default retry configuration for this request.
func WithEmbeddingRetry(cfg RetryConfig) EmbeddingOption {
	return func(o *EmbeddingOptions) {
		o.RetryConfig = &cfg
	}
}

// WithEmbeddingRetryDisabled disables retry for this request (single attempt).
func WithEmbeddingRetryDisabled() EmbeddingOption {
	return func(o *EmbeddingOptions) {
		disabled := DisabledRetryConfig()
		o.RetryConfig = &disabled
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
