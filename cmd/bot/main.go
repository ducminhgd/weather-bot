package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ducminhgd/weather-bot/internal/application"
	"github.com/ducminhgd/weather-bot/internal/domain"
	"github.com/ducminhgd/weather-bot/internal/infrastructure/config"
	"github.com/ducminhgd/weather-bot/internal/infrastructure/notification/telegram"
	"github.com/ducminhgd/weather-bot/internal/infrastructure/weather/openmeteo"
	"github.com/ducminhgd/weather-bot/internal/infrastructure/weather/openweathermap"
	"github.com/ducminhgd/weather-bot/internal/infrastructure/weather/weatherapi"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := newRootCmd().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	v := viper.New()
	var cfgFile string

	cmd := &cobra.Command{
		Use:   "bot",
		Short: "Fetch weather data and send notifications",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			_ = v.BindPFlag("days", cmd.Flags().Lookup("days"))
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(v, cfgFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			return run(cmd.Context(), cfg)
		},
	}

	cmd.Flags().StringVar(&cfgFile, "config", "", "path to config file (YAML or JSON)")
	cmd.Flags().Int("days", 0, "number of forecast days (overrides config / env)")

	return cmd
}

func run(ctx context.Context, cfg *config.Config) error {
	fetchers := buildFetchers(cfg)
	if len(fetchers) == 0 {
		fetchers = append(fetchers, openmeteo.New())
	}

	rules := make([]application.Rule, 0, len(cfg.Rules))
	for _, r := range cfg.Rules {
		params := r.Params
		// Inject the global rain threshold into rain rules that don't override it.
		if r.Type == config.RuleTypeRain {
			if _, hasOverride := params["threshold"]; !hasOverride {
				if params == nil {
					params = make(map[string]string)
				} else {
					// copy to avoid mutating the config
					copied := make(map[string]string, len(params))
					for k, v := range params {
						copied[k] = v
					}
					params = copied
				}
				params["threshold"] = strconv.Itoa(cfg.Forecast.RainThreshold)
			}
		}
		rules = append(rules, application.Rule{
			Name:   r.Name,
			Type:   r.Type,
			Params: params,
		})
	}

	locations := make([]domain.Location, 0, len(cfg.Locations))
	for _, l := range cfg.Locations {
		locations = append(locations, domain.Location{
			Name:     l.Name,
			Lat:      l.Lat,
			Lon:      l.Lon,
			Timezone: l.Timezone,
		})
	}

	svc := application.NewWeatherService(
		fetchers,
		buildNotifiers(cfg),
		rules,
		application.NewRuleRegistry(),
		locations,
		cfg.Days,
		cfg.Message.Split,
		cfg.Forecast.RainThreshold,
	)

	return svc.Run(ctx)
}

func buildFetchers(cfg *config.Config) []application.WeatherFetcher {
	result := make([]application.WeatherFetcher, 0, len(cfg.Sources))
	for name, src := range cfg.Sources {
		switch name {
		case config.SourceOpenWeatherMap:
			result = append(result, openweathermap.New(src.APIKey))
		case config.SourceOpenMeteo:
			result = append(result, openmeteo.New())
		case config.SourceWeatherAPI:
			result = append(result, weatherapi.New(src.APIKey))
		}
	}
	return result
}

func buildNotifiers(cfg *config.Config) []application.Notifier {
	result := make([]application.Notifier, 0, len(cfg.Notifications))
	for name, n := range cfg.Notifications {
		switch n.Type {
		case config.NotificationTypeTelegram:
			notifier, err := telegram.New(n.Params)
			if err != nil {
				log.Printf("skip notifier %q: %v", name, err)
				continue
			}
			result = append(result, notifier)
		}
	}
	return result
}
