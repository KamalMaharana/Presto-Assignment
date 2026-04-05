package models

import "time"

type TOURatePeriod struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	ChargerID     string     `gorm:"size:64;not null;index:idx_tou_charger_effective_start,priority:1;index" json:"charger_id"`
	EffectiveFrom time.Time  `gorm:"type:date;not null;index:idx_tou_charger_effective_start,priority:2" json:"effective_from"`
	EffectiveTo   *time.Time `gorm:"type:date;index" json:"effective_to,omitempty"`
	StartMinute   int        `gorm:"not null;index:idx_tou_charger_effective_start,priority:3" json:"start_minute"`
	EndMinute     int        `gorm:"not null" json:"end_minute"`
	PricePerKwh   float64    `gorm:"type:numeric(10,4);not null" json:"price_per_kwh"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
