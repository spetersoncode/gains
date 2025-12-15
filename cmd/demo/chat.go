package main

import (
	"context"
	"fmt"
	"os"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/client"
)

func demoChat(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│              Chat Demo                  │")
	fmt.Println("└─────────────────────────────────────────┘")

	messages := []ai.Message{
		{Role: ai.RoleUser, Content: "What is the capital of France? Reply in one sentence."},
	}

	fmt.Printf("\nUser: %s\n", messages[0].Content)
	fmt.Printf("\n%s: ", c.Provider())

	resp, err := c.Chat(ctx, messages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	fmt.Println(resp.Content)
	fmt.Printf("[Tokens: %d in, %d out]\n", resp.Usage.InputTokens, resp.Usage.OutputTokens)
}

func demoChatStream(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│          Chat Stream Demo               │")
	fmt.Println("└─────────────────────────────────────────┘")

	messages := []ai.Message{
		{Role: ai.RoleUser, Content: "Say hello in 3 different languages, one per line."},
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
