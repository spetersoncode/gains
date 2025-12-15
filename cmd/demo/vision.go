package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/client"
)

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
