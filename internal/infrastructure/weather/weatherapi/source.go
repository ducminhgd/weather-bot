// Package weatherapi implements application.WeatherFetcher using the WeatherAPI.com
// Forecast API.
// Docs: https://www.weatherapi.com/docs/
package weatherapi

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
const SourceName = "weatherapi"

// maxDays is the maximum forecast horizon accepted by the API (paid plan).
// Free-plan accounts are capped at 3 days by the API itself.
const maxDays = 14

const baseURL = "https://api.weatherapi.com/v1/forecast.json"

var _ application.WeatherFetcher = (*Source)(nil)

// Source fetches weather data from WeatherAPI.com.
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
	params.Set("key", s.apiKey)
	params.Set("q", strconv.FormatFloat(loc.Lat, 'f', -1, 64)+","+strconv.FormatFloat(loc.Lon, 'f', -1, 64))
	params.Set("days", strconv.Itoa(days))
	params.Set("aqi", "no")
	params.Set("alerts", "no")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("weatherapi: build request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("weatherapi: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return nil, fmt.Errorf("weatherapi: HTTP %d: %s", resp.StatusCode, apiErr.Error.Message)
	}

	var raw waResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("weatherapi: decode response: %w", err)
	}

	return mapResponse(raw), nil
}

// ── response types ────────────────────────────────────────────────────────────

type waResponse struct {
	Forecast waForecast `json:"forecast"`
}

type waForecast struct {
	ForecastDay []waForecastDay `json:"forecastday"`
}

type waForecastDay struct {
	Date  string  `json:"date"` // "YYYY-MM-DD"
	Day   waDay   `json:"day"`
	Hours []waHour `json:"hour"`
}

type waDay struct {
	MinTempC float64 `json:"mintemp_c"`
	MaxTempC float64 `json:"maxtemp_c"`
}

type waHour struct {
	TimeEpoch    int64   `json:"time_epoch"`    // Unix timestamp (UTC)
	PrecipMM     float64 `json:"precip_mm"`
	ChanceOfRain int     `json:"chance_of_rain"` // 0–100 %
}

// ── mapping ───────────────────────────────────────────────────────────────────

func mapResponse(raw waResponse) []domain.DailyWeather {
	result := make([]domain.DailyWeather, 0, len(raw.Forecast.ForecastDay))

	for _, fd := range raw.Forecast.ForecastDay {
		date, err := time.Parse("2006-01-02", fd.Date)
		if err != nil {
			continue
		}
		date = date.UTC()

		hourly := make([]domain.HourlyWeather, 0, len(fd.Hours))
		for _, h := range fd.Hours {
			hourly = append(hourly, domain.HourlyWeather{
				Time:                time.Unix(h.TimeEpoch, 0).UTC(),
				PrecipitationMM:     h.PrecipMM,
				PrecipitationChance: float64(h.ChanceOfRain) / 100.0, // normalise to 0.0–1.0
			})
		}

		result = append(result, domain.DailyWeather{
			Date:    date,
			TempMin: fd.Day.MinTempC,
			TempMax: fd.Day.MaxTempC,
			Hourly:  hourly,
		})
	}

	return result
}
