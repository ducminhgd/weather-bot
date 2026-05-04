package config

// Rule type constants — values for RuleConfig.Type.
const (
	RuleTypeRain = "rain"
	RuleTypeTemp = "temp"
)

// Rule condition constants — values for RuleConfig.Params["condition"].
const (
	RuleConditionLTE = "lte" // alert when value ≤ threshold
	RuleConditionGTE = "gte" // alert when value ≥ threshold
)

// Source name constants — keys used in Config.Sources.
const (
	SourceOpenWeatherMap = "openweathermap"
	SourceOpenMeteo      = "open-meteo"
)

// Notification type constants — values for NotificationConfig.Type.
const (
	NotificationTypeTelegram = "telegram"
)
