package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/client"
)

func demoVisionInput(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│         Vision/Image Input Demo         │")
	fmt.Println("└─────────────────────────────────────────┘")

	// Use a public domain image URL
	imageURL := "https://upload.wikimedia.org/wikipedia/commons/thumb/4/47/PNG_transparency_demonstration_1.png/300px-PNG_transparency_demonstration_1.png"

	messages := []ai.Message{
		{
			Role: ai.RoleUser,
			Parts: []ai.ContentPart{
				ai.NewTextPart("Describe this image in one sentence. What do you see?"),
				ai.NewImageURLPart(imageURL),
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

	// Select image model
	selectedModel := selectModel(getImageModels(), "Select image model:")

	prompt := "A serene mountain landscape at sunset with a calm lake reflection"
	fmt.Printf("Prompt: %q\n\n", prompt)

	resp, err := c.GenerateImage(ctx, prompt,
		ai.WithImageModel(selectedModel),
		ai.WithImageSize(ai.ImageSize1024x1024),
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
			// Save base64 image to file
			filename := fmt.Sprintf("generated_%s_%d.png", time.Now().Format("20060102_150405"), i+1)
			if err := saveBase64Image(img.Base64, filename); err != nil {
				fmt.Printf("  Base64: %d bytes (failed to save: %v)\n", len(img.Base64), err)
			} else {
				absPath, _ := filepath.Abs(filename)
				fmt.Printf("  Saved: %s\n", absPath)
			}
		}
		if img.RevisedPrompt != "" {
			fmt.Printf("  Revised prompt: %s\n", img.RevisedPrompt)
		}
	}
}

// saveBase64Image decodes and saves a base64-encoded image to a file.
func saveBase64Image(b64 string, filename string) error {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return fmt.Errorf("decode base64: %w", err)
	}
	return os.WriteFile(filename, data, 0644)
}
