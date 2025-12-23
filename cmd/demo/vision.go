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

func demoChatImageGeneration(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│     Chat-Based Image Generation Demo    │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println("This demo uses Gemini native image generation via Chat API.")
	fmt.Println("Unlike Imagen, this supports multimodal conversations with images.")
	fmt.Println()

	// Select chat image model
	selectedModel := selectModel(getChatImageModels(), "Select chat image model:")
	if selectedModel == nil {
		fmt.Println("No chat image models available. Need Google or Vertex AI credentials.")
		return
	}

	prompt := "Generate an image of a friendly robot serving coffee in a cozy cafe"
	fmt.Printf("Prompt: %q\n\n", prompt)

	messages := []ai.Message{
		{Role: ai.RoleUser, Content: prompt},
	}

	resp, err := c.Chat(ctx, messages,
		ai.WithModel(selectedModel),
		ai.WithImageOutput(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	fmt.Printf("Response text: %s\n", resp.Content)
	fmt.Printf("[Tokens: %d in, %d out]\n\n", resp.Usage.InputTokens, resp.Usage.OutputTokens)

	// Extract and save images from response parts
	imageCount := 0
	for _, part := range resp.Parts {
		if part.Type == ai.ContentPartTypeImage && part.Base64 != "" {
			imageCount++
			ext := "png"
			if part.MimeType == "image/jpeg" {
				ext = "jpg"
			}
			filename := fmt.Sprintf("chat_image_%s_%d.%s", time.Now().Format("20060102_150405"), imageCount, ext)
			if err := saveBase64Image(part.Base64, filename); err != nil {
				fmt.Printf("Image %d: Failed to save (%v)\n", imageCount, err)
			} else {
				absPath, _ := filepath.Abs(filename)
				fmt.Printf("Image %d: Saved to %s\n", imageCount, absPath)
			}
		}
	}

	if imageCount == 0 {
		fmt.Println("No images generated in response.")
	}
}

func demoChatImagePortrait(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│    Portrait (9:16) Image Generation     │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println("This demo generates a 9:16 portrait image using Gemini.")
	fmt.Println("Uses WithImageAspectRatio for native aspect ratio control.")
	fmt.Println()

	// Select chat image model
	selectedModel := selectModel(getChatImageModels(), "Select chat image model:")
	if selectedModel == nil {
		fmt.Println("No chat image models available. Need Google or Vertex AI credentials.")
		return
	}

	prompt := "Generate an image of a magical tower reaching into the clouds with glowing windows and birds flying around it"
	fmt.Printf("Prompt: %q\n", prompt)
	fmt.Printf("Aspect Ratio: 9:16\n\n")

	messages := []ai.Message{
		{Role: ai.RoleUser, Content: prompt},
	}

	resp, err := c.Chat(ctx, messages,
		ai.WithModel(selectedModel),
		ai.WithImageOutput(),
		ai.WithImageAspectRatio(ai.ImageAspectRatio9x16),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	fmt.Printf("Response text: %s\n", resp.Content)
	fmt.Printf("[Tokens: %d in, %d out]\n\n", resp.Usage.InputTokens, resp.Usage.OutputTokens)

	// Extract and save images from response parts
	imageCount := 0
	for _, part := range resp.Parts {
		if part.Type == ai.ContentPartTypeImage && part.Base64 != "" {
			imageCount++
			ext := "png"
			if part.MimeType == "image/jpeg" {
				ext = "jpg"
			}
			filename := fmt.Sprintf("chat_image_9x16_%s_%d.%s", time.Now().Format("20060102_150405"), imageCount, ext)
			if err := saveBase64Image(part.Base64, filename); err != nil {
				fmt.Printf("Image %d: Failed to save (%v)\n", imageCount, err)
			} else {
				absPath, _ := filepath.Abs(filename)
				fmt.Printf("Image %d: Saved to %s\n", imageCount, absPath)
			}
		}
	}

	if imageCount == 0 {
		fmt.Println("No images generated in response.")
	}
}
