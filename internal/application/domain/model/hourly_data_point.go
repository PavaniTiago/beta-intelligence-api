package model

// HourlyDataPoint represents session and lead data for a specific hour
type HourlyDataPoint struct {
	Hour       int `json:"hour"`
	Sessions   int `json:"sessions"`
	Leads      int `json:"leads"`
	Conversion int `json:"conversion"`
}
