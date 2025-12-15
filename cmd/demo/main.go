package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/client"
)

var reader = bufio.NewReader(os.Stdin)

func main() {
	godotenv.Load()
	ctx := context.Background()

	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║       gains - AI Library Demo          ║")
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Println()

	// Check available providers
	providers := []struct {
		name   client.ProviderName
		envKey string
		label  string
	}{
		{client.ProviderAnthropic, "ANTHROPIC_API_KEY", "Anthropic (Claude)"},
		{client.ProviderOpenAI, "OPENAI_API_KEY", "OpenAI (GPT-4)"},
		{client.ProviderGoogle, "GOOGLE_API_KEY", "Google (Gemini)"},
	}

	var available []struct {
		name   client.ProviderName
		apiKey string
		label  string
	}

	fmt.Println("Available providers:")
	for _, p := range providers {
		if key := os.Getenv(p.envKey); key != "" {
			fmt.Printf("  [%d] %s\n", len(available)+1, p.label)
			available = append(available, struct {
				name   client.ProviderName
				apiKey string
				label  string
			}{p.name, key, p.label})
		}
	}

	if len(available) == 0 {
		fmt.Println("  ✗ No API keys found. Set ANTHROPIC_API_KEY, OPENAI_API_KEY, or GOOGLE_API_KEY.")
		return
	}
	fmt.Println()

	// Let user select provider
	var selected int
	if len(available) == 1 {
		selected = 0
		fmt.Printf("Using %s (only available provider)\n", available[0].label)
	} else {
		fmt.Printf("Select provider [1-%d]: ", len(available))
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)
		fmt.Sscanf(answer, "%d", &selected)
		selected-- // Convert to 0-indexed
		if selected < 0 || selected >= len(available) {
			selected = 0
		}
		fmt.Printf("Using %s\n", available[selected].label)
	}
	fmt.Println()

	// Create unified client
	c, err := client.New(ctx, client.Config{
		Provider: available[selected].name,
		APIKey:   available[selected].apiKey,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create client: %v\n", err)
		return
	}

	// Show supported features
	fmt.Println("Supported features:")
	fmt.Printf("  Chat:       ✓\n")
	if c.SupportsFeature(client.FeatureImage) {
		fmt.Printf("  Images:     ✓\n")
	} else {
		fmt.Printf("  Images:     ✗\n")
	}
	if c.SupportsFeature(client.FeatureEmbedding) {
		fmt.Printf("  Embeddings: ✓\n")
	} else {
		fmt.Printf("  Embeddings: ✗\n")
	}
	fmt.Println()

	// Demo: Chat Streaming
	if askYesNo("Demo chat streaming?") {
		demoChatStreaming(ctx, c)
	}

	// Demo: Vision/Image Input
	if askYesNo("Demo vision/image input?") {
		demoVisionInput(ctx, c)
	}

	// Demo: Image Generation
	if c.SupportsFeature(client.FeatureImage) {
		if askYesNo("Demo image generation?") {
			demoImageGeneration(ctx, c)
		}
	}

	// Demo: Tool Calling
	if askYesNo("Demo tool/function calling?") {
		demoToolCalling(ctx, c)
	}

	// Demo: JSON Mode / Structured Output
	if askYesNo("Demo JSON mode / structured output?") {
		demoJSONMode(ctx, c)
	}

	// Demo: Embeddings
	if c.SupportsFeature(client.FeatureEmbedding) {
		if askYesNo("Demo embeddings?") {
			demoEmbeddings(ctx, c)
		}
	}

	fmt.Println("\n✨ Demo complete!")
}

