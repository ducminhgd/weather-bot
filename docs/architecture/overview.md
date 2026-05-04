# Architecture Overview

## Clean Architecture Layers

```
┌─────────────────────────────────────────────────────────────────┐
│  cmd/weather-bot/main.go  — entry point, wiring, cobra CLI      │
├─────────────────────────────────────────────────────────────────┤
│  internal/infrastructure/  — Frameworks & Drivers               │
│    weather/openweathermap/  weather/open-meteo/                  │
│    notification/telegram/   config/                              │
├─────────────────────────────────────────────────────────────────┤
│  internal/adapters/         — Interface Adapters (if any HTTP)   │
├─────────────────────────────────────────────────────────────────┤
│  internal/application/      — Use Cases                         │
│    weather_service.go        weather_fetcher.go (interface)      │
│    notifier.go (interface)   rules.go                            │
├─────────────────────────────────────────────────────────────────┤
│  internal/domain/           — Entities, Value Objects            │
│    weather.go  location.go  rules.go  errors.go                  │
└─────────────────────────────────────────────────────────────────┘
```

## Directory Layout

```
weather-bot/
├── cmd/
│   └── weather-bot/
│       └── main.go                  # cobra root cmd, DI wiring
│
├── internal/
│   ├── domain/
│   │   ├── weather.go               # DailyWeather, HourlyWeather entities
│   │   ├── location.go              # Location value object
│   │   ├── rules.go                 # RuleSet, rule evaluation
│   │   └── errors.go                # ErrNoData, ErrInvalidLocation, …
│   │
│   ├── application/
│   │   ├── weather_fetcher.go       # WeatherFetcher interface
│   │   ├── notifier.go              # Notifier interface
│   │   ├── rules_evaluator.go       # RulesEvaluator interface
│   │   └── weather_service.go       # main use case: fetch → evaluate → notify
│   │
│   └── infrastructure/
│       ├── weather/
│       │   ├── openweathermap/
│       │   │   └── source.go        # implements application.WeatherFetcher
│       │   └── openmeteo/
│       │       └── source.go        # implements application.WeatherFetcher
│       ├── notification/
│       │   ├── console/
│       │   │   └── notifier.go      # prints to stdout (always active)
│       │   └── telegram/
│       │       └── notifier.go      # implements application.Notifier
│       └── config/
│           └── config.go            # viper-based loader
│
├── docs/
│   ├── architecture/
│   │   └── overview.md              # this file
│   └── design/
│       ├── ADR-001-weather-services.md
│       └── ADR-002-configuration.md
│
├── go.mod
├── README.md
└── CLAUDE.md
```

## Data Flow

```
main.go
  │
  ├─ Load config (viper: defaults → file → env → flags)
  │
  ├─ Build WeatherFetcher(s) from config.sources
  │    └─ openweathermap.Source  /  openmeteo.Source
  │
  ├─ Build Notifier(s) from config.notifications
  │    └─ console.Notifier (always)  /  telegram.Notifier
  │
  └─ WeatherService.Run(ctx)
       │
       ├─ Fetch(location, days) → []DailyWeather
       ├─ EvaluateRules(data, ruleSet) → message string
       ├─ Print to stdout
       └─ Notify(message) → each configured notifier
```

## Key Interfaces

```go
// application/weather_fetcher.go
type WeatherFetcher interface {
    Fetch(ctx context.Context, loc domain.Location, days int) ([]domain.DailyWeather, error)
}

// application/notifier.go
type Notifier interface {
    Send(ctx context.Context, message string) error
}

// application/rules_evaluator.go
type RulesEvaluator interface {
    Evaluate(data []domain.DailyWeather, rules domain.RuleSet) string
}
```

## Domain Entities

```go
// domain/weather.go
type HourlyWeather struct {
    Time                time.Time
    PrecipitationMM     float64
    PrecipitationChance float64 // 0.0–1.0
}

type DailyWeather struct {
    Date    time.Time
    TempMin float64
    TempMax float64
    Hourly  []HourlyWeather
}

// domain/location.go
type Location struct {
    Lat float64
    Lon float64
}

// domain/rules.go
type RuleSet struct {
    Rain    bool
    TempLTE *float64
    TempGTE *float64
}
```

## Adding a New Weather Source

1. Create `internal/infrastructure/weather/<name>/source.go`
2. Implement `application.WeatherFetcher`
3. Register in `cmd/weather-bot/main.go` factory by source name key

No existing code changes required.

## Adding a New Notifier

1. Create `internal/infrastructure/notification/<name>/notifier.go`
2. Implement `application.Notifier`
3. Register in `cmd/weather-bot/main.go` factory by notification type key

No existing code changes required.
