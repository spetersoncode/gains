package main

import (
	"context"
	"fmt"
	"os"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/client"
)

// MovieInfo defines structured output for movie information.
type MovieInfo struct {
	Title    string   `json:"title" desc:"The movie title" required:"true"`
	Director string   `json:"director" desc:"The director's name" required:"true"`
	Year     int      `json:"year" desc:"Release year" required:"true"`
	Genre    string   `json:"genre" desc:"Primary genre" required:"true"`
	Rating   float64  `json:"rating" desc:"Rating out of 10" min:"0" max:"10"`
	Cast     []string `json:"cast" desc:"Main cast members" required:"true"`
}

func demoChatTyped(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│         ChatTyped Helper Demo           │")
	fmt.Println("└─────────────────────────────────────────┘")

	messages := []ai.Message{
		{Role: ai.RoleUser, Content: "Give me information about the movie 'The Matrix' (1999)."},
	}

	fmt.Println("User: Give me information about the movie 'The Matrix' (1999).")
	fmt.Println()
	fmt.Println("Using client.ChatTyped[MovieInfo]() - schema auto-generated from struct")
	fmt.Println()

	// ChatTyped auto-generates the schema and unmarshals the response
	movie, err := client.ChatTyped[MovieInfo](ctx, c, messages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	fmt.Println("Result (auto-unmarshaled):")
	fmt.Printf("  Title:    %s\n", movie.Title)
	fmt.Printf("  Director: %s\n", movie.Director)
	fmt.Printf("  Year:     %d\n", movie.Year)
	fmt.Printf("  Genre:    %s\n", movie.Genre)
	fmt.Printf("  Rating:   %.1f/10\n", movie.Rating)
	fmt.Printf("  Cast:     %v\n", movie.Cast)
}
