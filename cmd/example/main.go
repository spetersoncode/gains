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
	fmt.Println("=== Anthropic ===")
	testProvider(anthropic.New(), ctx, prompt)

	// OpenAI
	fmt.Println("\n=== OpenAI ===")
	testProvider(openai.New(), ctx, prompt)

	// Google
	fmt.Println("\n=== Google ===")
	googleClient, err := google.New(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Google client error: %v\n", err)
	} else {
		testProvider(googleClient, ctx, prompt)
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
