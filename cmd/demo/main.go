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

	// Collect available credentials
	creds := client.Credentials{
		Anthropic: os.Getenv("ANTHROPIC_API_KEY"),
		OpenAI:    os.Getenv("OPENAI_API_KEY"),
		Google:    os.Getenv("GOOGLE_API_KEY"),
		Vertex: client.VertexConfig{
			Project:  os.Getenv("VERTEX_PROJECT"),
			Location: os.Getenv("VERTEX_LOCATION"),
		},
	}

	// Check what's available
	var available []struct {
		name  string
		label string
	}

	fmt.Println("Available providers:")
	if creds.Anthropic != "" {
		fmt.Printf("  [%d] Anthropic (Claude)\n", len(available)+1)
		available = append(available, struct {
			name  string
			label string
		}{"anthropic", "Anthropic (Claude)"})
	}
	if creds.OpenAI != "" {
		fmt.Printf("  [%d] OpenAI (GPT)\n", len(available)+1)
		available = append(available, struct {
			name  string
			label string
		}{"openai", "OpenAI (GPT)"})
	}
	if creds.Google != "" {
		fmt.Printf("  [%d] Google (Gemini)\n", len(available)+1)
		available = append(available, struct {
			name  string
			label string
		}{"google", "Google (Gemini)"})
	}
	if creds.Vertex.Project != "" && creds.Vertex.Location != "" {
		fmt.Printf("  [%d] Vertex AI (Gemini)\n", len(available)+1)
		available = append(available, struct {
			name  string
			label string
		}{"vertex", "Vertex AI (Gemini)"})
	}

	if len(available) == 0 {
		fmt.Println("  No credentials found. Set ANTHROPIC_API_KEY, OPENAI_API_KEY, GOOGLE_API_KEY,")
		fmt.Println("  or VERTEX_PROJECT + VERTEX_LOCATION.")
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

	// Get available models for the selected provider
	providerName := available[selected].name
	models := getModelsForProvider(providerName)

	// Let user select model
	var defaultChatModel ai.Model
	if len(models) == 1 {
		defaultChatModel = models[0].model
		fmt.Printf("Using model: %s\n", models[0].label)
	} else {
		fmt.Println("Available models:")
		for i, m := range models {
			fmt.Printf("  [%d] %s\n", i+1, m.label)
		}
		fmt.Printf("Select model [1-%d]: ", len(models))
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)
		var modelIdx int
		fmt.Sscanf(answer, "%d", &modelIdx)
		modelIdx-- // Convert to 0-indexed
		if modelIdx < 0 || modelIdx >= len(models) {
			modelIdx = 0
		}
		defaultChatModel = models[modelIdx].model
		fmt.Printf("Using model: %s\n", models[modelIdx].label)
	}

	// Create unified client with all available credentials
	c := client.New(client.Config{
		Credentials: creds,
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

	// Show menu and run selected demos
	for {
		selected := showMenu(c)
		if selected == nil {
			fmt.Println("\n✨ Goodbye!")
			return
		}

		runDemos(ctx, c, selected)

		fmt.Println("\n✨ Demo complete!")
		fmt.Println()
	}
}
