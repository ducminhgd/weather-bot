# ADR-001: Weather Data Source Selection

* Status: accepted
* Date: 2026-05-04

## Context and Problem Statement

The bot needs to fetch weather data — current conditions and multi-day forecasts including temperature ranges and hourly precipitation. We need at least two supported sources so users can choose based on their region, rate limits, or cost constraints.

## Decision Drivers

* Must provide hourly precipitation data (rain start/stop detection)
* Must provide daily temperature min/max
* Free tier must exist for personal use
* Go-friendly HTTP/JSON API (no SDK dependency required)
* Multi-day forecast support (configurable via `DAYS`)

## Considered Options

* OpenWeatherMap One Call API 3.0
* Open-Meteo
* WeatherAPI.com

## Decision Outcome

Chosen options: **OpenWeatherMap** and **Open-Meteo**, both implemented.

- OpenWeatherMap is the dominant paid-tier option with broad regional accuracy.
- Open-Meteo is the free default with no API key requirement.

Either source is selected at runtime via the `SOURCES` configuration. The architecture allows additional sources to be added without modifying existing code.

---

## Option Details

### OpenWeatherMap One Call API 3.0

- **Base URL:** `https://api.openweathermap.org/data/3.0/onecall`
- **Auth:** API key via `appid` query parameter
- **Free tier:** 1,000 calls/day (requires subscription activation, billed at $0 above limit)
- **Relevant fields:**
  - `daily[].temp.min` / `daily[].temp.max` — daily temperature range
  - `hourly[].pop` — probability of precipitation (0.0–1.0)
  - `hourly[].rain.1h` — rain volume mm/h
  - `hourly[].dt` — Unix timestamp
- **Docs:** https://openweathermap.org/api/one-call-3

**Configuration keys:**

```yaml
sources:
  openweathermap:
    api_key: "<key>"
```

**Good, because:**
- Industry standard, widely used
- Covers 200,000+ cities globally
- Rich data model (UV index, alerts, etc.)

**Bad, because:**
- Requires registration and billing setup
- Rate-limited to 1,000 calls/day on free plan

---

### Open-Meteo

- **Base URL:** `https://api.open-meteo.com/v1/forecast`
- **Auth:** None required for non-commercial use (API key optional for higher limits)
- **Free tier:** Unlimited calls for non-commercial use
- **Relevant fields:**
  - `daily.temperature_2m_max` / `daily.temperature_2m_min`
  - `hourly.precipitation_probability`
  - `hourly.precipitation`
  - `hourly.time` — ISO 8601 timestamps
- **Docs:** https://open-meteo.com/en/docs

**Configuration keys:**

```yaml
sources:
  open-meteo: {}   # no credentials needed
```

**Good, because:**
- Completely free, no API key needed
- Open-source model data (ECMWF, GFS, etc.)
- Simple, clean JSON response

**Bad, because:**
- Less granular alert data
- No severe weather alerts

---

## Source Interface

Both sources implement the same application-layer interface:

```go
// internal/application/weather_fetcher.go
type WeatherFetcher interface {
    Fetch(ctx context.Context, loc domain.Location, days int) ([]domain.DailyWeather, error)
}
```

Each infrastructure implementation maps its API response to `domain.DailyWeather`.
