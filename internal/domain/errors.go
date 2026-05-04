package domain

import "errors"

var (
	// ErrNoData is returned when a weather source returns an empty response.
	ErrNoData = errors.New("no weather data returned")

	// ErrInvalidLocation is returned when the location coordinates are out of range.
	ErrInvalidLocation = errors.New("invalid location: lat must be -90..90, lon must be -180..180")
)
