package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Location holds geographic coordinates.
type Location struct {
	Lat float64 `mapstructure:"lat"`
	Lon float64 `mapstructure:"lon"`
}

// RuleConfig defines a single rule condition. Name is a human-readable label
// used in alert messages so the recipient knows which rule fired.
// Type identifies the evaluator; Params carries type-specific settings.
//
// Built-in types:
//
//	"rain"  — alert when rain is forecast; no params required.
//	"temp"  — alert on temperature threshold; params: "condition" ("lte"|"gte"), "value" (float string).
//
// New rule types can be added by registering a new evaluator for the type string.
type RuleConfig struct {
	Name   string            `mapstructure:"name"   json:"name"`
	Type   string            `mapstructure:"type"   json:"type"`
	Params map[string]string `mapstructure:"params" json:"params"`
}

// SourceConfig holds credentials for a weather data source.
type SourceConfig struct {
	APIKey string `mapstructure:"api_key"`
}

// NotificationConfig holds configuration for a single notification channel.
type NotificationConfig struct {
	Type   string            `mapstructure:"type"`
	Params map[string]string `mapstructure:"params"`
}

// Config is the top-level application configuration.
type Config struct {
	Location      Location                      `mapstructure:"location"`
	Days          int                           `mapstructure:"days"`
	Rules         []RuleConfig                  `mapstructure:"rules"`
	Sources       map[string]SourceConfig       `mapstructure:"sources"`
	Notifications map[string]NotificationConfig `mapstructure:"notifications"`
}

// Load builds a Config from the layered priority stack (lowest to highest):
//
//	code defaults → config file → environment variables → CLI flags
//
// CLI flags must be bound to v by the caller before invoking Load.
// cfgFile is optional; when empty, Load searches for config.yaml in ./ and $HOME/.weather-bot/.
func Load(v *viper.Viper, cfgFile string) (*Config, error) {
	// 1. Defaults
	v.SetDefault("days", 1)

	// 2. Config file
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.weather-bot")
	}
	_ = v.ReadInConfig() // absence of a config file is not an error

	// 3. Environment variables: LOCATION__LAT → location.lat
	v.SetEnvKeyReplacer(strings.NewReplacer("__", "."))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// RULES, SOURCES and NOTIFICATIONS may be provided as JSON strings in env vars,
	// which is convenient for CI/CD. They override any values already decoded
	// from the config file.
	if r := os.Getenv("RULES"); r != "" {
		if err := json.Unmarshal([]byte(r), &cfg.Rules); err != nil {
			return nil, fmt.Errorf("parsing RULES env var: %w", err)
		}
	}
	if s := os.Getenv("SOURCES"); s != "" {
		if err := json.Unmarshal([]byte(s), &cfg.Sources); err != nil {
			return nil, fmt.Errorf("parsing SOURCES env var: %w", err)
		}
	}
	if n := os.Getenv("NOTIFICATIONS"); n != "" {
		if err := json.Unmarshal([]byte(n), &cfg.Notifications); err != nil {
			return nil, fmt.Errorf("parsing NOTIFICATIONS env var: %w", err)
		}
	}

	return &cfg, nil
}
