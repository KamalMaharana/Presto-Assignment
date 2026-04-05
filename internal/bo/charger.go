package bo

import "time"

type Charger struct {
	ID                 string
	Name               string
	Location           string
	Timezone           string
	DefaultPricePerKWh float64
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type CreateChargerInput struct {
	ID                 string
	Name               string
	Location           string
	Timezone           string
	DefaultPricePerKWh float64
}
