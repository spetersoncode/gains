package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/client"
	"github.com/spetersoncode/gains/model"
)

func main() {
	godotenv.Load()
	ctx := context.Background()

	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║       gains - AI Library Demo          ║")
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Println()

	// Collect available API keys
	apiKeys := client.APIKeys{
		Anthropic: os.Getenv("ANTHROPIC_API_KEY"),
		OpenAI:    os.Getenv("OPENAI_API_KEY"),
		Google:    os.Getenv("GOOGLE_API_KEY"),
	}

	// Check what's available
	var available []struct {
		name  string
		label string
	}

	fmt.Println("Available providers:")
	if apiKeys.Anthropic != "" {
		fmt.Printf("  [%d] Anthropic (Claude)\n", len(available)+1)
		available = append(available, struct {
			name  string
			label string
		}{"anthropic", "Anthropic (Claude)"})
	}
	if apiKeys.OpenAI != "" {
		fmt.Printf("  [%d] OpenAI (GPT)\n", len(available)+1)
		available = append(available, struct {
			name  string
			label string
		}{"openai", "OpenAI (GPT)"})
	}
	if apiKeys.Google != "" {
		fmt.Printf("  [%d] Google (Gemini)\n", len(available)+1)
		available = append(available, struct {
			name  string
			label string
		}{"google", "Google (Gemini)"})
	}

	if len(available) == 0 {
		fmt.Println("  No API keys found. Set ANTHROPIC_API_KEY, OPENAI_API_KEY, or GOOGLE_API_KEY.")
		return
	}
	fmt.Println()

	// Let user select default provider for chat
	var selected int
	if len(available) == 1 {
		selected = 0
		fmt.Printf("Using %s for chat (only available provider)\n", available[0].label)
	} else {
		fmt.Printf("Select default chat provider [1-%d]: ", len(available))
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)
		fmt.Sscanf(answer, "%d", &selected)
		selected-- // Convert to 0-indexed
		if selected < 0 || selected >= len(available) {
			selected = 0
		}
		fmt.Printf("Using %s for chat\n", available[selected].label)
	}
	fmt.Println()

	// Determine default chat model based on selection
	var defaultChatModel ai.Model
	switch available[selected].name {
	case "anthropic":
		defaultChatModel = model.ClaudeSonnet45
	case "openai":
		defaultChatModel = model.GPT52
	case "google":
		defaultChatModel = model.Gemini25Flash
	}

	// Create unified client with all available API keys
	c := client.New(client.Config{
		APIKeys: apiKeys,
		Defaults: client.Defaults{
			Chat:      defaultChatModel,
			Embedding: model.TextEmbedding3Small,
			Image:     model.GPTImage1,
		},
	})

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

	// Demo: Workflow Chain
	if askYesNo("Demo workflow chain?") {
		demoWorkflowChain(ctx, c)
	}

	// Demo: Workflow Parallel
	if askYesNo("Demo workflow parallel?") {
		demoWorkflowParallel(ctx, c)
	}

	// Demo: Workflow Router
	if askYesNo("Demo workflow router?") {
		demoWorkflowRouter(ctx, c)
	}

	// Demo: Workflow Classifier
	if askYesNo("Demo workflow classifier?") {
		demoWorkflowClassifier(ctx, c)
	}

	fmt.Println("\n✨ Demo complete!")
}
