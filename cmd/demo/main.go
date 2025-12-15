package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/provider/anthropic"
	"github.com/spetersoncode/gains/provider/google"
	"github.com/spetersoncode/gains/provider/openai"
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
	hasAnthropic := os.Getenv("ANTHROPIC_API_KEY") != ""
	hasOpenAI := os.Getenv("OPENAI_API_KEY") != ""
	hasGoogle := os.Getenv("GOOGLE_API_KEY") != ""

	fmt.Println("Available providers:")
	if hasAnthropic {
		fmt.Println("  ✓ Anthropic (Claude)")
	}
	if hasOpenAI {
		fmt.Println("  ✓ OpenAI (GPT-4)")
	}
	if hasGoogle {
		fmt.Println("  ✓ Google (Gemini)")
	}
	if !hasAnthropic && !hasOpenAI && !hasGoogle {
		fmt.Println("  ✗ No API keys found. Set ANTHROPIC_API_KEY, OPENAI_API_KEY, or GOOGLE_API_KEY.")
		return
	}
	fmt.Println()

	// Initialize clients
	var anthropicClient *anthropic.Client
	var openaiClient *openai.Client
	var googleClient *google.Client

	if hasAnthropic {
		anthropicClient = anthropic.New(os.Getenv("ANTHROPIC_API_KEY"))
	}
	if hasOpenAI {
		openaiClient = openai.New(os.Getenv("OPENAI_API_KEY"))
	}
	if hasGoogle {
		var err error
		googleClient, err = google.New(ctx, os.Getenv("GOOGLE_API_KEY"))
		if err != nil {
			fmt.Printf("Warning: Failed to initialize Google client: %v\n", err)
			hasGoogle = false
		}
	}

	// Demo: Chat Streaming
	if askYesNo("Demo chat streaming?") {
		demoChatStreaming(ctx, anthropicClient, openaiClient, googleClient)
	}

	// Demo: Vision/Image Input
	if askYesNo("Demo vision/image input?") {
		demoVisionInput(ctx, anthropicClient, openaiClient, googleClient)
	}

	// Demo: Image Generation
	if hasOpenAI || hasGoogle {
		if askYesNo("Demo image generation?") {
			demoImageGeneration(ctx, openaiClient, googleClient)
		}
	}

	// Demo: Tool Calling
	if askYesNo("Demo tool/function calling?") {
		demoToolCalling(ctx, anthropicClient, openaiClient, googleClient)
	}

	// Demo: JSON Mode / Structured Output
	if askYesNo("Demo JSON mode / structured output?") {
		demoJSONMode(ctx, anthropicClient, openaiClient, googleClient)
	}

	fmt.Println("\n✨ Demo complete!")
}

