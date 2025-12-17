package main

import (
	"context"
	"fmt"
	"os"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/client"
	"github.com/spetersoncode/gains/event"
)

func demoChat(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│              Chat Demo                  │")
	fmt.Println("└─────────────────────────────────────────┘")

	messages := []ai.Message{
		{Role: ai.RoleUser, Content: "What is the capital of France? Reply in one sentence."},
	}

	fmt.Printf("\nUser: %s\n", messages[0].Content)
	fmt.Print("\nAssistant: ")

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

	fmt.Print("\nAssistant:\n")
	for ev := range stream {
		switch ev.Type {
		case event.MessageDelta:
			fmt.Print(ev.Delta)
		case event.MessageEnd:
			if ev.Response != nil {
				fmt.Printf("\n[Tokens: %d in, %d out]\n",
					ev.Response.Usage.InputTokens,
					ev.Response.Usage.OutputTokens)
			}
		case event.RunError:
			fmt.Fprintf(os.Stderr, "Stream error: %v\n", ev.Error)
			return
		}
	}
}
