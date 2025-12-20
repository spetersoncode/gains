package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds the server configuration loaded from environment variables.
type Config struct {
	// Server
	Port     string
	LogLevel string // debug, info, warn, error

	// Provider selection
	Provider string
	Model    string

	// API Keys
	AnthropicKey string
	OpenAIKey    string
	GoogleKey    string

	// Vertex AI (uses ADC for auth)
	VertexProject  string
	VertexLocation string

	// Agent config
	MaxSteps        int
	Timeout         time.Duration
	EnableDemoTools bool
}

// LoadConfig loads configuration from environment variables.
// It loads a .env file if present (silent fail if not found).
func LoadConfig() (*Config, error) {
	godotenv.Load() // Load .env file if present

	cfg := &Config{
		Port:            getEnvOrDefault("AGUI_PORT", "8000"),
		LogLevel:        getEnvOrDefault("AGUI_LOG_LEVEL", "info"),
		Provider:        os.Getenv("GAINS_PROVIDER"),
		Model:           os.Getenv("GAINS_MODEL"),
		AnthropicKey:    os.Getenv("ANTHROPIC_API_KEY"),
		OpenAIKey:       os.Getenv("OPENAI_API_KEY"),
		GoogleKey:       os.Getenv("GOOGLE_API_KEY"),
		VertexProject:   os.Getenv("VERTEX_PROJECT"),
		VertexLocation:  os.Getenv("VERTEX_LOCATION"),
		MaxSteps:        getEnvIntOrDefault("GAINS_MAX_STEPS", 10),
		Timeout:         getEnvDurationOrDefault("GAINS_TIMEOUT", 2*time.Minute),
		EnableDemoTools: getEnvBoolOrDefault("GAINS_DEMO_TOOLS", true),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that required configuration is present.
func (c *Config) Validate() error {
	if c.Provider == "" {
		return fmt.Errorf("GAINS_PROVIDER is required (anthropic, openai, google, or vertex)")
	}

	switch c.Provider {
	case "anthropic":
		if c.AnthropicKey == "" {
			return fmt.Errorf("ANTHROPIC_API_KEY is required for anthropic provider")
		}
	case "openai":
		if c.OpenAIKey == "" {
			return fmt.Errorf("OPENAI_API_KEY is required for openai provider")
		}
	case "google":
		if c.GoogleKey == "" {
			return fmt.Errorf("GOOGLE_API_KEY is required for google provider")
		}
	case "vertex":
		if c.VertexProject == "" || c.VertexLocation == "" {
			return fmt.Errorf("VERTEX_PROJECT and VERTEX_LOCATION are required for vertex provider")
		}
	default:
		return fmt.Errorf("unknown provider: %s (must be anthropic, openai, google, or vertex)", c.Provider)
	}

	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}
