package service

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gin-app/internal/bo"
	"gin-app/internal/models"
	"gin-app/internal/repository"
)

type TOUService interface {
	ReplaceSchedule(chargerID string, input bo.UpsertTOUScheduleInput) error
	GetScheduleByDate(chargerID string, date string) (*bo.TOUSchedule, error)
	GetRateAt(chargerID string, date string, atTime string) (*bo.TOURateAtTime, error)
}

type touService struct {
	chargerRepo repository.ChargerRepository
	touRepo     repository.TOURepository
}

func NewTOUService(chargerRepo repository.ChargerRepository, touRepo repository.TOURepository) TOUService {
	return &touService{chargerRepo: chargerRepo, touRepo: touRepo}
}

func (s *touService) ReplaceSchedule(chargerID string, input bo.UpsertTOUScheduleInput) error {
	charger, err := s.chargerRepo.GetByID(chargerID)
	if err != nil {
		return err
	}

	effectiveFrom, err := parseDate(input.EffectiveFrom, charger.Timezone)
	if err != nil {
		return errors.New("effective_from must be YYYY-MM-DD")
	}

	var effectiveTo *time.Time
	if strings.TrimSpace(input.EffectiveTo) != "" {
		parsedTo, parseErr := parseDate(input.EffectiveTo, charger.Timezone)
		if parseErr != nil {
			return errors.New("effective_to must be YYYY-MM-DD")
		}
		if parsedTo.Before(effectiveFrom) {
			return errors.New("effective_to must be greater than or equal to effective_from")
		}
		effectiveTo = &parsedTo
	}

	incoming, err := validateAndConvertPeriods(chargerID, effectiveFrom, effectiveTo, input.Periods)
	if err != nil {
		return err
	}

	existing, err := s.touRepo.GetPeriodsByEffectiveFrom(chargerID, effectiveFrom)
	if err != nil {
		return err
	}

	if effectiveTo == nil && len(existing) > 0 {
		effectiveTo = existing[0].EffectiveTo
	}

	merged := mergePeriods(existing, incoming, chargerID, effectiveFrom, effectiveTo)
	if len(merged) == 0 {
		return errors.New("at least one period is required")
	}

	return s.touRepo.ReplaceDailySchedule(chargerID, effectiveFrom, effectiveTo, merged)
}

func (s *touService) GetScheduleByDate(chargerID string, date string) (*bo.TOUSchedule, error) {
	charger, err := s.chargerRepo.GetByID(chargerID)
	if err != nil {
		return nil, err
	}

	selectedDate, err := parseDate(date, charger.Timezone)
	if err != nil {
		return nil, errors.New("date must be YYYY-MM-DD")
	}

	periods, effectiveFrom, err := s.touRepo.GetApplicablePeriods(chargerID, selectedDate)
	if err != nil {
		if IsNotFoundError(err) {
			return &bo.TOUSchedule{
				ChargerID:          chargerID,
				Timezone:           charger.Timezone,
				DefaultPricePerKwh: charger.DefaultPricePerKwh,
				Periods:            []bo.TOUPeriod{},
			}, nil
		}

		return nil, err
	}

	resp := &bo.TOUSchedule{
		ChargerID:          chargerID,
		Timezone:           charger.Timezone,
		DefaultPricePerKwh: charger.DefaultPricePerKwh,
		EffectiveFrom:      effectiveFrom.Format("2006-01-02"),
		Periods:            make([]bo.TOUPeriod, 0, len(periods)),
	}

	if len(periods) > 0 && periods[0].EffectiveTo != nil {
		resp.EffectiveTo = periods[0].EffectiveTo.Format("2006-01-02")
	}

	for _, period := range periods {
		resp.Periods = append(resp.Periods, bo.TOUPeriod{
			StartTime:   minuteToClock(period.StartMinute),
			EndTime:     minuteToClock(period.EndMinute),
			PricePerKwh: period.PricePerKwh,
		})
	}

	return resp, nil
}

func (s *touService) GetRateAt(chargerID string, date string, atTime string) (*bo.TOURateAtTime, error) {
	charger, err := s.chargerRepo.GetByID(chargerID)
	if err != nil {
		return nil, err
	}

	selectedDate, err := parseDate(date, charger.Timezone)
	if err != nil {
		return nil, errors.New("date must be YYYY-MM-DD")
	}

	minute, err := parseClock(atTime, false)
	if err != nil {
		return nil, errors.New("time must be HH:MM")
	}

	period, err := s.touRepo.GetPeriodForMinute(chargerID, selectedDate, minute)
	if err != nil {
		if IsNotFoundError(err) {
			return &bo.TOURateAtTime{
				ChargerID:          chargerID,
				Timezone:           charger.Timezone,
				DefaultPricePerKwh: charger.DefaultPricePerKwh,
				DefaultApplied:     true,
				Date:               date,
				Time:               atTime,
				PricePerKwh:        charger.DefaultPricePerKwh,
			}, nil
		}

		return nil, err
	}

	return &bo.TOURateAtTime{
		ChargerID:          chargerID,
		Timezone:           charger.Timezone,
		DefaultPricePerKwh: charger.DefaultPricePerKwh,
		DefaultApplied:     false,
		Date:               date,
		Time:               atTime,
		PricePerKwh:        period.PricePerKwh,
		PeriodStart:        minuteToClock(period.StartMinute),
		PeriodEnd:          minuteToClock(period.EndMinute),
		EffectiveFrom:      period.EffectiveFrom.Format("2006-01-02"),
	}, nil
}

