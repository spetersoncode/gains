package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/client"
)

// BookInfo defines structured output for book information via struct tags.
type BookInfo struct {
	Title  string   `json:"title" desc:"The book title" required:"true"`
	Author string   `json:"author" desc:"The author's name" required:"true"`
	Year   int      `json:"year" desc:"Publication year" required:"true"`
	Genres []string `json:"genres" desc:"List of genres" required:"true"`
}

func demoJSONMode(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│      JSON Mode / Structured Output      │")
	fmt.Println("└─────────────────────────────────────────┘")

	// Define a schema for structured output using struct tags
	responseSchema := ai.ResponseSchema{
		Name:        "book_info",
		Description: "Information about a book",
		Schema:      ai.MustSchemaFor[BookInfo](),
	}

	messages := []ai.Message{
		{Role: ai.RoleUser, Content: "Give me information about the book '1984' by George Orwell."},
	}

	fmt.Println("User: Give me information about the book '1984' by George Orwell.")
	fmt.Println("Schema: book_info (title, author, year, genres)")
	fmt.Println()

	resp, err := c.Chat(ctx, messages, ai.WithResponseSchema(responseSchema))
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
