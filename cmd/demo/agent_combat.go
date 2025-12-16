package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/agent"
	"github.com/spetersoncode/gains/client"
	"github.com/spetersoncode/gains/tool"
)

// Combat state management
type Combatant struct {
	Name      string `json:"name"`
	HP        int    `json:"hp"`
	MaxHP     int    `json:"max_hp"`
	AC        int    `json:"ac"`
	Initiative int   `json:"initiative"`
}

type CombatState struct {
	mu          sync.Mutex
	combatants  map[string]*Combatant
	round       int
	turnOrder   []string
	currentTurn int
	log         []string
}

func NewCombatState() *CombatState {
	return &CombatState{
		combatants: make(map[string]*Combatant),
		round:      0,
		log:        []string{},
	}
}

func (cs *CombatState) AddCombatant(name string, hp, ac int) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.combatants[name] = &Combatant{
		Name:  name,
		HP:    hp,
		MaxHP: hp,
		AC:    ac,
	}
}

func (cs *CombatState) GetStatus() map[string]*Combatant {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	result := make(map[string]*Combatant)
	for k, v := range cs.combatants {
		copy := *v
		result[k] = &copy
	}
	return result
}

func (cs *CombatState) ApplyDamage(name string, damage int) (int, bool, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	c, ok := cs.combatants[name]
	if !ok {
		return 0, false, fmt.Errorf("combatant %q not found", name)
	}
	c.HP -= damage
	if c.HP < 0 {
		c.HP = 0
	}
	return c.HP, c.HP == 0, nil
}

func (cs *CombatState) Heal(name string, amount int) (int, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	c, ok := cs.combatants[name]
	if !ok {
		return 0, fmt.Errorf("combatant %q not found", name)
	}
	c.HP += amount
	if c.HP > c.MaxHP {
		c.HP = c.MaxHP
	}
	return c.HP, nil
}

// Dice rolling arguments
type RollDiceArgs struct {
	Notation string `json:"notation" desc:"Dice notation like '1d20', '2d6+5', '3d8-2', or '1d20+7' for attack rolls" required:"true"`
	Purpose  string `json:"purpose" desc:"What this roll is for, e.g. 'attack roll', 'damage', 'initiative'" required:"true"`
}

type AddCombatantArgs struct {
	Name string `json:"name" desc:"Name of the combatant" required:"true"`
	HP   int    `json:"hp" desc:"Starting and maximum hit points" required:"true" min:"1"`
	AC   int    `json:"ac" desc:"Armor class (difficulty to hit)" required:"true" min:"1"`
}

type ApplyDamageArgs struct {
	Target string `json:"target" desc:"Name of the combatant taking damage" required:"true"`
	Damage int    `json:"damage" desc:"Amount of damage to apply" required:"true" min:"0"`
}

type HealArgs struct {
	Target string `json:"target" desc:"Name of the combatant to heal" required:"true"`
	Amount int    `json:"amount" desc:"Amount of HP to restore" required:"true" min:"1"`
}

type GetStatusArgs struct{}

// parseDiceNotation parses strings like "2d6+3" or "1d20-2"
func parseDiceNotation(notation string) (numDice, sides, modifier int, err error) {
	// Pattern: NdS or NdS+M or NdS-M
	re := regexp.MustCompile(`^(\d+)d(\d+)([+-]\d+)?$`)
	matches := re.FindStringSubmatch(strings.ToLower(strings.TrimSpace(notation)))
	if matches == nil {
		return 0, 0, 0, fmt.Errorf("invalid dice notation: %s (use format like '2d6' or '1d20+5')", notation)
	}

	numDice, _ = strconv.Atoi(matches[1])
	sides, _ = strconv.Atoi(matches[2])

	if matches[3] != "" {
		modifier, _ = strconv.Atoi(matches[3])
	}

	if numDice < 1 || numDice > 100 {
		return 0, 0, 0, fmt.Errorf("number of dice must be between 1 and 100")
	}
	if sides < 2 || sides > 100 {
		return 0, 0, 0, fmt.Errorf("dice sides must be between 2 and 100")
	}

	return numDice, sides, modifier, nil
}

func rollDice(numDice, sides int) ([]int, int) {
	rolls := make([]int, numDice)
	total := 0
	for i := 0; i < numDice; i++ {
		roll := rand.Intn(sides) + 1
		rolls[i] = roll
		total += roll
	}
	return rolls, total
}

