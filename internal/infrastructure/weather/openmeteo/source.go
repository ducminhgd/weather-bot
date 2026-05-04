// Package openmeteo implements application.WeatherFetcher using the
// Open-Meteo forecast API (no API key required for non-commercial use).
// Docs: https://open-meteo.com/en/docs
package openmeteo

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
const SourceName = "open-meteo"

const baseURL = "https://api.open-meteo.com/v1/forecast"

// hourlyTimeLayout matches the format returned by Open-Meteo ("2006-01-02T15:04").
const hourlyTimeLayout = "2006-01-02T15:04"

var _ application.WeatherFetcher = (*Source)(nil)

// Source fetches weather data from Open-Meteo.
type Source struct {
	client *http.Client
}

// New creates a Source. No credentials are required.
func New() *Source {
	return &Source{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Fetch retrieves a forecast for loc covering the next days days.
func (s *Source) Fetch(ctx context.Context, loc domain.Location, days int) ([]domain.DailyWeather, error) {
	params := url.Values{}
	params.Set("latitude", strconv.FormatFloat(loc.Lat, 'f', -1, 64))
	params.Set("longitude", strconv.FormatFloat(loc.Lon, 'f', -1, 64))
	params.Set("daily", "temperature_2m_max,temperature_2m_min")
	params.Set("hourly", "precipitation_probability,precipitation")
	params.Set("timezone", "UTC")
	params.Set("forecast_days", strconv.Itoa(days))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("open-meteo: build request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("open-meteo: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Reason string `json:"reason"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return nil, fmt.Errorf("open-meteo: HTTP %d: %s", resp.StatusCode, apiErr.Reason)
	}

	var raw omResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("open-meteo: decode response: %w", err)
	}

	return mapResponse(raw)
}

// ── response types ────────────────────────────────────────────────────────────

type omResponse struct {
	Daily  omDailyData  `json:"daily"`
	Hourly omHourlyData `json:"hourly"`
}

type omDailyData struct {
	Time    []string  `json:"time"`
	TempMax []float64 `json:"temperature_2m_max"`
	TempMin []float64 `json:"temperature_2m_min"`
}

type omHourlyData struct {
	Time                []string  `json:"time"`
	PrecipitationProb   []float64 `json:"precipitation_probability"` // 0–100 %
	Precipitation       []float64 `json:"precipitation"`              // mm
}

// ── mapping ───────────────────────────────────────────────────────────────────

func mapResponse(raw omResponse) ([]domain.DailyWeather, error) {
	// group hourly entries by date key (YYYY-MM-DD)
	hourlyByDate := make(map[string][]domain.HourlyWeather, len(raw.Hourly.Time))
	for i, ts := range raw.Hourly.Time {
		t, err := time.Parse(hourlyTimeLayout, ts)
		if err != nil {
			return nil, fmt.Errorf("open-meteo: parse hourly time %q: %w", ts, err)
		}
		t = t.UTC()
		key := t.Format("2006-01-02")

		prob := 0.0
		if i < len(raw.Hourly.PrecipitationProb) {
			prob = raw.Hourly.PrecipitationProb[i] / 100.0 // normalise to 0.0–1.0
		}
		mm := 0.0
		if i < len(raw.Hourly.Precipitation) {
			mm = raw.Hourly.Precipitation[i]
		}

		hourlyByDate[key] = append(hourlyByDate[key], domain.HourlyWeather{
			Time:                t,
			PrecipitationMM:     mm,
			PrecipitationChance: prob,
		})
	}

	result := make([]domain.DailyWeather, 0, len(raw.Daily.Time))
	for i, dateStr := range raw.Daily.Time {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, fmt.Errorf("open-meteo: parse daily time %q: %w", dateStr, err)
		}
		date = date.UTC()

		dw := domain.DailyWeather{
			Date:   date,
			Hourly: hourlyByDate[date.Format("2006-01-02")],
		}
		if i < len(raw.Daily.TempMax) {
			dw.TempMax = raw.Daily.TempMax[i]
		}
		if i < len(raw.Daily.TempMin) {
			dw.TempMin = raw.Daily.TempMin[i]
		}
		result = append(result, dw)
	}

	return result, nil
}
