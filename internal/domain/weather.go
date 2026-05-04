package domain

import "time"

// HourlyWeather holds weather measurements for a single hour.
type HourlyWeather struct {
	Time                time.Time
	PrecipitationMM     float64 // rainfall in mm
	PrecipitationChance float64 // probability 0.0–1.0
}

// DailyWeather holds the forecast for a single day.
type DailyWeather struct {
	Date    time.Time
	TempMin float64 // °C
	TempMax float64 // °C
	Hourly  []HourlyWeather
}
