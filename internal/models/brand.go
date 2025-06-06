package models

import "time"

// Brand represents a brand entity.
// This will be used for managing different brands in the system.
// Each member data upload will be associated with a brand.
type Brand struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
