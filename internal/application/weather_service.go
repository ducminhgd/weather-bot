package application

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ducminhgd/weather-bot/internal/domain"
)

// LocationForecast pairs a location with its fetched daily forecast.
type LocationForecast struct {
	Location domain.Location
	Days     []domain.DailyWeather
}

// WeatherService is the main use case: fetch → format → print → notify.
type WeatherService struct {
	fetchers      []WeatherFetcher
	notifiers     []Notifier
	rules         []Rule
	registry      *RuleRegistry
	locations     []domain.Location
	days          int
	splitMessages bool
}

// NewWeatherService creates a WeatherService.
// fetchers are tried in order; the first successful response is used.
// notifiers are called when a rule matches (or unconditionally when no rules are set).
// splitMessages true sends one message per location; false combines all into one.
func NewWeatherService(
	fetchers []WeatherFetcher,
	notifiers []Notifier,
	rules []Rule,
	registry *RuleRegistry,
	locations []domain.Location,
	days int,
	splitMessages bool,
) *WeatherService {
	return &WeatherService{
		fetchers:      fetchers,
		notifiers:     notifiers,
		rules:         rules,
		registry:      registry,
		locations:     locations,
		days:          days,
		splitMessages: splitMessages,
	}
}

// Run executes the full pipeline for every configured location.
func (s *WeatherService) Run(ctx context.Context) error {
	forecasts := make([]LocationForecast, 0, len(s.locations))
	for _, loc := range s.locations {
		data, err := s.fetchWeather(ctx, loc)
		if err != nil {
			log.Printf("fetch %q: %v", loc.Name, err)
			continue
		}
		forecasts = append(forecasts, LocationForecast{Location: loc, Days: data})
	}

	if len(forecasts) == 0 {
		return domain.ErrNoData
	}

	if s.splitMessages {
		for _, f := range forecasts {
			msg := FormatMessage(f.Location, f.Days)
			fmt.Println(msg)
			s.notify(ctx, msg, f.Days)
		}
	} else {
		msg := FormatCombinedMessage(forecasts)
		fmt.Println(msg)
		// notify if any location triggers a rule
		for _, f := range forecasts {
			if len(s.rules) == 0 || s.registry.MatchesAny(f.Days, s.rules) {
				s.sendAll(ctx, msg)
				break
			}
		}
	}

	return nil
}

// notify sends msg to all notifiers when the rule conditions are met for data.
func (s *WeatherService) notify(ctx context.Context, msg string, data []domain.DailyWeather) {
	if len(s.rules) == 0 || s.registry.MatchesAny(data, s.rules) {
		s.sendAll(ctx, msg)
	}
}

func (s *WeatherService) sendAll(ctx context.Context, msg string) {
	for _, n := range s.notifiers {
		if err := n.Send(ctx, msg); err != nil {
			log.Printf("notifier error: %v", err)
		}
	}
}

// fetchWeather tries each fetcher in order and returns the first successful result.
func (s *WeatherService) fetchWeather(ctx context.Context, loc domain.Location) ([]domain.DailyWeather, error) {
	var lastErr error
	for _, f := range s.fetchers {
		data, err := f.Fetch(ctx, loc, s.days)
		if err == nil && len(data) > 0 {
			return data, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, domain.ErrNoData
}

// ── message formatting ────────────────────────────────────────────────────────

// FormatMessage builds a weather report for a single location.
func FormatMessage(loc domain.Location, days []domain.DailyWeather) string {
	var b strings.Builder
	tz := loadTZ(loc.Timezone)

	b.WriteString("**Weather forecast**\n")
	fmt.Fprintf(&b, "1. **Location:** %s\n", locationLabel(loc))
	writeDays(&b, tz, days, 2)

	return strings.TrimRight(b.String(), "\n")
}

// FormatCombinedMessage builds a single report containing all locations.
func FormatCombinedMessage(forecasts []LocationForecast) string {
	var b strings.Builder

	b.WriteString("**Weather forecast**\n")
	for _, f := range forecasts {
		tz := loadTZ(f.Location.Timezone)
		fmt.Fprintf(&b, "\n**%s**\n", locationLabel(f.Location))
		writeDays(&b, tz, f.Days, 1)
	}

	return strings.TrimRight(b.String(), "\n")
}

// writeDays appends the numbered day entries to b. startIndex is the opening list number.
func writeDays(b *strings.Builder, tz *time.Location, days []domain.DailyWeather, startIndex int) {
	now := time.Now().In(tz)
	todayStr := now.Format("2006-01-02")
	tomorrowStr := now.AddDate(0, 0, 1).Format("2006-01-02")

	for i, day := range days {
		dayInTZ := day.Date.In(tz)
		dateStr := dayInTZ.Format("2006-01-02")
		dayLabel := dayInTZ.Format("02/01/2006")
		switch dateStr {
		case todayStr:
			dayLabel += " (today)"
		case tomorrowStr:
			dayLabel += " (tomorrow)"
		}

		fmt.Fprintf(b, "%d. **%s**\n", startIndex+i, dayLabel)
		fmt.Fprintf(b, "   1. **Temperature:** %.1f - %.1f °C\n", day.TempMin, day.TempMax)

		intervals := rainIntervals(day.Hourly, tz)
		if len(intervals) == 0 {
			b.WriteString("   2. **Rain chances**: No\n")
		} else {
			b.WriteString("   2. **Rain chances**: Yes\n")
			for j, iv := range intervals {
				fmt.Fprintf(b, "      %d. From %s to %s\n",
					j+1,
					iv[0].In(tz).Format("15:04"),
					iv[1].In(tz).Format("15:04"),
				)
			}
		}
	}
}

// locationLabel returns a display string for the location including timezone.
func locationLabel(loc domain.Location) string {
	label := loc.Name
	if label == "" {
		label = fmt.Sprintf("%.4f, %.4f", loc.Lat, loc.Lon)
	}
	if loc.Timezone != "" {
		label += " (" + loc.Timezone + ")"
	}
	return label
}

// loadTZ returns the time.Location for tz, falling back to UTC on error.
func loadTZ(tz string) *time.Location {
	if tz == "" {
		return time.UTC
	}
	l, err := time.LoadLocation(tz)
	if err != nil {
		log.Printf("unknown timezone %q, falling back to UTC", tz)
		return time.UTC
	}
	return l
}

// rainIntervals returns consecutive time windows within hours where precipitation
// is expected (chance ≥ RainThreshold or measured mm > 0).
// Times in the returned intervals are in UTC; the caller converts to display timezone.
func rainIntervals(hours []domain.HourlyWeather, _ *time.Location) [][2]time.Time {
	var intervals [][2]time.Time
	inRain := false
	var start time.Time

	for _, h := range hours {
		rainy := h.PrecipitationChance >= RainThreshold || h.PrecipitationMM > 0
		if rainy && !inRain {
			start = h.Time
			inRain = true
		} else if !rainy && inRain {
			intervals = append(intervals, [2]time.Time{start, h.Time})
			inRain = false
		}
	}
	if inRain && len(hours) > 0 {
		end := hours[len(hours)-1].Time.Add(time.Hour)
		intervals = append(intervals, [2]time.Time{start, end})
	}

	return intervals
}
