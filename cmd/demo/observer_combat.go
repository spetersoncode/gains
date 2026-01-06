package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/agent"
	"github.com/spetersoncode/gains/client"
	"github.com/spetersoncode/gains/model"
	"github.com/spetersoncode/gains/observer"
	"github.com/spetersoncode/gains/tool"
)

func demoObserverCombat(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│    Observer Demo - Combat Encounter     │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println("This demo combines the D&D combat agent with the observer package.")
	fmt.Println("The observer tracks all events, tool calls, and calculates costs.")
	fmt.Println()

	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	// Initialize combat state
	state := NewCombatState()

	// Create tool registry with combat tools
	registry := tool.NewRegistry()

	// Roll dice tool
	tool.MustRegisterFunc(registry, "roll_dice", "Roll dice using standard notation (e.g., 1d20, 2d6+3)",
		func(ctx context.Context, args RollDiceArgs) (string, error) {
			numDice, sides, modifier, err := parseDiceNotation(args.Notation)
			if err != nil {
				return "", err
			}
			rolls, subtotal := rollDice(numDice, sides)
			total := subtotal + modifier

			result := fmt.Sprintf(`{"notation":%q,"purpose":%q,"rolls":%v,"total":%d}`,
				args.Notation, args.Purpose, rolls, total)

			// Check for nat 20/1
			if sides == 20 && numDice == 1 {
				if rolls[0] == 20 {
					result = fmt.Sprintf(`{"notation":%q,"purpose":%q,"rolls":%v,"total":%d,"critical":"NAT 20!"}`,
						args.Notation, args.Purpose, rolls, total)
				} else if rolls[0] == 1 {
					result = fmt.Sprintf(`{"notation":%q,"purpose":%q,"rolls":%v,"total":%d,"critical":"NAT 1!"}`,
						args.Notation, args.Purpose, rolls, total)
				}
			}
			return result, nil
		},
	)

	// Add combatant tool
	tool.MustRegisterFunc(registry, "add_combatant", "Add a combatant to the battle",
		func(ctx context.Context, args AddCombatantArgs) (string, error) {
			state.AddCombatant(args.Name, args.HP, args.AC)
			return fmt.Sprintf(`{"success":true,"message":"%s joins battle with %d HP, AC %d"}`,
				args.Name, args.HP, args.AC), nil
		},
	)

	// Apply damage tool
	tool.MustRegisterFunc(registry, "apply_damage", "Apply damage to a combatant",
		func(ctx context.Context, args ApplyDamageArgs) (string, error) {
			newHP, defeated, err := state.ApplyDamage(args.Target, args.Damage)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf(`{"target":%q,"damage":%d,"new_hp":%d,"defeated":%t}`,
				args.Target, args.Damage, newHP, defeated), nil
		},
	)

	// Get combat status tool
	tool.MustRegisterFunc(registry, "get_combat_status", "Get the current status of all combatants",
		func(ctx context.Context, args GetStatusArgs) (string, error) {
			status := state.GetStatus()
			parts := []string{}
			for _, c := range status {
				parts = append(parts, fmt.Sprintf(`{"name":%q,"hp":%d,"max_hp":%d,"ac":%d}`,
					c.Name, c.HP, c.MaxHP, c.AC))
			}
			return fmt.Sprintf(`{"combatants":[%s]}`, joinStrings(parts, ",")), nil
		},
	)

	// Create Console observer with cost tracking
	obs := observer.NewStdoutConsole(
		observer.WithCostTracking(model.ClaudeSonnet45),
		observer.WithVerbose(),
	)
	defer obs.Close()

	// Create the agent
	a := agent.New(c, registry)

	// Simplified combat prompt for faster demo
	combatPrompt := `You are a tabletop RPG Game Master. Run a quick combat encounter!

SETUP:
1. Add "Hero" (HP: 25, AC: 14) and "Goblin" (HP: 12, AC: 11)

COMBAT RULES:
- Roll 1d20+4 for attacks, compare to defender's AC
- Hero damage: 1d8+2, Goblin damage: 1d6+1
- Alternate turns until someone reaches 0 HP

Keep narration brief but exciting. GO!`

	fmt.Println("─── Observer Output ───")
	fmt.Println()

	// Run with observer - the observer will print events as they happen
	result, err := a.Run(ctx, []ai.Message{
		{Role: ai.RoleUser, Content: combatPrompt},
	},
		agent.WithObserver(obs),
		agent.WithMaxSteps(20),
		agent.WithTimeout(5*time.Minute),
	)

	fmt.Println()
	fmt.Println("───────────────────────")

	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		return
	}

	// Show agent results
	fmt.Println("\n═══ Agent Results ═══")
	fmt.Printf("Steps: %d\n", result.Steps)
	fmt.Printf("Termination: %s\n", result.Termination)

	// Show cost summary
	if summary := obs.CostSummary(); summary != nil {
		fmt.Println("\n═══ Cost Summary ═══")
		fmt.Printf("Input tokens:  %d\n", summary.InputTokens)
		fmt.Printf("Output tokens: %d\n", summary.OutputTokens)
		fmt.Printf("Total tokens:  %d\n", summary.TotalTokens)
		fmt.Printf("Estimated cost: $%.6f\n", summary.Cost)
	}

	// Show final combat status
	fmt.Println("\n═══ Final Status ═══")
	for _, c := range state.GetStatus() {
		status := "DEFEATED"
		if c.HP > 0 {
			status = "STANDING"
		}
		fmt.Printf("%s: %d/%d HP - %s\n", c.Name, c.HP, c.MaxHP, status)
	}
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
