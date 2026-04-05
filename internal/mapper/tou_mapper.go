package mapper

import (
	"gin-app/internal/bo"
	"gin-app/internal/dto"
)

func ToBOUpsertTOUScheduleInput(req dto.UpsertTOUScheduleRequest) bo.UpsertTOUScheduleInput {
	periods := make([]bo.TOUPeriod, 0, len(req.Periods))
	for _, period := range req.Periods {
		periods = append(periods, bo.TOUPeriod{
			StartTime:   period.StartTime,
			EndTime:     period.EndTime,
			PricePerKwh: period.PricePerKwh,
		})
	}

	return bo.UpsertTOUScheduleInput{
		EffectiveFrom: req.EffectiveFrom,
		EffectiveTo:   req.EffectiveTo,
		Periods:       periods,
	}
}

func ToDTOTOUScheduleResponse(schedule bo.TOUSchedule) dto.TOUScheduleResponse {
	periods := make([]dto.TOUPeriod, 0, len(schedule.Periods))
	for _, period := range schedule.Periods {
		periods = append(periods, dto.TOUPeriod{
			StartTime:   period.StartTime,
			EndTime:     period.EndTime,
			PricePerKwh: period.PricePerKwh,
		})
	}

	return dto.TOUScheduleResponse{
		ChargerID:          schedule.ChargerID,
		Timezone:           schedule.Timezone,
		DefaultPricePerKwh: schedule.DefaultPricePerKwh,
		EffectiveFrom:      schedule.EffectiveFrom,
		EffectiveTo:        schedule.EffectiveTo,
		Periods:            periods,
	}
}

func ToDTOTOURateAtTimeResponse(rate bo.TOURateAtTime) dto.TOURateAtTimeResponse {
	return dto.TOURateAtTimeResponse{
		ChargerID:          rate.ChargerID,
		Timezone:           rate.Timezone,
		DefaultPricePerKwh: rate.DefaultPricePerKwh,
		DefaultApplied:     rate.DefaultApplied,
		Date:               rate.Date,
		Time:               rate.Time,
		PricePerKwh:        rate.PricePerKwh,
		PeriodStart:        rate.PeriodStart,
		PeriodEnd:          rate.PeriodEnd,
		EffectiveFrom:      rate.EffectiveFrom,
	}
}