func validateAndConvertPeriods(chargerID string, effectiveFrom time.Time, effectiveTo *time.Time, inputs []bo.TOUPeriod) ([]models.TOURatePeriod, error) {
	if len(inputs) == 0 {
		return nil, errors.New("at least one period is required")
	}

	periods := make([]models.TOURatePeriod, 0, len(inputs))
	for _, input := range inputs {
		if input.PricePerKwh <= 0 {
			return nil, errors.New("price_per_kwh must be greater than 0")
		}

		startMinute, err := parseClock(input.StartTime, false)
		if err != nil {
			return nil, fmt.Errorf("invalid start_time: %s", input.StartTime)
		}
		endMinute, err := parseClock(input.EndTime, true)
		if err != nil {
			return nil, fmt.Errorf("invalid end_time: %s", input.EndTime)
		}
		if endMinute <= startMinute {
			return nil, errors.New("end_time must be greater than start_time")
		}

		periods = append(periods, models.TOURatePeriod{
			ChargerID:     chargerID,
			EffectiveFrom: effectiveFrom,
			EffectiveTo:   effectiveTo,
			StartMinute:   startMinute,
			EndMinute:     endMinute,
			PricePerKwh:   input.PricePerKwh,
		})
	}

	sort.Slice(periods, func(i, j int) bool {
		return periods[i].StartMinute < periods[j].StartMinute
	})

	for i := 1; i < len(periods); i++ {
		if periods[i-1].EndMinute > periods[i].StartMinute {
			return nil, errors.New("incoming periods must not overlap each other")
		}
	}

	return periods, nil
}

func mergePeriods(existing []models.TOURatePeriod, incoming []models.TOURatePeriod, chargerID string, effectiveFrom time.Time, effectiveTo *time.Time) []models.TOURatePeriod {
	merged := make([]models.TOURatePeriod, 0, len(existing))
	for _, period := range existing {
		merged = append(merged, models.TOURatePeriod{
			ChargerID:     chargerID,
			EffectiveFrom: effectiveFrom,
			EffectiveTo:   effectiveTo,
			StartMinute:   period.StartMinute,
			EndMinute:     period.EndMinute,
			PricePerKwh:   period.PricePerKwh,
		})
	}

	for _, candidate := range incoming {
		next := make([]models.TOURatePeriod, 0, len(merged)+1)
		for _, current := range merged {
			overlaps := current.StartMinute < candidate.EndMinute && candidate.StartMinute < current.EndMinute
			if !overlaps {
				next = append(next, current)
				continue
			}

			if current.StartMinute < candidate.StartMinute {
				next = append(next, models.TOURatePeriod{
					ChargerID:     chargerID,
					EffectiveFrom: effectiveFrom,
					EffectiveTo:   effectiveTo,
					StartMinute:   current.StartMinute,
					EndMinute:     candidate.StartMinute,
					PricePerKwh:   current.PricePerKwh,
				})
			}

			if current.EndMinute > candidate.EndMinute {
				next = append(next, models.TOURatePeriod{
					ChargerID:     chargerID,
					EffectiveFrom: effectiveFrom,
					EffectiveTo:   effectiveTo,
					StartMinute:   candidate.EndMinute,
					EndMinute:     current.EndMinute,
					PricePerKwh:   current.PricePerKwh,
				})
			}
		}

		next = append(next, models.TOURatePeriod{
			ChargerID:     chargerID,
			EffectiveFrom: effectiveFrom,
			EffectiveTo:   effectiveTo,
			StartMinute:   candidate.StartMinute,
			EndMinute:     candidate.EndMinute,
			PricePerKwh:   candidate.PricePerKwh,
		})
		merged = next
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].StartMinute < merged[j].StartMinute
	})

	return normalizeMergedPeriods(merged, chargerID, effectiveFrom, effectiveTo)
}

func normalizeMergedPeriods(periods []models.TOURatePeriod, chargerID string, effectiveFrom time.Time, effectiveTo *time.Time) []models.TOURatePeriod {
	if len(periods) == 0 {
		return periods
	}

	normalized := make([]models.TOURatePeriod, 0, len(periods))
	current := periods[0]
	for i := 1; i < len(periods); i++ {
		next := periods[i]
		if current.EndMinute == next.StartMinute && current.PricePerKwh == next.PricePerKwh {
			current.EndMinute = next.EndMinute
			continue
		}

		if current.EndMinute > current.StartMinute {
			current.ChargerID = chargerID
			current.EffectiveFrom = effectiveFrom
			current.EffectiveTo = effectiveTo
			normalized = append(normalized, current)
		}
		current = next
	}

	if current.EndMinute > current.StartMinute {
		current.ChargerID = chargerID
		current.EffectiveFrom = effectiveFrom
		current.EffectiveTo = effectiveTo
		normalized = append(normalized, current)
	}

	return normalized
}

func parseDate(value string, timezone string) (time.Time, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, err
	}

	parsed, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(value), location)
	if err != nil {
		return time.Time{}, err
	}

	year, month, day := parsed.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), nil
}

func parseClock(value string, allow24 bool) (int, error) {
	v := strings.TrimSpace(value)
	if allow24 && v == "24:00" {
		return 24 * 60, nil
	}

	parsed, err := time.Parse("15:04", v)
	if err != nil {
		return 0, err
	}

	return parsed.Hour()*60 + parsed.Minute(), nil
}

func minuteToClock(minute int) string {
	if minute == 24*60 {
		return "24:00"
	}

	h := minute / 60
	m := minute % 60
	return fmt.Sprintf("%02d:%02d", h, m)
}
