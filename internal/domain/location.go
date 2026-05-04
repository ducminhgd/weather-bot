package domain

// Location is a named geographic point with optional display timezone.
type Location struct {
	Name     string
	Lat      float64
	Lon      float64
	Timezone string // IANA timezone name, e.g. "Asia/Ho_Chi_Minh". Empty means UTC.
}
