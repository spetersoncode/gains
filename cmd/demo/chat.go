package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/client"
)

func demoChatStreaming(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│          Chat Streaming Demo            │")
	fmt.Println("└─────────────────────────────────────────┘")

	messages := []gains.Message{
		{Role: gains.RoleUser, Content: "Say hello in 3 different languages, one per line."},
	}

	stream, err := c.ChatStream(ctx, messages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	fmt.Printf("\n%s:\n", c.Provider())
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
