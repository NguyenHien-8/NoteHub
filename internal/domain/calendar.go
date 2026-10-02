package domain

// CalendarDay is a compact month summary. Date uses YYYY-MM-DD in the caller's
// requested location.
type CalendarDay struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}
