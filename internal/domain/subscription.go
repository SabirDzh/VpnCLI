package domain

import "time"

// Subscription — источник профилей по URL.
type Subscription struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	LastUpdated time.Time `json:"lastUpdated,omitempty"`
}