func askYesNo(question string) bool {
	fmt.Printf("%s [y/N]: ", question)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

func demoChatStreaming(ctx context.Context, anthropicClient *anthropic.Client, openaiClient *openai.Client, googleClient *google.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│          Chat Streaming Demo            │")
	fmt.Println("└─────────────────────────────────────────┘")

	messages := []gains.Message{
		{Role: gains.RoleUser, Content: "Say hello in 3 different languages, one per line."},
	}

	if anthropicClient != nil {
		fmt.Println("\n=== Anthropic (Claude) ===")
		streamChat(ctx, anthropicClient, messages)
	}

	if openaiClient != nil {
		fmt.Println("\n=== OpenAI (GPT-4) ===")
		streamChat(ctx, openaiClient, messages)
	}

	if googleClient != nil {
		fmt.Println("\n=== Google (Gemini) ===")
		streamChat(ctx, googleClient, messages)
	}
}

func streamChat(ctx context.Context, client gains.ChatProvider, messages []gains.Message) {
	stream, err := client.ChatStream(ctx, messages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

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

func demoVisionInput(ctx context.Context, anthropicClient *anthropic.Client, openaiClient *openai.Client, googleClient *google.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│         Vision/Image Input Demo         │")
	fmt.Println("└─────────────────────────────────────────┘")

	// Use a public domain image URL (Wikipedia commons)
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

	if anthropicClient != nil {
		fmt.Println("=== Anthropic (Claude) ===")
		demoVisionWithProvider(ctx, anthropicClient, messages)
	}

	if openaiClient != nil {
		fmt.Println("\n=== OpenAI (GPT-4) ===")
		demoVisionWithProvider(ctx, openaiClient, messages)
	}

	if googleClient != nil {
		fmt.Println("\n=== Google (Gemini) ===")
		demoVisionWithProvider(ctx, googleClient, messages)
	}
}

func demoVisionWithProvider(ctx context.Context, client gains.ChatProvider, messages []gains.Message) {
	resp, err := client.Chat(ctx, messages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", resp.Content)
	fmt.Printf("[Tokens: %d in, %d out]\n", resp.Usage.InputTokens, resp.Usage.OutputTokens)
}

func demoImageGeneration(ctx context.Context, openaiClient *openai.Client, googleClient *google.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│         Image Generation Demo           │")
	fmt.Println("└─────────────────────────────────────────┘")

	prompt := "A serene mountain landscape at sunset with a calm lake reflection"
	fmt.Printf("Prompt: %q\n", prompt)

	if openaiClient != nil {
		fmt.Println("\n=== OpenAI (DALL-E 3) ===")
		generateImage(ctx, openaiClient, prompt)
	}

	if googleClient != nil {
		fmt.Println("\n=== Google (Imagen) ===")
		generateImage(ctx, googleClient, prompt)
	}
}

func generateImage(ctx context.Context, client gains.ImageProvider, prompt string) {
	resp, err := client.GenerateImage(ctx, prompt,
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

func demoToolCalling(ctx context.Context, anthropicClient *anthropic.Client, openaiClient *openai.Client, googleClient *google.Client) {
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

	if anthropicClient != nil {
		fmt.Println("=== Anthropic (Claude) ===")
		demoToolWithProvider(ctx, anthropicClient, messages, tools)
	}

	if openaiClient != nil {
		fmt.Println("\n=== OpenAI (GPT-4) ===")
		demoToolWithProvider(ctx, openaiClient, messages, tools)
	}

	if googleClient != nil {
		fmt.Println("\n=== Google (Gemini) ===")
		demoToolWithProvider(ctx, googleClient, messages, tools)
	}
}

func demoToolWithProvider(ctx context.Context, client gains.ChatProvider, messages []gains.Message, tools []gains.Tool) {
	// First call: model should request tool use
	resp, err := client.Chat(ctx, messages, gains.WithTools(tools))
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
	finalResp, err := client.Chat(ctx, messages, gains.WithTools(tools))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	fmt.Printf("Final response: %s\n", finalResp.Content)
	fmt.Printf("[Tokens: %d in, %d out]\n", finalResp.Usage.InputTokens, finalResp.Usage.OutputTokens)
}

func demoJSONMode(ctx context.Context, anthropicClient *anthropic.Client, openaiClient *openai.Client, googleClient *google.Client) {
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

	if anthropicClient != nil {
		fmt.Println("=== Anthropic (Claude) - Tool-based JSON ===")
		demoJSONWithProvider(ctx, anthropicClient, messages, schema)
	}

	if openaiClient != nil {
		fmt.Println("\n=== OpenAI (GPT-4) - Native JSON Schema ===")
		demoJSONWithProvider(ctx, openaiClient, messages, schema)
	}

	if googleClient != nil {
		fmt.Println("\n=== Google (Gemini) - Native JSON ===")
		demoJSONWithProvider(ctx, googleClient, messages, schema)
	}
}

func demoJSONWithProvider(ctx context.Context, client gains.ChatProvider, messages []gains.Message, schema gains.ResponseSchema) {
	resp, err := client.Chat(ctx, messages, gains.WithResponseSchema(schema))
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
