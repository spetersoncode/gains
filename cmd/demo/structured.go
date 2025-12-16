package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/client"
	"github.com/spetersoncode/gains/schema"
)

func demoJSONMode(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│      JSON Mode / Structured Output      │")
	fmt.Println("└─────────────────────────────────────────┘")

	// Define a schema for structured output using the schema builder
	responseSchema := ai.ResponseSchema{
		Name:        "book_info",
		Description: "Information about a book",
		Schema: schema.Object().
			Field("title", schema.String().Desc("The book title").Required()).
			Field("author", schema.String().Desc("The author's name").Required()).
			Field("year", schema.Int().Desc("Publication year").Required()).
			Field("genres", schema.Array(schema.String()).Desc("List of genres").Required()).
			MustBuild(),
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
