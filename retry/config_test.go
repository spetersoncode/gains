package retry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, 10, cfg.MaxAttempts)
	assert.Equal(t, time.Second, cfg.InitialDelay)
	assert.Equal(t, 60*time.Second, cfg.MaxDelay)
	assert.Equal(t, 2.0, cfg.Multiplier)
	assert.Equal(t, 0.1, cfg.Jitter)
}

func TestDisabled(t *testing.T) {
	cfg := Disabled()
	assert.Equal(t, 1, cfg.MaxAttempts)
}

func TestConfigDelay(t *testing.T) {
	cfg := Config{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		Jitter:       0, // No jitter for deterministic test
	}

	assert.Equal(t, 100*time.Millisecond, cfg.Delay(0))
	assert.Equal(t, 200*time.Millisecond, cfg.Delay(1))
	assert.Equal(t, 400*time.Millisecond, cfg.Delay(2))
	assert.Equal(t, 800*time.Millisecond, cfg.Delay(3))
	assert.Equal(t, 1600*time.Millisecond, cfg.Delay(4))
}

func TestConfigDelayMaxCap(t *testing.T) {
	cfg := Config{
		InitialDelay: time.Second,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
		Jitter:       0,
	}

	// 1s * 2^10 = 1024s, but capped at 5s
	assert.Equal(t, 5*time.Second, cfg.Delay(10))
}

func TestConfigDelayNegativeAttempt(t *testing.T) {
	cfg := Config{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		Jitter:       0,
	}

	// Negative attempt should be treated as 0
	assert.Equal(t, 100*time.Millisecond, cfg.Delay(-5))
}

func TestConfigDelayWithJitter(t *testing.T) {
	cfg := Config{
		InitialDelay: time.Second,
		MaxDelay:     60 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1, // 10% jitter
	}

	// Run multiple times to verify jitter adds variance
	delays := make(map[time.Duration]bool)
	for i := 0; i < 100; i++ {
		delay := cfg.Delay(0)
		delays[delay] = true
		// Delay should be within 10% of 1 second: [900ms, 1100ms]
		assert.GreaterOrEqual(t, delay, 900*time.Millisecond)
		assert.LessOrEqual(t, delay, 1100*time.Millisecond)
	}

	// With jitter, we should see multiple different values
	assert.Greater(t, len(delays), 1, "jitter should produce varying delays")
}
