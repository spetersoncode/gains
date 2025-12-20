package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/model"
)

var reader = bufio.NewReader(os.Stdin)

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
			{model.Gemini3Pro, "Gemini 3.0 Pro"},
			{model.Gemini3DeepThink, "Gemini 3.0 Deep Think (reasoning)"},
		}
	case "vertex":
		return []modelOption{
			{model.VertexGemini25Flash, "Gemini 2.5 Flash (recommended)"},
			{model.VertexGemini25Pro, "Gemini 2.5 Pro (most capable)"},
			{model.VertexGemini25FlashLite, "Gemini 2.5 Flash Lite (fastest)"},
			{model.VertexGemini3Pro, "Gemini 3.0 Pro"},
			{model.VertexGemini3DeepThink, "Gemini 3.0 Deep Think (reasoning)"},
		}
	default:
		return nil
	}
}
