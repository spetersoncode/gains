package main

import (
	"context"
	"fmt"
	"os"
	"time"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/client"
	"github.com/spetersoncode/gains/schema"
	"github.com/spetersoncode/gains/workflow"
)

// SentimentAnalysis represents structured output from sentiment analysis.
type SentimentAnalysis struct {
	Sentiment  string   `json:"sentiment"`
	Confidence float64  `json:"confidence"`
	Keywords   []string `json:"keywords"`
	Summary    string   `json:"summary"`
}

// ContentSuggestions represents structured output for content improvements.
type ContentSuggestions struct {
	Tone        string   `json:"tone"`
	Suggestions []string `json:"suggestions"`
	Rewrite     string   `json:"rewrite"`
}

func demoTypedWorkflow(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│       Typed Workflow Demo               │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Println("\nThis demo shows TypedPromptStep with automatic unmarshaling:")
	fmt.Println("  1. Analyze text sentiment (returns *SentimentAnalysis)")
	fmt.Println("  2. Generate content suggestions (returns *ContentSuggestions)")
	fmt.Println("  3. Access results with type-safe GetTyped[T]")

	// Define schemas using the schema package
	sentimentSchema := ai.ResponseSchema{
		Name:        "sentiment_analysis",
		Description: "Sentiment analysis result",
		Schema: schema.Object().
			Field("sentiment", schema.String().
				Desc("The overall sentiment: positive, negative, or neutral").
				Enum("positive", "negative", "neutral").
				Required()).
			Field("confidence", schema.Number().
				Desc("Confidence score from 0.0 to 1.0").
				Min(0).Max(1).
				Required()).
			Field("keywords", schema.Array(schema.String()).
				Desc("Key words or phrases that influenced the analysis").
				Required()).
			Field("summary", schema.String().
				Desc("Brief explanation of the sentiment").
				Required()).
			MustBuild(),
	}

	suggestionsSchema := ai.ResponseSchema{
		Name:        "content_suggestions",
		Description: "Content improvement suggestions",
		Schema: schema.Object().
			Field("tone", schema.String().
				Desc("Suggested tone: professional, casual, enthusiastic, or empathetic").
				Enum("professional", "casual", "enthusiastic", "empathetic").
				Required()).
			Field("suggestions", schema.Array(schema.String()).
				Desc("List of specific improvement suggestions").
				Required()).
			Field("rewrite", schema.String().
				Desc("Rewritten version of the text with improvements applied").
				Required()).
			MustBuild(),
	}

	// Step 1: Typed sentiment analysis
	analyzeStep := workflow.NewTypedPromptStep[SentimentAnalysis](
		"analyze",
		c,
		func(s *workflow.State) []ai.Message {
			text := s.GetString("input_text")
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf(
					"Analyze the sentiment of this text:\n\n%s", text)},
			}
		},
		sentimentSchema,
		"analysis",
	)

	// Step 2: Generate suggestions based on analysis
	suggestStep := workflow.NewTypedPromptStep[ContentSuggestions](
		"suggest",
		c,
		func(s *workflow.State) []ai.Message {
			text := s.GetString("input_text")
			// Use type-safe accessor to get the analysis
			analysis, ok := workflow.GetTyped[*SentimentAnalysis](s, "analysis")
			if !ok {
				return []ai.Message{
					{Role: ai.RoleUser, Content: "No analysis available"},
				}
			}
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf(
					`Given this text:

%s

And this sentiment analysis:
- Sentiment: %s (confidence: %.0f%%)
- Keywords: %v
- Summary: %s

Suggest improvements to make this text more engaging and positive.`,
					text, analysis.Sentiment, analysis.Confidence*100,
					analysis.Keywords, analysis.Summary)},
			}
		},
		suggestionsSchema,
		"suggestions",
	)

	// Create chain workflow
	chain := workflow.NewChain("content-pipeline", analyzeStep, suggestStep)
	wf := workflow.New("typed-workflow", chain)

	// Sample text to analyze
	inputText := "I've been waiting for two weeks and still haven't received my order. " +
		"The tracking says it's stuck somewhere. This is really frustrating " +
		"and I'm considering canceling my subscription."

	fmt.Printf("\n--- Input Text ---\n%s\n", inputText)

	// Run workflow
	fmt.Println("\n--- Executing Typed Workflow ---")
	state := workflow.NewStateFrom(map[string]any{"input_text": inputText})
	events := wf.RunStream(ctx, state, workflow.WithTimeout(2*time.Minute))

	for event := range events {
		switch event.Type {
		case workflow.EventStepStart:
			fmt.Printf("\n[%s] Processing...\n", event.StepName)
		case workflow.EventStreamDelta:
			// Don't print deltas for JSON responses (they're partial JSON)
		case workflow.EventStepComplete:
			fmt.Printf("[%s] Complete\n", event.StepName)
		case workflow.EventError:
			fmt.Fprintf(os.Stderr, "\nError: %v\n", event.Error)
			return
		}
	}

	// Access results with type-safe accessors
	fmt.Println("\n--- Results (Type-Safe Access) ---")

	// Using GetTyped - returns (value, ok)
	analysis, ok := workflow.GetTyped[*SentimentAnalysis](state, "analysis")
	if ok {
		fmt.Println("\nSentiment Analysis:")
		fmt.Printf("  Sentiment:  %s\n", analysis.Sentiment)
		fmt.Printf("  Confidence: %.0f%%\n", analysis.Confidence*100)
		fmt.Printf("  Keywords:   %v\n", analysis.Keywords)
		fmt.Printf("  Summary:    %s\n", analysis.Summary)
	}

	// Using MustGet - panics if not found (use when you're certain it exists)
	suggestions := workflow.MustGet[*ContentSuggestions](state, "suggestions")
	fmt.Println("\nContent Suggestions:")
	fmt.Printf("  Recommended Tone: %s\n", suggestions.Tone)
	fmt.Println("  Suggestions:")
	for i, s := range suggestions.Suggestions {
		fmt.Printf("    %d. %s\n", i+1, s)
	}
	fmt.Printf("\n  Rewritten:\n  %s\n", suggestions.Rewrite)

	// Demonstrate GetTypedOr with default value
	missingData := workflow.GetTypedOr(state, "nonexistent", &SentimentAnalysis{
		Sentiment: "unknown",
	})
	fmt.Printf("\n  GetTypedOr fallback demo: sentiment = %q\n", missingData.Sentiment)
}
