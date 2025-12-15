package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/client"
)

func demoJSONMode(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│      JSON Mode / Structured Output      │")
	fmt.Println("└─────────────────────────────────────────┘")

	// Define a schema for structured output
	schema := gains.ResponseSchema{
		Name:        "book_info",
		Description: "Information about a book",
		Schema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"title": {
					"type": "string",
					"description": "The book title"
				},
				"author": {
					"type": "string",
					"description": "The author's name"
				},
				"year": {
					"type": "integer",
					"description": "Publication year"
				},
				"genres": {
					"type": "array",
					"items": {"type": "string"},
					"description": "List of genres"
				}
			},
			"required": ["title", "author", "year", "genres"]
		}`),
	}

	messages := []gains.Message{
		{Role: gains.RoleUser, Content: "Give me information about the book '1984' by George Orwell."},
	}

	fmt.Println("User: Give me information about the book '1984' by George Orwell.")
	fmt.Println("Schema: book_info (title, author, year, genres)")
	fmt.Println()

	resp, err := c.Chat(ctx, messages, gains.WithResponseSchema(schema))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	fmt.Println("Raw JSON response:")
	fmt.Println(resp.Content)

	// Parse and display structured data
	var book struct {
		Title  string   `json:"title"`
		Author string   `json:"author"`
		Year   int      `json:"year"`
		Genres []string `json:"genres"`
	}
	if err := json.Unmarshal([]byte(resp.Content), &book); err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		return
	}

	fmt.Println("\nParsed data:")
	fmt.Printf("  Title:  %s\n", book.Title)
	fmt.Printf("  Author: %s\n", book.Author)
	fmt.Printf("  Year:   %d\n", book.Year)
	fmt.Printf("  Genres: %v\n", book.Genres)
	fmt.Printf("[Tokens: %d in, %d out]\n", resp.Usage.InputTokens, resp.Usage.OutputTokens)
}
