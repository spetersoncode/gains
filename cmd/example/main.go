package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/provider/anthropic"
	"github.com/spetersoncode/gains/provider/google"
	"github.com/spetersoncode/gains/provider/openai"
)

func main() {
	godotenv.Load()
	ctx := context.Background()

	prompt := []gains.Message{
		{Role: gains.RoleUser, Content: "Say hello in 3 different languages, one per line."},
	}

	// Anthropic
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		fmt.Println("=== Anthropic ===")
		testProvider(anthropic.New(key), ctx, prompt)
	}

	// OpenAI
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		fmt.Println("\n=== OpenAI ===")
		openaiClient := openai.New(key)
		testProvider(openaiClient, ctx, prompt)

		fmt.Println("\n=== OpenAI Image Generation ===")
		testImageProvider(openaiClient, ctx)
	}

	// Google
	if key := os.Getenv("GOOGLE_API_KEY"); key != "" {
		fmt.Println("\n=== Google ===")
		googleClient, err := google.New(ctx, key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Google client error: %v\n", err)
		} else {
			testProvider(googleClient, ctx, prompt)

			fmt.Println("\n=== Google Image Generation ===")
			testImageProvider(googleClient, ctx)
		}
	}
}

func testProvider(client gains.ChatProvider, ctx context.Context, messages []gains.Message) {
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

func testImageProvider(client gains.ImageProvider, ctx context.Context) {
	resp, err := client.GenerateImage(ctx, "A serene mountain landscape at sunset with a calm lake reflection",
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
