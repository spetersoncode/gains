// Package vertex provides a Vertex AI client for the gains library.
//
// Vertex AI is Google Cloud's AI platform that provides access to Gemini models
// using Google Cloud authentication (Application Default Credentials) instead
// of API keys. This is the preferred method for production deployments on GCP.
//
// # Authentication
//
// Vertex AI uses Application Default Credentials (ADC) which automatically
// discovers credentials in the following order:
//
//  1. GOOGLE_APPLICATION_CREDENTIALS environment variable (path to service account key)
//  2. gcloud CLI credentials (gcloud auth application-default login)
//  3. Attached service account (GKE Workload Identity, Compute Engine, Cloud Run)
//
// # Usage
//
//	client, err := vertex.New(ctx, "my-project", "us-central1")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := client.Chat(ctx, messages, gains.WithModel(model.VertexGemini25Flash))
//
// # Available Regions
//
// Common Vertex AI regions include: us-central1, us-east4, us-west1,
// europe-west1, europe-west4, asia-northeast1, asia-southeast1.
// See https://cloud.google.com/vertex-ai/docs/general/locations for full list.
package vertex
