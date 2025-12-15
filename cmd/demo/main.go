package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spetersoncode/gains/client"
)

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

	// Demo: Chat
	if askYesNo("Demo chat?") {
		demoChat(ctx, c)
	}

	// Demo: Chat Stream
	if askYesNo("Demo chat stream?") {
		demoChatStream(ctx, c)
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

	// Demo: Agent
	if askYesNo("Demo agent?") {
		demoAgent(ctx, c)
	}

	// Demo: Agent Stream
	if askYesNo("Demo agent stream?") {
		demoAgentStream(ctx, c)
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
