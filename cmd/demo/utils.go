package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/client"
	"github.com/spetersoncode/gains/model"
)

var reader = bufio.NewReader(os.Stdin)

// availableCreds stores credentials for model filtering.
// Set by main() after loading environment.
var availableCreds client.Credentials

func askYesNo(question string) bool {
	fmt.Printf("%s [y/N]: ", question)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

type modelOption struct {
	model ai.Model
	label string
}

func getModelsForProvider(provider string) []modelOption {
	switch provider {
	case "anthropic":
		return []modelOption{
			{model.ClaudeSonnet45, "Claude Sonnet 4.5 (recommended)"},
			{model.ClaudeOpus45, "Claude Opus 4.5 (most capable)"},
			{model.ClaudeHaiku45, "Claude Haiku 4.5 (fastest)"},
		}
	case "openai":
		return []modelOption{
			{model.GPT52, "GPT-5.2 (recommended)"},
			{model.GPT52Pro, "GPT-5.2 Pro (most capable)"},
			{model.GPT51, "GPT-5.1"},
			{model.GPT51Mini, "GPT-5.1 Mini (fastest)"},
			{model.O3, "O3 (reasoning)"},
			{model.O3Mini, "O3 Mini (fast reasoning)"},
		}
	case "google":
		return []modelOption{
			{model.Gemini25Flash, "Gemini 2.5 Flash (recommended)"},
			{model.Gemini25Pro, "Gemini 2.5 Pro (most capable)"},
			{model.Gemini25FlashLite, "Gemini 2.5 Flash Lite (fastest)"},
			{model.Gemini3FlashPreview, "Gemini 3 Flash Preview"},
			{model.Gemini3Pro, "Gemini 3.0 Pro"},
			{model.Gemini3DeepThink, "Gemini 3.0 Deep Think (reasoning)"},
		}
	case "vertex":
		return []modelOption{
			{model.VertexGemini25Flash, "Gemini 2.5 Flash (recommended)"},
			{model.VertexGemini25Pro, "Gemini 2.5 Pro (most capable)"},
			{model.VertexGemini25FlashLite, "Gemini 2.5 Flash Lite (fastest)"},
			{model.VertexGemini3FlashPreview, "Gemini 3 Flash Preview"},
			{model.VertexGemini3Pro, "Gemini 3.0 Pro"},
			{model.VertexGemini3DeepThink, "Gemini 3.0 Deep Think (reasoning)"},
		}
	default:
		return nil
	}
}

func getEmbeddingModels() []modelOption {
	var models []modelOption

	if availableCreds.OpenAI != "" {
		models = append(models,
			modelOption{model.TextEmbedding3Small, "OpenAI text-embedding-3-small (recommended)"},
			modelOption{model.TextEmbedding3Large, "OpenAI text-embedding-3-large"},
		)
	}
	if availableCreds.Google != "" {
		models = append(models,
			modelOption{model.GeminiEmbedding001, "Google gemini-embedding-001"},
		)
	}
	if availableCreds.Vertex.Project != "" && availableCreds.Vertex.Location != "" {
		models = append(models,
			modelOption{model.VertexGeminiEmbedding001, "Vertex AI gemini-embedding-001"},
		)
	}

	return models
}

func getImageModels() []modelOption {
	var models []modelOption

	if availableCreds.OpenAI != "" {
		models = append(models,
			modelOption{model.GPTImage15, "OpenAI GPT Image 1.5 (recommended)"},
			modelOption{model.GPTImage1, "OpenAI GPT Image 1"},
			modelOption{model.GPTImage1Mini, "OpenAI GPT Image 1 Mini"},
		)
	}
	if availableCreds.Google != "" {
		models = append(models,
			modelOption{model.Imagen4, "Google Imagen 4 (recommended)"},
			modelOption{model.Imagen4Fast, "Google Imagen 4 Fast"},
			modelOption{model.Imagen4Ultra, "Google Imagen 4 Ultra"},
		)
	}
	if availableCreds.Vertex.Project != "" && availableCreds.Vertex.Location != "" {
		models = append(models,
			modelOption{model.VertexImagen4, "Vertex AI Imagen 4 (recommended)"},
			modelOption{model.VertexImagen4Fast, "Vertex AI Imagen 4 Fast"},
			modelOption{model.VertexImagen4Ultra, "Vertex AI Imagen 4 Ultra"},
		)
	}

	return models
}

func getChatImageModels() []modelOption {
	var models []modelOption

	if availableCreds.Google != "" {
		models = append(models,
			modelOption{model.Gemini3ProImagePreview, "Gemini 3 Pro Image Preview (recommended)"},
			modelOption{model.Gemini25FlashImage, "Gemini 2.5 Flash Image"},
		)
	}
	if availableCreds.Vertex.Project != "" && availableCreds.Vertex.Location != "" {
		models = append(models,
			modelOption{model.VertexGemini3ProImagePreview, "Vertex Gemini 3 Pro Image Preview"},
			modelOption{model.VertexGemini25FlashImage, "Vertex Gemini 2.5 Flash Image"},
		)
	}

	return models
}

func selectModel(models []modelOption, prompt string) ai.Model {
	if len(models) == 0 {
		return nil
	}
	if len(models) == 1 {
		fmt.Printf("Using: %s\n", models[0].label)
		return models[0].model
	}

	fmt.Println(prompt)
	for i, m := range models {
		fmt.Printf("  [%d] %s\n", i+1, m.label)
	}
	fmt.Printf("Select [1-%d]: ", len(models))
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(answer)
	var idx int
	fmt.Sscanf(answer, "%d", &idx)
	idx--
	if idx < 0 || idx >= len(models) {
		idx = 0
	}
	fmt.Printf("Using: %s\n\n", models[idx].label)
	return models[idx].model
}