func askYesNo(question string) bool {
	fmt.Printf("%s [y/N]: ", question)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

func demoChatStreaming(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│          Chat Streaming Demo            │")
	fmt.Println("└─────────────────────────────────────────┘")

	messages := []gains.Message{
		{Role: gains.RoleUser, Content: "Say hello in 3 different languages, one per line."},
	}

	stream, err := c.ChatStream(ctx, messages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	fmt.Printf("\n%s:\n", c.Provider())
	for event := range stream {
		if event.Err != nil {
			fmt.Fprintf(os.Stderr, "Stream error: %v\n", event.Err)
			return
		}
		fmt.Print(event.Delta)
		if event.Done {
			fmt.Printf("\n[Tokens: %d in, %d out]\n",
				event.Response.Usage.InputTokens,
				event.Response.Usage.OutputTokens)
		}
	}
}

func demoVisionInput(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│         Vision/Image Input Demo         │")
	fmt.Println("└─────────────────────────────────────────┘")

	// Use a public domain image URL
	imageURL := "https://upload.wikimedia.org/wikipedia/commons/thumb/4/47/PNG_transparency_demonstration_1.png/300px-PNG_transparency_demonstration_1.png"

	messages := []gains.Message{
		{
			Role: gains.RoleUser,
			Parts: []gains.ContentPart{
				gains.NewTextPart("Describe this image in one sentence. What do you see?"),
				gains.NewImageURLPart(imageURL),
			},
		},
	}

	fmt.Printf("Image URL: %s\n", imageURL)
	fmt.Println("Question: Describe this image in one sentence. What do you see?")
	fmt.Println()

	resp, err := c.Chat(ctx, messages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", resp.Content)
	fmt.Printf("[Tokens: %d in, %d out]\n", resp.Usage.InputTokens, resp.Usage.OutputTokens)
}

func demoImageGeneration(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│         Image Generation Demo           │")
	fmt.Println("└─────────────────────────────────────────┘")

	prompt := "A serene mountain landscape at sunset with a calm lake reflection"
	fmt.Printf("Prompt: %q\n\n", prompt)

	resp, err := c.GenerateImage(ctx, prompt,
		gains.WithImageSize(gains.ImageSize1024x1024),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	for i, img := range resp.Images {
		fmt.Printf("Image %d:\n", i+1)
		if img.URL != "" {
			fmt.Printf("  URL: %s\n", img.URL)
		}
		if img.Base64 != "" {
			fmt.Printf("  Base64: %d bytes\n", len(img.Base64))
		}
		if img.RevisedPrompt != "" {
			fmt.Printf("  Revised prompt: %s\n", img.RevisedPrompt)
		}
	}
}

func demoToolCalling(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│          Tool Calling Demo              │")
	fmt.Println("└─────────────────────────────────────────┘")

	// Define a weather tool
	tools := []gains.Tool{
		{
			Name:        "get_weather",
			Description: "Get the current weather for a location",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"location": {
						"type": "string",
						"description": "The city name, e.g. San Francisco"
					},
					"unit": {
						"type": "string",
						"enum": ["celsius", "fahrenheit"],
						"description": "The temperature unit"
					}
				},
				"required": ["location"]
			}`),
		},
	}

	messages := []gains.Message{
		{Role: gains.RoleUser, Content: "What's the weather like in Tokyo?"},
	}

	fmt.Println("User: What's the weather like in Tokyo?")
	fmt.Println("Tools available: get_weather(location, unit)")
	fmt.Println()

	// First call: model should request tool use
	resp, err := c.Chat(ctx, messages, gains.WithTools(tools))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	if len(resp.ToolCalls) == 0 {
		fmt.Println("Model response (no tool call):", resp.Content)
		return
	}

	// Show the tool call
	tc := resp.ToolCalls[0]
	fmt.Printf("Model requested tool: %s\n", tc.Name)
	fmt.Printf("Arguments: %s\n", tc.Arguments)

	// Simulate tool execution
	toolResult := `{"temperature": 22, "unit": "celsius", "conditions": "Partly cloudy"}`
	fmt.Printf("Tool result: %s\n", toolResult)

	// Continue conversation with tool result
	messages = append(messages,
		gains.Message{
			Role:      gains.RoleAssistant,
			ToolCalls: resp.ToolCalls,
		},
		gains.NewToolResultMessage(gains.ToolResult{
			ToolCallID: tc.ID,
			Content:    toolResult,
		}),
	)

	// Second call: model should use the tool result
	finalResp, err := c.Chat(ctx, messages, gains.WithTools(tools))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	fmt.Printf("Final response: %s\n", finalResp.Content)
	fmt.Printf("[Tokens: %d in, %d out]\n", finalResp.Usage.InputTokens, finalResp.Usage.OutputTokens)
}

func demoJSONMode(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│      JSON Mode / Structured Output      │")
	fmt.Println("└─────────────────────────────────────────┘")

	// Define a schema for structured output
	schema := gains.ResponseSchema{
		Name:        "book_info",
		Description: "Information about a book",
		Schema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"title": {
					"type": "string",
					"description": "The book title"
				},
				"author": {
					"type": "string",
					"description": "The author's name"
				},
				"year": {
					"type": "integer",
					"description": "Publication year"
				},
				"genres": {
					"type": "array",
					"items": {"type": "string"},
					"description": "List of genres"
				}
			},
			"required": ["title", "author", "year", "genres"]
		}`),
	}

	messages := []gains.Message{
		{Role: gains.RoleUser, Content: "Give me information about the book '1984' by George Orwell."},
	}

	fmt.Println("User: Give me information about the book '1984' by George Orwell.")
	fmt.Println("Schema: book_info (title, author, year, genres)")
	fmt.Println()

	resp, err := c.Chat(ctx, messages, gains.WithResponseSchema(schema))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	fmt.Println("Raw JSON response:")
	fmt.Println(resp.Content)

	// Parse and display structured data
	var book struct {
		Title  string   `json:"title"`
		Author string   `json:"author"`
		Year   int      `json:"year"`
		Genres []string `json:"genres"`
	}
	if err := json.Unmarshal([]byte(resp.Content), &book); err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		return
	}

	fmt.Println("\nParsed data:")
	fmt.Printf("  Title:  %s\n", book.Title)
	fmt.Printf("  Author: %s\n", book.Author)
	fmt.Printf("  Year:   %d\n", book.Year)
	fmt.Printf("  Genres: %v\n", book.Genres)
	fmt.Printf("[Tokens: %d in, %d out]\n", resp.Usage.InputTokens, resp.Usage.OutputTokens)
}

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
