package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// LocationConfig holds a named location with coordinates and display timezone.
type LocationConfig struct {
	Name     string  `mapstructure:"name"`
	Lat      float64 `mapstructure:"lat"`
	Lon      float64 `mapstructure:"lon"`
	Timezone string  `mapstructure:"timezone"` // IANA name, e.g. "Asia/Ho_Chi_Minh"
}

// MessageConfig controls how the weather report is delivered.
type MessageConfig struct {
	// Split true → one message per location; false → all locations in one message.
	Split bool `mapstructure:"split"`
}

// ForecastConfig holds settings that affect how forecast data is interpreted.
type ForecastConfig struct {
	// RainThreshold is the minimum precipitation probability (0–100 %) required
	// to consider an hour as rainy. Defaults to 80.
	RainThreshold int `mapstructure:"rain_threshold"`
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
	Locations     []LocationConfig              `mapstructure:"locations"`
	Days          int                           `mapstructure:"days"`
	Rules         []RuleConfig                  `mapstructure:"rules"`
	Sources       map[string]SourceConfig       `mapstructure:"sources"`
	Notifications map[string]NotificationConfig `mapstructure:"notifications"`
	Message       MessageConfig                 `mapstructure:"message"`
	Forecast      ForecastConfig                `mapstructure:"forecast"`
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
	v.SetDefault("forecast.rain_threshold", 80)

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

	// 3. Environment variables: scalar keys use __ as separator (DAYS, MESSAGE__SPLIT, …).
	// Array/object env vars (LOCATIONS, RULES, SOURCES, NOTIFICATIONS) are parsed as JSON
	// and injected via v.Set — which has the highest viper priority — before Unmarshal,
	// so they override config file values without viper misinterpreting the JSON string.
	// Viper uppercases the config key then applies the replacer to form the env var name.
	// e.g. "message.split" → "MESSAGE.SPLIT" → replacer "." → "__" → "MESSAGE__SPLIT"
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
	v.AutomaticEnv()

	if err := setJSONEnv(v, "LOCATIONS", "locations"); err != nil {
		return nil, err
	}
	if err := setJSONEnv(v, "RULES", "rules"); err != nil {
		return nil, err
	}
	if err := setJSONEnv(v, "SOURCES", "sources"); err != nil {
		return nil, err
	}
	if err := setJSONEnv(v, "NOTIFICATIONS", "notifications"); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}

// setJSONEnv parses the env var envKey as a JSON value and calls v.Set(viperKey, …)
// so it takes highest priority during Unmarshal. No-ops when the env var is unset.
func setJSONEnv(v *viper.Viper, envKey, viperKey string) error {
	val := os.Getenv(envKey)
	if val == "" {
		return nil
	}
	var data any
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return fmt.Errorf("parsing %s env var: %w", envKey, err)
	}
	v.Set(viperKey, data)
	return nil
}
