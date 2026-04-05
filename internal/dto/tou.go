package dto

type TOUPeriod struct {
	StartTime   string  `json:"start_time"`
	EndTime     string  `json:"end_time"`
	PricePerKwh float64 `json:"price_per_kwh"`
}

type UpsertTOUScheduleRequest struct {
	EffectiveFrom string      `json:"effective_from"`
	EffectiveTo   string      `json:"effective_to,omitempty"`
	Periods       []TOUPeriod `json:"periods"`
}

type TOUScheduleResponse struct {
	ChargerID          string      `json:"charger_id"`
	Timezone           string      `json:"timezone"`
	DefaultPricePerKwh float64     `json:"default_price_per_kwh"`
	EffectiveFrom      string      `json:"effective_from,omitempty"`
	EffectiveTo        string      `json:"effective_to,omitempty"`
	Periods            []TOUPeriod `json:"periods"`
}

type TOURateAtTimeResponse struct {
	ChargerID          string  `json:"charger_id"`
	Timezone           string  `json:"timezone"`
	DefaultPricePerKwh float64 `json:"default_price_per_kwh"`
	DefaultApplied     bool    `json:"default_applied"`
	Date               string  `json:"date"`
	Time               string  `json:"time"`
	PricePerKwh        float64 `json:"price_per_kwh"`
	PeriodStart        string  `json:"period_start,omitempty"`
	PeriodEnd          string  `json:"period_end,omitempty"`
	EffectiveFrom      string  `json:"effective_from,omitempty"`
}
