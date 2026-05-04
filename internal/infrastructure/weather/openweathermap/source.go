// Package openweathermap implements application.WeatherFetcher using the
// OpenWeatherMap One Call API 3.0.
// Docs: https://openweathermap.org/api/one-call-3
package openweathermap

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ducminhgd/weather-bot/internal/application"
	"github.com/ducminhgd/weather-bot/internal/domain"
)

// SourceName is the key used in config.sources to select this source.
const SourceName = "openweathermap"

// maxDays is the maximum number of daily entries the One Call API returns.
const maxDays = 8

const baseURL = "https://api.openweathermap.org/data/3.0/onecall"

var _ application.WeatherFetcher = (*Source)(nil)

// Source fetches weather data from OpenWeatherMap.
type Source struct {
	apiKey string
	client *http.Client
}

// New creates a Source with the given API key.
func New(apiKey string) *Source {
	return &Source{
		apiKey: apiKey,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Fetch retrieves a forecast for loc covering the next days days.
func (s *Source) Fetch(ctx context.Context, loc domain.Location, days int) ([]domain.DailyWeather, error) {
	if days > maxDays {
		days = maxDays
	}

	params := url.Values{}
	params.Set("lat", strconv.FormatFloat(loc.Lat, 'f', -1, 64))
	params.Set("lon", strconv.FormatFloat(loc.Lon, 'f', -1, 64))
	params.Set("appid", s.apiKey)
	params.Set("units", "metric")
	params.Set("exclude", "minutely,alerts,current")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("openweathermap: build request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openweathermap: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return nil, fmt.Errorf("openweathermap: HTTP %d: %s", resp.StatusCode, apiErr.Message)
	}

	var raw owmResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("openweathermap: decode response: %w", err)
	}

	return mapResponse(raw, days), nil
}

// ── response types ────────────────────────────────────────────────────────────

type owmResponse struct {
	Daily  []owmDaily  `json:"daily"`
	Hourly []owmHourly `json:"hourly"`
}

type owmDaily struct {
	Dt   int64   `json:"dt"`
	Temp owmTemp `json:"temp"`
}

type owmTemp struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type owmHourly struct {
	Dt   int64    `json:"dt"`
	Pop  float64  `json:"pop"` // probability of precipitation 0.0–1.0
	Rain *owmRain `json:"rain"`
}

type owmRain struct {
	OneH float64 `json:"1h"`
}

// ── mapping ───────────────────────────────────────────────────────────────────

func mapResponse(raw owmResponse, days int) []domain.DailyWeather {
	// group hourly entries by date key (YYYY-MM-DD in UTC)
	hourlyByDate := make(map[string][]domain.HourlyWeather, len(raw.Hourly))
	for _, h := range raw.Hourly {
		t := time.Unix(h.Dt, 0).UTC()
		key := t.Format("2006-01-02")
		mm := 0.0
		if h.Rain != nil {
			mm = h.Rain.OneH
		}
		hourlyByDate[key] = append(hourlyByDate[key], domain.HourlyWeather{
			Time:                t,
			PrecipitationMM:     mm,
			PrecipitationChance: h.Pop,
		})
	}

	if days > len(raw.Daily) {
		days = len(raw.Daily)
	}

	result := make([]domain.DailyWeather, 0, days)
	for _, d := range raw.Daily[:days] {
		date := time.Unix(d.Dt, 0).UTC().Truncate(24 * time.Hour)
		key := date.Format("2006-01-02")
		result = append(result, domain.DailyWeather{
			Date:    date,
			TempMin: d.Temp.Min,
			TempMax: d.Temp.Max,
			Hourly:  hourlyByDate[key],
		})
	}
	return result
}
