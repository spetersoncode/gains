package main

import (
	"context"
	"fmt"
	"os"
	"time"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/client"
	"github.com/spetersoncode/gains/event"
	"github.com/spetersoncode/gains/workflow"
)

// SentimentAnalysis represents structured output from sentiment analysis.
// Struct tags define the JSON schema for the LLM response.
type SentimentAnalysis struct {
	Sentiment  string   `json:"sentiment" desc:"The overall sentiment: positive, negative, or neutral" enum:"positive,negative,neutral" required:"true"`
	Confidence float64  `json:"confidence" desc:"Confidence score from 0.0 to 1.0" min:"0" max:"1" required:"true"`
	Keywords   []string `json:"keywords" desc:"Key words or phrases that influenced the analysis" required:"true"`
	Summary    string   `json:"summary" desc:"Brief explanation of the sentiment" required:"true"`
}

// ContentSuggestions represents structured output for content improvements.
// Struct tags define the JSON schema for the LLM response.
type ContentSuggestions struct {
	Tone        string   `json:"tone" desc:"Suggested tone: professional, casual, enthusiastic, or empathetic" enum:"professional,casual,enthusiastic,empathetic" required:"true"`
	Suggestions []string `json:"suggestions" desc:"List of specific improvement suggestions" required:"true"`
	Rewrite     string   `json:"rewrite" desc:"Rewritten version of the text with improvements applied" required:"true"`
}

// Define typed keys - SINGLE SOURCE OF TRUTH for state key-type mapping.
// This pattern provides compile-time type safety and IDE autocomplete.
var (
	KeyInputText   = workflow.StringKey("input_text")
	KeyAnalysis    = workflow.NewKey[*SentimentAnalysis]("analysis")
	KeySuggestions = workflow.NewKey[*ContentSuggestions]("suggestions")
)

func demoTypedWorkflow(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│       Typed Workflow Demo               │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Println("\nThis demo shows TypedPromptStep with type-safe Key[T] state access:")
	fmt.Println("  1. Analyze text sentiment (returns *SentimentAnalysis)")
	fmt.Println("  2. Generate content suggestions (returns *ContentSuggestions)")
	fmt.Println("  3. Access results with type-safe Get/MustGet using typed keys")

	// Define schemas using struct tags - the struct definitions above
	// include all the schema metadata via desc, enum, min, max, required tags
	sentimentSchema := ai.ResponseSchema{
		Name:        "sentiment_analysis",
		Description: "Sentiment analysis result",
		Schema:      ai.MustSchemaFor[SentimentAnalysis](),
	}

	suggestionsSchema := ai.ResponseSchema{
		Name:        "content_suggestions",
		Description: "Content improvement suggestions",
		Schema:      ai.MustSchemaFor[ContentSuggestions](),
	}

	// Step 1: Typed sentiment analysis using typed key
	analyzeStep := workflow.NewTypedPromptStepWithKey(
		"analyze",
		c,
		func(s *workflow.State) []ai.Message {
			text := workflow.MustGet(s, KeyInputText)
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf(
					"Analyze the sentiment of this text:\n\n%s", text)},
			}
		},
		sentimentSchema,
		KeyAnalysis,
	)

	// Step 2: Generate suggestions based on analysis
	suggestStep := workflow.NewTypedPromptStepWithKey(
		"suggest",
		c,
		func(s *workflow.State) []ai.Message {
			text := workflow.MustGet(s, KeyInputText)
			// Use type-safe accessor - type inferred from key
			analysis, ok := workflow.Get(s, KeyAnalysis)
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
		KeySuggestions,
	)

	// Create chain workflow
	chain := workflow.NewChain("content-pipeline", analyzeStep, suggestStep)
	wf := workflow.New("typed-workflow", chain)

	// Sample text to analyze
	inputText := "I've been waiting for two weeks and still haven't received my order. " +
		"The tracking says it's stuck somewhere. This is really frustrating " +
		"and I'm considering canceling my subscription."

	fmt.Printf("\n--- Input Text ---\n%s\n", inputText)

	// Run workflow - set state using typed key
	fmt.Println("\n--- Executing Typed Workflow ---")
	state := workflow.NewStateFrom(nil)
	workflow.Set(state, KeyInputText, inputText)

	events := wf.RunStream(ctx, state, workflow.WithTimeout(2*time.Minute))

	for ev := range events {
		switch ev.Type {
		case event.StepStart:
			fmt.Printf("\n[%s] Processing...\n", ev.StepName)
		case event.MessageDelta:
			// Don't print deltas for JSON responses (they're partial JSON)
		case event.StepEnd:
			fmt.Printf("[%s] Complete\n", ev.StepName)
		case event.RunError:
			fmt.Fprintf(os.Stderr, "\nError: %v\n", ev.Error)
			return
		}
	}

	// Access results with type-safe accessors
	fmt.Println("\n--- Results (Type-Safe Access with Key[T]) ---")

	// Using Get - returns (value, ok), type inferred from key
	analysis, ok := workflow.Get(state, KeyAnalysis)
	if ok {
		fmt.Println("\nSentiment Analysis:")
		fmt.Printf("  Sentiment:  %s\n", analysis.Sentiment)
		fmt.Printf("  Confidence: %.0f%%\n", analysis.Confidence*100)
		fmt.Printf("  Keywords:   %v\n", analysis.Keywords)
		fmt.Printf("  Summary:    %s\n", analysis.Summary)
	}

	// Using MustGet - panics if not found, type inferred from key
	suggestions := workflow.MustGet(state, KeySuggestions)
	fmt.Println("\nContent Suggestions:")
	fmt.Printf("  Recommended Tone: %s\n", suggestions.Tone)
	fmt.Println("  Suggestions:")
	for i, s := range suggestions.Suggestions {
		fmt.Printf("    %d. %s\n", i+1, s)
	}
	fmt.Printf("\n  Rewritten:\n  %s\n", suggestions.Rewrite)

	// Demonstrate GetOr with default value
	missingKey := workflow.NewKey[*SentimentAnalysis]("nonexistent")
	missingData := workflow.GetOr(state, missingKey, &SentimentAnalysis{
		Sentiment: "unknown",
	})
	fmt.Printf("\n  GetOr fallback demo: sentiment = %q\n", missingData.Sentiment)
}
