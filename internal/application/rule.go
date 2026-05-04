package application

import (
	"strconv"

	"github.com/ducminhgd/weather-bot/internal/domain"
)

// Rule is the application-layer representation of a single rule condition.
// It mirrors config.RuleConfig but is decoupled from the config package.
type Rule struct {
	Name   string
	Type   string
	Params map[string]string
}

// EvaluatorFunc reports whether a rule condition is met for a given day.
type EvaluatorFunc func(day domain.DailyWeather, params map[string]string) bool

// RuleRegistry maps rule type strings to their evaluator functions.
// New rule types are added by calling Register — no existing code needs to change.
type RuleRegistry struct {
	evaluators map[string]EvaluatorFunc
}

// NewRuleRegistry returns a registry pre-loaded with the built-in rule types.
func NewRuleRegistry() *RuleRegistry {
	r := &RuleRegistry{evaluators: make(map[string]EvaluatorFunc)}
	r.Register("rain", evaluateRain)
	r.Register("temp", evaluateTemp)
	return r
}

// Register adds or replaces the evaluator for ruleType.
func (r *RuleRegistry) Register(ruleType string, fn EvaluatorFunc) {
	r.evaluators[ruleType] = fn
}

// Matches reports whether rule fires for day.
func (r *RuleRegistry) Matches(day domain.DailyWeather, rule Rule) bool {
	fn, ok := r.evaluators[rule.Type]
	if !ok {
		return false
	}
	return fn(day, rule.Params)
}

// MatchesAny reports whether any rule fires for any day in data.
func (r *RuleRegistry) MatchesAny(data []domain.DailyWeather, rules []Rule) bool {
	for _, day := range data {
		for _, rule := range rules {
			if r.Matches(day, rule) {
				return true
			}
		}
	}
	return false
}

// ── built-in evaluators ───────────────────────────────────────────────────────

// RainThreshold is the default precipitation-probability threshold (30 %).
const RainThreshold = 0.3

func evaluateRain(day domain.DailyWeather, params map[string]string) bool {
	threshold := RainThreshold
	if v, ok := params["threshold"]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			threshold = f / 100.0
		}
	}
	for _, h := range day.Hourly {
		if h.PrecipitationChance >= threshold || h.PrecipitationMM > 0 {
			return true
		}
	}
	return false
}

func evaluateTemp(day domain.DailyWeather, params map[string]string) bool {
	condition := params["condition"]
	value, err := strconv.ParseFloat(params["value"], 64)
	if err != nil {
		return false
	}
	switch condition {
	case "lte":
		return day.TempMin <= value
	case "gte":
		return day.TempMax >= value
	}
	return false
}