func demoAgentCombat(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│       Combat Encounter Agent            │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println("This demo showcases an agent running a tabletop RPG combat:")
	fmt.Println("  - Dice rolling tool (supports notation like 1d20+5, 2d6)")
	fmt.Println("  - Combat state tracking (HP, AC, damage)")
	fmt.Println("  - AI-driven narrative combat resolution")
	fmt.Println()

	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	// Initialize combat state
	state := NewCombatState()

	// Create tool registry
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

			// Build result description
			modStr := ""
			if modifier > 0 {
				modStr = fmt.Sprintf(" + %d", modifier)
			} else if modifier < 0 {
				modStr = fmt.Sprintf(" - %d", -modifier)
			}

			result := map[string]interface{}{
				"notation":  args.Notation,
				"purpose":   args.Purpose,
				"rolls":     rolls,
				"subtotal":  subtotal,
				"modifier":  modifier,
				"total":     total,
				"breakdown": fmt.Sprintf("%v%s = %d", rolls, modStr, total),
			}

			// Check for natural 20 or natural 1 on d20
			if sides == 20 && numDice == 1 {
				if rolls[0] == 20 {
					result["critical"] = "NATURAL 20! Critical success!"
				} else if rolls[0] == 1 {
					result["critical"] = "NATURAL 1! Critical failure!"
				}
			}

			jsonBytes, _ := json.Marshal(result)
			return string(jsonBytes), nil
		},
	)

	// Add combatant tool
	tool.MustRegisterFunc(registry, "add_combatant", "Add a combatant to the battle",
		func(ctx context.Context, args AddCombatantArgs) (string, error) {
			state.AddCombatant(args.Name, args.HP, args.AC)
			result := map[string]interface{}{
				"success": true,
				"message": fmt.Sprintf("%s joins the battle with %d HP and %d AC", args.Name, args.HP, args.AC),
				"combatant": map[string]interface{}{
					"name": args.Name,
					"hp":   args.HP,
					"ac":   args.AC,
				},
			}
			jsonBytes, _ := json.Marshal(result)
			return string(jsonBytes), nil
		},
	)

	// Apply damage tool
	tool.MustRegisterFunc(registry, "apply_damage", "Apply damage to a combatant",
		func(ctx context.Context, args ApplyDamageArgs) (string, error) {
			newHP, defeated, err := state.ApplyDamage(args.Target, args.Damage)
			if err != nil {
				return "", err
			}

			result := map[string]interface{}{
				"target":   args.Target,
				"damage":   args.Damage,
				"new_hp":   newHP,
				"defeated": defeated,
			}
			if defeated {
				result["message"] = fmt.Sprintf("%s takes %d damage and is DEFEATED!", args.Target, args.Damage)
			} else {
				result["message"] = fmt.Sprintf("%s takes %d damage! (%d HP remaining)", args.Target, args.Damage, newHP)
			}

			jsonBytes, _ := json.Marshal(result)
			return string(jsonBytes), nil
		},
	)

	// Heal tool
	tool.MustRegisterFunc(registry, "heal", "Heal a combatant",
		func(ctx context.Context, args HealArgs) (string, error) {
			newHP, err := state.Heal(args.Target, args.Amount)
			if err != nil {
				return "", err
			}

			result := map[string]interface{}{
				"target":  args.Target,
				"healed":  args.Amount,
				"new_hp":  newHP,
				"message": fmt.Sprintf("%s is healed for %d HP! (now at %d HP)", args.Target, args.Amount, newHP),
			}

			jsonBytes, _ := json.Marshal(result)
			return string(jsonBytes), nil
		},
	)

	// Get combat status tool
	tool.MustRegisterFunc(registry, "get_combat_status", "Get the current status of all combatants",
		func(ctx context.Context, args GetStatusArgs) (string, error) {
			status := state.GetStatus()

			combatants := make([]map[string]interface{}, 0, len(status))
			for _, c := range status {
				combatants = append(combatants, map[string]interface{}{
					"name":   c.Name,
					"hp":     c.HP,
					"max_hp": c.MaxHP,
					"ac":     c.AC,
					"status": getHealthStatus(c.HP, c.MaxHP),
				})
			}

			result := map[string]interface{}{
				"combatants":     combatants,
				"total_fighters": len(combatants),
			}

			jsonBytes, _ := json.Marshal(result)
			return string(jsonBytes), nil
		},
	)

	fmt.Println("Registered tools:")
	for _, name := range registry.Names() {
		fmt.Printf("  - %s\n", name)
	}
	fmt.Println()

	// Create the agent
	a := agent.New(c, registry)

	// The combat scenario prompt
	combatPrompt := `You are a dramatic tabletop RPG Game Master running an exciting combat encounter!

SCENARIO: A brave adventurer named "Hero" (HP: 30, AC: 15) faces off against a fearsome "Goblin Chief" (HP: 22, AC: 13) in a torch-lit dungeon chamber!

YOUR TASK:
1. First, add both combatants to the battle using add_combatant
2. Run the combat encounter, alternating turns between Hero and Goblin Chief
3. For each attack:
   - Roll 1d20+5 for the attacker's attack roll
   - Compare to defender's AC - if roll >= AC, it's a hit!
   - On a hit, roll damage: Hero uses 1d8+3 (longsword), Goblin Chief uses 1d6+2 (scimitar)
   - Apply the damage
   - Narrate what happens dramatically!
4. Continue until one combatant is defeated (0 HP)
5. Announce the victor with a dramatic conclusion!

Be creative with your narration! Describe the attacks, near-misses, and dramatic moments.
Use get_combat_status periodically to check HP levels.
Natural 20s are critical hits - roll damage twice! Natural 1s are embarrassing misses!

BEGIN THE BATTLE!`

	fmt.Println("Combat Scenario:")
	fmt.Println("─────────────────────────────────────────")
	fmt.Println("Hero (30 HP, AC 15) vs Goblin Chief (22 HP, AC 13)")
	fmt.Println("─────────────────────────────────────────")
	fmt.Println()
	fmt.Print("Press Enter to begin the battle...")
	reader.ReadString('\n')

	fmt.Println("\n⚔️  COMBAT BEGINS! ⚔️")
	fmt.Println()

	// Run agent with streaming
	events := a.RunStream(ctx, []ai.Message{
		{Role: ai.RoleUser, Content: combatPrompt},
	},
		agent.WithMaxSteps(30), // Combat may take many turns
		agent.WithTimeout(10*time.Minute),
	)

	// Track dice rolls for fun stats
	rollCount := 0
	nat20s := 0
	nat1s := 0

	// Process streaming events
	for event := range events {
		switch event.Type {
		case agent.EventStepStart:
			fmt.Printf("\n────── Round %d ──────\n", event.Step)

		case agent.EventStreamDelta:
			fmt.Print(event.Delta)

		case agent.EventToolCallRequested:
			// Special formatting for dice rolls
			if event.ToolCall.Name == "roll_dice" {
				var args RollDiceArgs
				json.Unmarshal([]byte(event.ToolCall.Arguments), &args)
				fmt.Printf("\n  🎲 Rolling %s for %s...\n", args.Notation, args.Purpose)
			} else {
				fmt.Printf("\n  [%s]\n", event.ToolCall.Name)
			}

		case agent.EventToolResult:
			if event.ToolCall.Name == "roll_dice" {
				rollCount++
				var result map[string]interface{}
				json.Unmarshal([]byte(event.ToolResult.Content), &result)

				// Check for crits
				if crit, ok := result["critical"].(string); ok {
					if strings.Contains(crit, "20") {
						nat20s++
						fmt.Printf("  🎯 %s\n", crit)
					} else {
						nat1s++
						fmt.Printf("  💀 %s\n", crit)
					}
				}

				if breakdown, ok := result["breakdown"].(string); ok {
					total := result["total"].(float64)
					fmt.Printf("  → %s (Total: %.0f)\n", breakdown, total)
				}
			} else if event.ToolCall.Name == "apply_damage" {
				var result map[string]interface{}
				json.Unmarshal([]byte(event.ToolResult.Content), &result)
				if msg, ok := result["message"].(string); ok {
					if defeated, ok := result["defeated"].(bool); ok && defeated {
						fmt.Printf("  💥 %s\n", msg)
					} else {
						fmt.Printf("  ⚔️  %s\n", msg)
					}
				}
			} else if event.ToolCall.Name == "get_combat_status" {
				var result map[string]interface{}
				json.Unmarshal([]byte(event.ToolResult.Content), &result)
				if combatants, ok := result["combatants"].([]interface{}); ok {
					fmt.Println("  📊 Combat Status:")
					for _, c := range combatants {
						if cm, ok := c.(map[string]interface{}); ok {
							name := cm["name"].(string)
							hp := cm["hp"].(float64)
							maxHP := cm["max_hp"].(float64)
							status := cm["status"].(string)
							bar := healthBar(int(hp), int(maxHP))
							fmt.Printf("     %s: %s %.0f/%.0f HP (%s)\n", name, bar, hp, maxHP, status)
						}
					}
				}
			}

		case agent.EventStepComplete:
			// Just a visual separator

		case agent.EventAgentComplete:
			fmt.Println()
			fmt.Println()
			fmt.Println("═══════════════════════════════════════")
			fmt.Println("         COMBAT CONCLUDED!")
			fmt.Println("═══════════════════════════════════════")
			fmt.Printf("\nBattle Statistics:\n")
			fmt.Printf("  Total Dice Rolls: %d\n", rollCount)
			fmt.Printf("  Natural 20s: %d 🎯\n", nat20s)
			fmt.Printf("  Natural 1s: %d 💀\n", nat1s)

			// Final status
			fmt.Println("\nFinal Status:")
			for _, c := range state.GetStatus() {
				status := "DEFEATED"
				if c.HP > 0 {
					status = "VICTORIOUS"
				}
				fmt.Printf("  %s: %d/%d HP - %s\n", c.Name, c.HP, c.MaxHP, status)
			}

		case agent.EventError:
			fmt.Fprintf(os.Stderr, "\nError: %v\n", event.Error)
		}
	}
}

func getHealthStatus(hp, maxHP int) string {
	ratio := float64(hp) / float64(maxHP)
	switch {
	case ratio <= 0:
		return "Defeated"
	case ratio <= 0.25:
		return "Critical"
	case ratio <= 0.5:
		return "Bloodied"
	case ratio <= 0.75:
		return "Wounded"
	default:
		return "Healthy"
	}
}

func healthBar(hp, maxHP int) string {
	width := 10
	filled := int(float64(hp) / float64(maxHP) * float64(width))
	if hp > 0 && filled == 0 {
		filled = 1
	}
	empty := width - filled
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", empty) + "]"
}
