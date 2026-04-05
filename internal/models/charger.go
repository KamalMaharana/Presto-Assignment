// Package models defines database entities.
package models

import "time"

type Charger struct {
	ID                 string    `gorm:"primaryKey;size:64" json:"id"`
	Name               string    `gorm:"size:120;not null" json:"name"`
	Location           string    `gorm:"size:255" json:"location"`
	Timezone           string    `gorm:"size:64;not null;default:UTC" json:"timezone"`
	DefaultPricePerKwh float64   `gorm:"type:numeric(10,4);not null;default:0.2000" json:"default_price_per_kwh"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
