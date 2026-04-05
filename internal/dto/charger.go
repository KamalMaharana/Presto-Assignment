// Package dto contains data transfer objects for the application.
package dto

import "time"

type CreateChargerRequest struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	Location           string  `json:"location"`
	Timezone           string  `json:"timezone"`
	DefaultPricePerKWh float64 `json:"default_price_per_kwh"`
}

type ChargerResponse struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Location           string    `json:"location"`
	Timezone           string    `json:"timezone"`
	DefaultPricePerKWh float64   `json:"default_price_per_kwh"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
