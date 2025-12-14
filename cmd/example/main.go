package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/provider/anthropic"
)

func main() {
	godotenv.Load() // loads .env from current directory

	client := anthropic.New()

	fmt.Println("=== Streaming ===")
	stream, err := client.ChatStream(context.Background(), []gains.Message{
		{Role: gains.RoleUser, Content: "Say hello in 3 different languages, one per line."},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	for event := range stream {
		if event.Err != nil {
			fmt.Fprintf(os.Stderr, "Stream error: %v\n", event.Err)
			os.Exit(1)
		}
		fmt.Print(event.Delta)
		if event.Done {
			fmt.Printf("\n\n[Tokens: %d in, %d out]\n",
				event.Response.Usage.InputTokens,
				event.Response.Usage.OutputTokens)
		}
	}
}
