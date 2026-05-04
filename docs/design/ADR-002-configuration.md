# ADR-002: Configuration System Design

* Status: accepted
* Date: 2026-05-04

## Context and Problem Statement

The bot must be configurable via multiple mechanisms to support local development, CI/CD (GitHub Actions), and direct CLI invocation. Values from more specific sources must override less specific ones.

## Decision Drivers

* Must support JSON and YAML config files
* Must support environment variables
* Must support runtime CLI flags
* Priority order (highest to lowest): CLI flags > env vars > config file > code defaults
* Must map cleanly to a typed Go struct for compile-time safety

## Considered Options

* `spf13/viper` — full-featured config library with layered loading
* Manual layered loading — custom code reading env vars and flags into a struct
* `knadh/koanf` — lightweight, modular, similar to viper

## Decision Outcome

Chosen option: **`spf13/viper`**

Viper handles all four layers (defaults, file, env, flags) natively and integrates with `spf13/cobra` for CLI flags — which will be needed for the `cmd/` entry point.

---

## Configuration Schema

### Priority Stack (highest → lowest)

```
CLI flags  →  Environment variables  →  Config file  →  Code defaults
```

### Struct Definition

```go
// internal/infrastructure/config/config.go

type Location struct {
    Lat float64 `mapstructure:"lat"`
    Lon float64 `mapstructure:"lon"`
}

// RuleConfig is one rule in the ordered list of rules.
// Name appears in alert messages so the recipient knows which rule fired.
// Type identifies the evaluator; Params carries type-specific values.
// Multiple rules of the same type are allowed.
type RuleConfig struct {
    Name   string            `mapstructure:"name"   json:"name"`
    Type   string            `mapstructure:"type"   json:"type"`
    Params map[string]string `mapstructure:"params" json:"params"`
}

type SourceConfig struct {
    APIKey string `mapstructure:"api_key"`
}

type NotificationConfig struct {
    Type  string            `mapstructure:"type"`
    Params map[string]string `mapstructure:"params"`
}

type Config struct {
    Location      Location                      `mapstructure:"location"`
    Days          int                           `mapstructure:"days"`
    Rules         []RuleConfig                  `mapstructure:"rules"`
    Sources       map[string]SourceConfig       `mapstructure:"sources"`
    Notifications map[string]NotificationConfig `mapstructure:"notifications"`
}
```

### Default Values (in code)

| Field | Default |
|---|---|
| `days` | `1` |
| `rules` | empty slice (no rules active) |
| `sources` | empty map |
| `notifications` | empty map |

---

## Environment Variable Mapping

Viper maps env vars to config keys with `__` as the key delimiter:

| Env Variable | Config Key |
|---|---|
| `LOCATION__LAT` | `location.lat` |
| `LOCATION__LON` | `location.lon` |
| `DAYS` | `days` |

`RULES`, `SOURCES`, and `NOTIFICATIONS` are passed as JSON strings when set via env vars, and decoded separately (they cannot be expressed as flat `KEY__SUBKEY` env vars because they are slices/maps of arbitrary keys).

---

## Config File

Supported formats: `.yaml`, `.yml`, `.json`.

Default lookup path (in order):
1. Path specified by `--config` CLI flag
2. `./config.yaml`
3. `$HOME/.weather-bot/config.yaml`

Example `config.yaml`:

```yaml
location:
  lat: 10.823
  lon: 106.630

days: 3

rules:
  - name: rain-alert
    type: rain
  - name: cold-alert
    type: temp
    params:
      condition: lte
      value: "20.0"
  - name: heat-alert
    type: temp
    params:
      condition: gte
      value: "38.0"

sources:
  openweathermap:
    api_key: "abc123"
  open-meteo: {}

notifications:
  my-telegram:
    type: telegram
    params:
      bot_token: "1234:ABCD"
      chat_id: "-100123456"
```

---

## CLI Flags

Registered with `cobra` in `cmd/weather-bot/main.go`:

| Flag | Maps to |
|---|---|
| `--config` | config file path |
| `--lat` | `location.lat` |
| `--lon` | `location.lon` |
| `--days` | `days` |

---

## Loader Implementation

```go
func Load(cfgFile string) (*Config, error) {
    v := viper.New()

    // 1. Defaults
    v.SetDefault("days", 1)
    v.SetDefault("rules.rain", false)

    // 2. Config file
    if cfgFile != "" {
        v.SetConfigFile(cfgFile)
    } else {
        v.SetConfigName("config")
        v.SetConfigType("yaml")
        v.AddConfigPath(".")
        v.AddConfigPath("$HOME/.weather-bot")
    }
    _ = v.ReadInConfig() // file is optional

    // 3. Env vars
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
    v.AutomaticEnv()

    // 4. CLI flags bound externally by cobra before calling Load

    var cfg Config
    return &cfg, v.Unmarshal(&cfg)
}
```

---

## Pros and Cons of Options

### spf13/viper

* Good, because handles all four layers natively
* Good, because integrates with cobra for CLI flags
* Good, because supports JSON and YAML files
* Bad, because adds a dependency (~400KB compiled)
* Bad, because global state in default viper instance (mitigated by using `viper.New()`)

### Manual layered loading

* Good, because zero dependencies
* Bad, because significant boilerplate for env parsing, type coercion, and file formats
* Bad, because YAML support requires an additional library anyway

### koanf

* Good, because more modular than viper
* Good, because no global state
* Bad, because smaller community, less `cobra` integration documentation
