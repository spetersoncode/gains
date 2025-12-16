package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spetersoncode/gains/client"
)

// Category groups related demos together.
type Category string

const (
	CategoryChat     Category = "Chat"
	CategoryTools    Category = "Tools & Agents"
	CategoryOutput   Category = "Structured Output"
	CategoryWorkflow Category = "Workflows"
)

// categoryOrder defines the display order of categories.
var categoryOrder = []Category{
	CategoryChat,
	CategoryTools,
	CategoryOutput,
	CategoryWorkflow,
}

// Demo represents a single demo with its metadata.
type Demo struct {
	Name        string
	Description string
	Category    Category
	Feature     client.Feature // Required feature ("" = none)
	Run         func(ctx context.Context, c *client.Client)
}

// demos is the registry of all available demos.
var demos = []Demo{
	// Chat
	{Name: "chat", Description: "Basic chat with token counting", Category: CategoryChat, Run: demoChat},
	{Name: "stream", Description: "Streaming chat responses", Category: CategoryChat, Run: demoChatStream},
	{Name: "vision", Description: "Vision/image input analysis", Category: CategoryChat, Run: demoVisionInput},
	{Name: "image-gen", Description: "Image generation", Category: CategoryChat, Feature: client.FeatureImage, Run: demoImageGeneration},

	// Tools & Agents
	{Name: "tools", Description: "Tool/function calling", Category: CategoryTools, Run: demoToolCalling},
	{Name: "agent", Description: "Agent with tool execution", Category: CategoryTools, Run: demoAgent},
	{Name: "agent-stream", Description: "Agent with streaming events", Category: CategoryTools, Run: demoAgentStream},
	{Name: "research", Description: "Research agent with approval workflow", Category: CategoryTools, Run: demoAgentResearch},
	{Name: "combat", Description: "RPG combat with dice rolling", Category: CategoryTools, Run: demoAgentCombat},

	// Structured Output
	{Name: "json", Description: "JSON mode / structured output", Category: CategoryOutput, Run: demoJSONMode},
	{Name: "embeddings", Description: "Text embeddings & similarity", Category: CategoryOutput, Feature: client.FeatureEmbedding, Run: demoEmbeddings},

	// Workflows
	{Name: "typed", Description: "Typed workflow with Key[T]", Category: CategoryWorkflow, Run: demoTypedWorkflow},
	{Name: "chain", Description: "Sequential chain workflow", Category: CategoryWorkflow, Run: demoWorkflowChain},
	{Name: "parallel", Description: "Parallel execution workflow", Category: CategoryWorkflow, Run: demoWorkflowParallel},
	{Name: "router", Description: "Conditional router workflow", Category: CategoryWorkflow, Run: demoWorkflowRouter},
	{Name: "classifier", Description: "LLM classifier router", Category: CategoryWorkflow, Run: demoWorkflowClassifier},
	{Name: "loop", Description: "Iterative loop workflow", Category: CategoryWorkflow, Run: demoWorkflowLoop},
	{Name: "tool-step", Description: "Direct tool execution in workflow", Category: CategoryWorkflow, Run: demoWorkflowToolStep},
	{Name: "agent-step", Description: "Autonomous agent in workflow", Category: CategoryWorkflow, Run: demoWorkflowAgentStep},
}

// availableDemos returns demos filtered by client capabilities.
func availableDemos(c *client.Client) []Demo {
	var result []Demo
	for _, d := range demos {
		if d.Feature == "" || c.SupportsFeature(d.Feature) {
			result = append(result, d)
		}
	}
	return result
}

// showMenu displays the numbered menu with category headers and returns user selection.
// Returns indices of selected demos, or nil if user quits.
func showMenu(c *client.Client) []int {
	available := availableDemos(c)

	// Build category -> demos mapping preserving order
	byCategory := make(map[Category][]int)
	for i, d := range available {
		byCategory[d.Category] = append(byCategory[d.Category], i)
	}

	// Display menu
	fmt.Println("┌────────────────────────────────────────┐")
	fmt.Println("│             Select Demos               │")
	fmt.Println("└────────────────────────────────────────┘")
	fmt.Println()

	for _, cat := range categoryOrder {
		indices, ok := byCategory[cat]
		if !ok || len(indices) == 0 {
			continue
		}

		fmt.Printf("─── %s ───\n", cat)
		for _, i := range indices {
			d := available[i]
			fmt.Printf("  [%2d] %-14s %s\n", i+1, d.Name, d.Description)
		}
		fmt.Println()
	}

	fmt.Println("─── Options ───")
	fmt.Println("  [a]  Run all demos")
	fmt.Println("  [q]  Quit")
	fmt.Println()

	return promptSelection(len(available))
}

// promptSelection handles user input and returns selected demo indices.
func promptSelection(total int) []int {
	for {
		fmt.Print("Enter selection (number, range like 1-3, comma-separated, 'a' for all, 'q' to quit): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "q" || input == "quit" {
			return nil
		}

		if input == "a" || input == "all" {
			result := make([]int, total)
			for i := range result {
				result[i] = i
			}
			return result
		}

		// Parse selection
		selected, err := parseSelection(input, total)
		if err != nil {
			fmt.Printf("Invalid selection: %v\n", err)
			continue
		}

		if len(selected) == 0 {
			fmt.Println("No demos selected. Try again.")
			continue
		}

		return selected
	}
}

// parseSelection parses user input like "1", "1-3", "1,3,5", or "1-3,7".
func parseSelection(input string, total int) ([]int, error) {
	seen := make(map[int]bool)
	var result []int

	parts := strings.Split(input, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check for range
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range: %s", part)
			}

			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid number: %s", rangeParts[0])
			}

			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid number: %s", rangeParts[1])
			}

			if start > end {
				start, end = end, start
			}

			for i := start; i <= end; i++ {
				if i < 1 || i > total {
					return nil, fmt.Errorf("number out of range: %d (must be 1-%d)", i, total)
				}
				idx := i - 1
				if !seen[idx] {
					seen[idx] = true
					result = append(result, idx)
				}
			}
		} else {
			// Single number
			n, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid number: %s", part)
			}

			if n < 1 || n > total {
				return nil, fmt.Errorf("number out of range: %d (must be 1-%d)", n, total)
			}

			idx := n - 1
			if !seen[idx] {
				seen[idx] = true
				result = append(result, idx)
			}
		}
	}

	return result, nil
}

// runDemos executes the selected demos.
func runDemos(ctx context.Context, c *client.Client, indices []int) {
	available := availableDemos(c)

	for i, idx := range indices {
		d := available[idx]

		fmt.Println()
		fmt.Printf("━━━ [%d/%d] %s: %s ━━━\n", i+1, len(indices), d.Name, d.Description)
		fmt.Println()

		d.Run(ctx, c)

		// Pause between demos if more than one
		if i < len(indices)-1 {
			fmt.Println()
			fmt.Print("Press Enter to continue...")
			reader.ReadString('\n')
		}
	}
}
