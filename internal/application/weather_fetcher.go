package application

import (
	"context"

	"github.com/ducminhgd/weather-bot/internal/domain"
)

// WeatherFetcher retrieves weather forecast data from an external source.
// Implementations live in internal/infrastructure/weather/.
type WeatherFetcher interface {
	Fetch(ctx context.Context, loc domain.Location, days int) ([]domain.DailyWeather, error)
}
