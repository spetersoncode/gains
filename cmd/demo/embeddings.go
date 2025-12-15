package main

import (
	"context"
	"fmt"
	"math"
	"os"

	"github.com/spetersoncode/gains/client"
)

func demoEmbeddings(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│            Embeddings Demo              │")
	fmt.Println("└─────────────────────────────────────────┘")

	texts := []string{
		"The quick brown fox jumps over the lazy dog.",
		"A fast auburn fox leaps above a sleepy canine.",
		"The weather is beautiful today.",
	}

	fmt.Println("Texts to embed:")
	for i, text := range texts {
		fmt.Printf("  %d. %q\n", i+1, text)
	}
	fmt.Println()

	resp, err := c.Embed(ctx, texts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	for i, emb := range resp.Embeddings {
		fmt.Printf("Text %d: %d dimensions (first 5: [%.4f, %.4f, %.4f, %.4f, %.4f]...)\n",
			i+1, len(emb), emb[0], emb[1], emb[2], emb[3], emb[4])
	}

	if resp.Usage.InputTokens > 0 {
		fmt.Printf("[Tokens: %d]\n", resp.Usage.InputTokens)
	}

	// Calculate cosine similarity between texts
	if len(resp.Embeddings) >= 3 {
		sim12 := cosineSimilarity(resp.Embeddings[0], resp.Embeddings[1])
		sim13 := cosineSimilarity(resp.Embeddings[0], resp.Embeddings[2])
		fmt.Printf("\nSimilarity(1,2): %.4f  Similarity(1,3): %.4f\n", sim12, sim13)
		fmt.Println("Text 1 and 2 are semantically similar (both about a fox)")
		fmt.Println("Text 3 is semantically different (about weather)")
	}
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
