package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/agent"
	"github.com/spetersoncode/gains/client"
	"github.com/spetersoncode/gains/tool"
)

func demoAgentResearch(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│      Research Assistant Agent           │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println("This demo showcases an advanced agent with:")
	fmt.Println("  - File tools (read, write, list directory)")
	fmt.Println("  - HTTP tools for web requests")
	fmt.Println("  - Selective approval (write operations only)")
	fmt.Println("  - Parallel tool execution")
	fmt.Println("  - Streaming event handling")
	fmt.Println()

	// Create workspace directory for the demo
	workspacePath := filepath.Join(os.TempDir(), "gains-research-demo")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating workspace: %v\n", err)
		return
	}
	defer os.RemoveAll(workspacePath) // Cleanup after demo

	fmt.Printf("Workspace: %s\n", workspacePath)
	fmt.Println()

	// Create tool registry with file and HTTP tools
	registry := tool.NewRegistry()

	// Register file tools - restricted to workspace directory
	// This prevents the agent from accessing files outside the workspace
	tool.MustRegisterAll(registry, tool.FileTools(
		tool.WithBasePath(workspacePath),
		tool.WithMaxFileSize(1024*1024), // 1MB limit
	))

	// Register HTTP tools - restricted to specific hosts for safety
	// Only allows requests to GitHub-related domains
	tool.MustRegisterAll(registry, tool.WebTools(
		tool.WithAllowedHosts(
			"github.com",
			"api.github.com",
			"raw.githubusercontent.com",
		),
		tool.WithMaxResponseSize(512*1024), // 512KB limit
		tool.WithHTTPTimeout(30*time.Second),
	))

	fmt.Println("Registered tools:")
	for _, name := range registry.Names() {
		fmt.Printf("  - %s\n", name)
	}
	fmt.Println()

	// Define approval handler for write operations
	// This demonstrates human-in-the-loop approval workflow
	approver := func(ctx context.Context, call ai.ToolCall) (bool, string) {
		// Parse the tool arguments for display
		var args map[string]interface{}
		_ = json.Unmarshal([]byte(call.Arguments), &args)

		fmt.Println()
		fmt.Println("┌──────────────────────────────────────────┐")
		fmt.Println("│          APPROVAL REQUIRED               │")
		fmt.Println("└──────────────────────────────────────────┘")
		fmt.Printf("Tool: %s\n", call.Name)

		// Show relevant details based on tool
		switch call.Name {
		case "write_file":
			if path, ok := args["path"].(string); ok {
				fmt.Printf("Path: %s\n", path)
			}
			if content, ok := args["content"].(string); ok {
				preview := content
				if len(preview) > 300 {
					preview = preview[:300] + "\n... (truncated)"
				}
				fmt.Printf("Content preview:\n%s\n", preview)
			}
		case "edit_file":
			if path, ok := args["path"].(string); ok {
				fmt.Printf("Path: %s\n", path)
			}
			if mode, ok := args["mode"].(string); ok {
				fmt.Printf("Mode: %s\n", mode)
			}
		}
		fmt.Println()

		// Prompt user for approval
		if askYesNo("Approve this operation?") {
			return true, ""
		}
		return false, "User rejected the write operation"
	}

	// Create the agent
	a := agent.New(c, registry)

	// Define the research task prompt
	researchPrompt := `You are a research assistant. Your task is to:

1. First, use http_request to fetch the README from the Go repository:
   URL: https://raw.githubusercontent.com/golang/go/master/README.md
   Method: GET

2. Read the content and identify the key information about the Go project.

3. Use list_directory to see what's in the current workspace (use "." as path).

4. Create a research report file called "go-research-report.md" that summarizes:
   - What Go is (based on the README)
   - Key features or links mentioned
   - The official repository URL
   - Date of this research

5. After writing the report, read it back using read_file to confirm it was saved correctly.

Be thorough but concise. Start by fetching the README.`

	fmt.Println("Research Task:")
	fmt.Println("─────────────────────────────────────────")
	fmt.Println(researchPrompt)
	fmt.Println("─────────────────────────────────────────")
	fmt.Println()
	fmt.Print("Press Enter to start the research agent...")
	reader.ReadString('\n')

	fmt.Println("\n--- Agent Execution ---")

	// Track execution metrics
	toolCallCount := 0
	approvedCount := 0
	rejectedCount := 0

	// Run agent with streaming and all advanced options
	events := a.RunStream(ctx, []ai.Message{
		{Role: ai.RoleUser, Content: researchPrompt},
	},
		agent.WithMaxSteps(10),                                // Allow up to 10 iterations
		agent.WithTimeout(5*time.Minute),                      // Overall timeout
		agent.WithHandlerTimeout(60*time.Second),              // Per-tool timeout
		agent.WithParallelToolCalls(true),                     // Enable parallel execution
		agent.WithApprover(approver),                          // Human-in-the-loop
		agent.WithApprovalRequired("write_file", "edit_file"), // Only require approval for writes
	)

	// Process streaming events
	for event := range events {
		switch event.Type {
		case agent.EventStepStart:
			fmt.Printf("\n╔══════════════════════════════════════════╗\n")
			fmt.Printf("║  Step %d                                   ║\n", event.Step)
			fmt.Printf("╚══════════════════════════════════════════╝\n")

		case agent.EventStreamDelta:
			// Print streaming tokens as they arrive
			fmt.Print(event.Delta)

		case agent.EventToolCallRequested:
			toolCallCount++
			fmt.Printf("\n\n  -> Tool Requested: %s\n", event.ToolCall.Name)
			// Pretty-print arguments
			var prettyArgs map[string]interface{}
			if err := json.Unmarshal([]byte(event.ToolCall.Arguments), &prettyArgs); err == nil {
				argsJSON, _ := json.MarshalIndent(prettyArgs, "     ", "  ")
				fmt.Printf("     Arguments:\n     %s\n", string(argsJSON))
			}

		case agent.EventToolCallApproved:
			approvedCount++
			fmt.Printf("  -> Approved: %s\n", event.ToolCall.Name)

		case agent.EventToolCallRejected:
			rejectedCount++
			fmt.Printf("  -> Rejected: %s - %s\n", event.ToolCall.Name, event.Message)

		case agent.EventToolCallStarted:
			fmt.Printf("  -> Executing: %s\n", event.ToolCall.Name)

		case agent.EventToolResult:
			status := "Success"
			if event.ToolResult.IsError {
				status = "Error"
			}
			// Truncate long results for display
			content := truncateForDisplay(event.ToolResult.Content, 100)
			fmt.Printf("  <- Result [%s]: %s\n", status, content)

		case agent.EventStepComplete:
			if event.Response != nil {
				fmt.Printf("\n  [Tokens: %d in, %d out]\n",
					event.Response.Usage.InputTokens,
					event.Response.Usage.OutputTokens)
			}

		case agent.EventAgentComplete:
			fmt.Println("\n╔══════════════════════════════════════════╗")
			fmt.Println("║           AGENT COMPLETE                 ║")
			fmt.Println("╚══════════════════════════════════════════╝")
			fmt.Printf("Termination: %s\n", event.Message)
			fmt.Printf("Total Steps: %d\n", event.Step)
			fmt.Printf("Tool Calls: %d (approved: %d, rejected: %d)\n",
				toolCallCount, approvedCount, rejectedCount)

			if event.Response != nil && event.Response.Content != "" {
				fmt.Println("\n--- Final Response ---")
				fmt.Println(event.Response.Content)
			}

		case agent.EventError:
			fmt.Fprintf(os.Stderr, "\nError: %v\n", event.Error)
		}
	}

	// Show workspace contents after execution
	fmt.Println("\n--- Workspace Contents ---")
	entries, err := os.ReadDir(workspacePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading workspace: %v\n", err)
	} else if len(entries) == 0 {
		fmt.Println("(empty)")
	} else {
		for _, entry := range entries {
			info, _ := entry.Info()
			if info != nil {
				fmt.Printf("  %s (%d bytes)\n", entry.Name(), info.Size())
			}
		}
	}

	// If report was created, show it
	reportPath := filepath.Join(workspacePath, "go-research-report.md")
	if content, err := os.ReadFile(reportPath); err == nil {
		fmt.Println("\n--- Generated Report ---")
		fmt.Println(string(content))
	}
}

// truncateForDisplay truncates a string for console display, replacing newlines
func truncateForDisplay(s string, maxLen int) string {
	// Replace newlines with spaces for single-line display
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
