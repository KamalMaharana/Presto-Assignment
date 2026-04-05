// Package bo contains business objects for the TOU service.
package bo

type TOUPeriod struct {
	StartTime   string
	EndTime     string
	PricePerKwh float64
}

type UpsertTOUScheduleInput struct {
	EffectiveFrom string
	EffectiveTo   string
	Periods       []TOUPeriod
}

type TOUSchedule struct {
	ChargerID          string
	Timezone           string
	DefaultPricePerKwh float64
	EffectiveFrom      string
	EffectiveTo        string
	Periods            []TOUPeriod
}

type TOURateAtTime struct {
	ChargerID          string
	Timezone           string
	DefaultPricePerKwh float64
	DefaultApplied     bool
	Date               string
	Time               string
	PricePerKwh        float64
	PeriodStart        string
	PeriodEnd          string
	EffectiveFrom      string
}
