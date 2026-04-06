package service

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"gin-app/internal/bo"
	"gin-app/internal/models"

	"gorm.io/gorm"
)

func TestParseDate_PreservesCalendarDateAcrossTimezones(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		timezone string
		want     time.Time
	}{
		{
			name:     "utc timezone",
			input:    "2026-04-15",
			timezone: "UTC",
			want:     time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "positive offset timezone",
			input:    "2026-04-15",
			timezone: "Asia/Kolkata",
			want:     time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "far positive offset timezone",
			input:    "2026-01-01",
			timezone: "Pacific/Auckland",
			want:     time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "negative offset timezone",
			input:    "2026-11-05",
			timezone: "America/Los_Angeles",
			want:     time.Date(2026, time.November, 5, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseDate(tt.input, tt.timezone)
			if err != nil {
				t.Fatalf("parseDate returned error: %v", err)
			}
			if !got.Equal(tt.want) {
				t.Fatalf("expected %s, got %s", tt.want.Format(time.RFC3339), got.Format(time.RFC3339))
			}
			if got.Location() != time.UTC {
				t.Fatalf("expected UTC location, got %s", got.Location().String())
			}
		})
	}
}

func TestParseDate_InvalidInputs(t *testing.T) {
	t.Parallel()

	if _, err := parseDate("2026-99-99", "UTC"); err == nil {
		t.Fatal("expected error for invalid date")
	}

	if _, err := parseDate("2026-04-15", "Mars/Olympus_Mons"); err == nil {
		t.Fatal("expected error for invalid timezone")
	}
}

type replaceCall struct {
	chargerID     string
	effectiveFrom time.Time
	effectiveTo   *time.Time
	periods       []models.TOURatePeriod
}

type fakeChargerRepo struct {
	chargers map[string]models.Charger
	getErr   error
}

func (f *fakeChargerRepo) Create(_ *models.Charger) error {
	return nil
}

func (f *fakeChargerRepo) GetByID(chargerID string) (*models.Charger, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	charger, ok := f.chargers[chargerID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return &charger, nil
}

func (f *fakeChargerRepo) List() ([]models.Charger, error) {
	return nil, nil
}

type fakeTOURepo struct {
	getPeriodsByEffectiveFrom map[string][]models.TOURatePeriod
	getPeriodsErr             error

	overlappingSchedules []models.TOURatePeriod
	overlapErr           error

	replaceErr   error
	replaceCalls []replaceCall

	applicablePeriods       []models.TOURatePeriod
	applicableEffectiveFrom time.Time
	applicableErr           error

	getPeriodResult *models.TOURatePeriod
	getPeriodErr    error

	lastGetPeriodMinute int
}

func (f *fakeTOURepo) ReplaceDailySchedule(chargerID string, effectiveFrom time.Time, effectiveTo *time.Time, periods []models.TOURatePeriod) error {
	f.replaceCalls = append(f.replaceCalls, replaceCall{
		chargerID:     chargerID,
		effectiveFrom: effectiveFrom,
		effectiveTo:   effectiveTo,
		periods:       periods,
	})
	return f.replaceErr
}

func (f *fakeTOURepo) GetPeriodsByEffectiveFrom(chargerID string, effectiveFrom time.Time) ([]models.TOURatePeriod, error) {
	if f.getPeriodsErr != nil {
		return nil, f.getPeriodsErr
	}
	key := fmt.Sprintf("%s|%s", chargerID, effectiveFrom.Format("2006-01-02"))
	return f.getPeriodsByEffectiveFrom[key], nil
}

func (f *fakeTOURepo) ListOverlappingSchedules(_ string, _ time.Time, _ *time.Time, _ *time.Time) ([]models.TOURatePeriod, error) {
	if f.overlapErr != nil {
		return nil, f.overlapErr
	}
	return f.overlappingSchedules, nil
}

func (f *fakeTOURepo) GetApplicablePeriods(_ string, _ time.Time) ([]models.TOURatePeriod, time.Time, error) {
	if f.applicableErr != nil {
		return nil, time.Time{}, f.applicableErr
	}
	return f.applicablePeriods, f.applicableEffectiveFrom, nil
}

func (f *fakeTOURepo) GetPeriodForMinute(_ string, _ time.Time, minute int) (*models.TOURatePeriod, error) {
	f.lastGetPeriodMinute = minute
	if f.getPeriodErr != nil {
		return nil, f.getPeriodErr
	}
	return f.getPeriodResult, nil
}

func TestReplaceSchedule_MergesWithExistingPeriodsAndInheritsEffectiveTo(t *testing.T) {
	t.Parallel()

	effectiveTo := time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC)
	chargerRepo := &fakeChargerRepo{
		chargers: map[string]models.Charger{
			"charger-001": {
				ID:                 "charger-001",
				Timezone:           "UTC",
				DefaultPricePerKwh: 0.22,
			},
		},
	}
	touRepo := &fakeTOURepo{
		getPeriodsByEffectiveFrom: map[string][]models.TOURatePeriod{
			"charger-001|2026-04-01": {
				{StartMinute: 0, EndMinute: 360, PricePerKwh: 0.10, EffectiveTo: &effectiveTo},
				{StartMinute: 360, EndMinute: 720, PricePerKwh: 0.20, EffectiveTo: &effectiveTo},
				{StartMinute: 720, EndMinute: 1440, PricePerKwh: 0.30, EffectiveTo: &effectiveTo},
			},
		},
	}

	svc := NewTOUService(chargerRepo, touRepo)
	err := svc.ReplaceSchedule("charger-001", bo.UpsertTOUScheduleInput{
		EffectiveFrom: "2026-04-01",
		Periods: []bo.TOUPeriod{
			{StartTime: "05:00", EndTime: "08:00", PricePerKwh: 0.50},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(touRepo.replaceCalls) != 1 {
		t.Fatalf("expected one replace call, got %d", len(touRepo.replaceCalls))
	}

	call := touRepo.replaceCalls[0]
	if call.effectiveTo == nil || !call.effectiveTo.Equal(effectiveTo) {
		t.Fatalf("expected inherited effective_to %s, got %v", effectiveTo.Format("2006-01-02"), call.effectiveTo)
	}

	want := []models.TOURatePeriod{
		{StartMinute: 0, EndMinute: 300, PricePerKwh: 0.10},
		{StartMinute: 300, EndMinute: 480, PricePerKwh: 0.50},
		{StartMinute: 480, EndMinute: 720, PricePerKwh: 0.20},
		{StartMinute: 720, EndMinute: 1440, PricePerKwh: 0.30},
	}
	if len(call.periods) != len(want) {
		t.Fatalf("expected %d periods, got %d", len(want), len(call.periods))
	}
	for i := range want {
		if call.periods[i].StartMinute != want[i].StartMinute ||
			call.periods[i].EndMinute != want[i].EndMinute ||
			call.periods[i].PricePerKwh != want[i].PricePerKwh {
			t.Fatalf("unexpected period at index %d: got %+v want %+v", i, call.periods[i], want[i])
		}
	}
}

func TestReplaceSchedule_ReturnsOverlapError(t *testing.T) {
	t.Parallel()

	existingFrom := time.Date(2026, time.January, 6, 0, 0, 0, 0, time.UTC)
	existingTo := time.Date(2026, time.December, 6, 0, 0, 0, 0, time.UTC)

	chargerRepo := &fakeChargerRepo{
		chargers: map[string]models.Charger{
			"charger-001": {ID: "charger-001", Timezone: "UTC"},
		},
	}
	touRepo := &fakeTOURepo{
		overlappingSchedules: []models.TOURatePeriod{
			{EffectiveFrom: existingFrom, EffectiveTo: &existingTo},
		},
	}

	svc := NewTOUService(chargerRepo, touRepo)
	err := svc.ReplaceSchedule("charger-001", bo.UpsertTOUScheduleInput{
		EffectiveFrom: "2026-02-06",
		EffectiveTo:   "2026-11-06",
		Periods: []bo.TOUPeriod{
			{StartTime: "00:00", EndTime: "24:00", PricePerKwh: 0.20},
		},
	})
	if err == nil {
		t.Fatal("expected overlap error, got nil")
	}
	if !IsOverlappingScheduleError(err) {
		t.Fatalf("expected overlapping error type, got %T", err)
	}

	overlapErr := AsOverlappingScheduleError(err)
	if overlapErr == nil {
		t.Fatal("expected overlap error details")
	}
	if overlapErr.ChargerID != "charger-001" {
		t.Fatalf("unexpected charger id: %s", overlapErr.ChargerID)
	}
	if overlapErr.Proposed.EffectiveFrom != "2026-02-06" || overlapErr.Proposed.EffectiveTo != "2026-11-06" {
		t.Fatalf("unexpected proposed range: %+v", overlapErr.Proposed)
	}
	wantExisting := []ScheduleRange{{EffectiveFrom: "2026-01-06", EffectiveTo: "2026-12-06"}}
	if !reflect.DeepEqual(overlapErr.Existing, wantExisting) {
		t.Fatalf("unexpected existing ranges: got %+v want %+v", overlapErr.Existing, wantExisting)
	}
	if len(touRepo.replaceCalls) != 0 {
		t.Fatalf("expected replace not to be called on overlap, got %d calls", len(touRepo.replaceCalls))
	}
}

func TestReplaceSchedule_InvalidInputs(t *testing.T) {
	t.Parallel()

	chargerRepo := &fakeChargerRepo{
		chargers: map[string]models.Charger{
			"charger-001": {ID: "charger-001", Timezone: "UTC"},
		},
	}
	touRepo := &fakeTOURepo{}
	svc := NewTOUService(chargerRepo, touRepo)

	testCases := []struct {
		name    string
		input   bo.UpsertTOUScheduleInput
		wantErr string
	}{
		{
			name: "invalid effective_from format",
			input: bo.UpsertTOUScheduleInput{
				EffectiveFrom: "06-02-2026",
				Periods:       []bo.TOUPeriod{{StartTime: "00:00", EndTime: "24:00", PricePerKwh: 0.2}},
			},
			wantErr: "effective_from must be YYYY-MM-DD",
		},
		{
			name: "effective_to before effective_from",
			input: bo.UpsertTOUScheduleInput{
				EffectiveFrom: "2026-06-10",
				EffectiveTo:   "2026-06-09",
				Periods:       []bo.TOUPeriod{{StartTime: "00:00", EndTime: "24:00", PricePerKwh: 0.2}},
			},
			wantErr: "effective_to must be greater than or equal to effective_from",
		},
		{
			name: "no periods",
			input: bo.UpsertTOUScheduleInput{
				EffectiveFrom: "2026-06-10",
				Periods:       []bo.TOUPeriod{},
			},
			wantErr: "at least one period is required",
		},
		{
			name: "non positive price",
			input: bo.UpsertTOUScheduleInput{
				EffectiveFrom: "2026-06-10",
				Periods:       []bo.TOUPeriod{{StartTime: "00:00", EndTime: "24:00", PricePerKwh: 0}},
			},
			wantErr: "price_per_kwh must be greater than 0",
		},
		{
			name: "invalid start time",
			input: bo.UpsertTOUScheduleInput{
				EffectiveFrom: "2026-06-10",
				Periods:       []bo.TOUPeriod{{StartTime: "25:00", EndTime: "24:00", PricePerKwh: 0.2}},
			},
			wantErr: "invalid start_time",
		},
		{
			name: "end not greater than start",
			input: bo.UpsertTOUScheduleInput{
				EffectiveFrom: "2026-06-10",
				Periods:       []bo.TOUPeriod{{StartTime: "10:00", EndTime: "09:00", PricePerKwh: 0.2}},
			},
			wantErr: "end_time must be greater than start_time",
		},
		{
			name: "incoming periods overlap",
			input: bo.UpsertTOUScheduleInput{
				EffectiveFrom: "2026-06-10",
				Periods: []bo.TOUPeriod{
					{StartTime: "00:00", EndTime: "12:00", PricePerKwh: 0.2},
					{StartTime: "11:00", EndTime: "24:00", PricePerKwh: 0.3},
				},
			},
			wantErr: "incoming periods must not overlap each other",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := svc.ReplaceSchedule("charger-001", tc.input)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}

func TestGetScheduleByDate_Success(t *testing.T) {
	t.Parallel()

	effectiveTo := time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC)
	chargerRepo := &fakeChargerRepo{
		chargers: map[string]models.Charger{
			"charger-001": {
				ID:                 "charger-001",
				Timezone:           "America/Los_Angeles",
				DefaultPricePerKwh: 0.19,
			},
		},
	}
	touRepo := &fakeTOURepo{
		applicableEffectiveFrom: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC),
		applicablePeriods: []models.TOURatePeriod{
			{StartMinute: 0, EndMinute: 360, PricePerKwh: 0.15, EffectiveTo: &effectiveTo},
			{StartMinute: 360, EndMinute: 1440, PricePerKwh: 0.25, EffectiveTo: &effectiveTo},
		},
	}

	svc := NewTOUService(chargerRepo, touRepo)
	schedule, err := svc.GetScheduleByDate("charger-001", "2026-04-15")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if schedule == nil {
		t.Fatal("expected schedule, got nil")
	}
	if schedule.EffectiveFrom != "2026-04-01" || schedule.EffectiveTo != "2026-12-31" {
		t.Fatalf("unexpected effective range: %s - %s", schedule.EffectiveFrom, schedule.EffectiveTo)
	}
	if len(schedule.Periods) != 2 {
		t.Fatalf("expected 2 periods, got %d", len(schedule.Periods))
	}
	if schedule.Periods[1].StartTime != "06:00" || schedule.Periods[1].EndTime != "24:00" {
		t.Fatalf("unexpected period mapping: %+v", schedule.Periods[1])
	}
}

func TestGetScheduleByDate_ReturnsDefaultWhenNoTOUSchedule(t *testing.T) {
	t.Parallel()

	chargerRepo := &fakeChargerRepo{
		chargers: map[string]models.Charger{
			"charger-001": {
				ID:                 "charger-001",
				Timezone:           "UTC",
				DefaultPricePerKwh: 0.21,
			},
		},
	}
	touRepo := &fakeTOURepo{
		applicableErr: gorm.ErrRecordNotFound,
	}

	svc := NewTOUService(chargerRepo, touRepo)
	schedule, err := svc.GetScheduleByDate("charger-001", "2026-04-15")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if schedule == nil {
		t.Fatal("expected schedule response, got nil")
	}
	if len(schedule.Periods) != 0 {
		t.Fatalf("expected empty periods, got %d", len(schedule.Periods))
	}
	if schedule.DefaultPricePerKwh != 0.21 {
		t.Fatalf("expected default price 0.21, got %v", schedule.DefaultPricePerKwh)
	}
}

func TestGetRateAt_Success(t *testing.T) {
	t.Parallel()

	effectiveFrom := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)
	chargerRepo := &fakeChargerRepo{
		chargers: map[string]models.Charger{
			"charger-001": {
				ID:                 "charger-001",
				Timezone:           "UTC",
				DefaultPricePerKwh: 0.18,
			},
		},
	}
	touRepo := &fakeTOURepo{
		getPeriodResult: &models.TOURatePeriod{
			StartMinute:   840,
			EndMinute:     1080,
			PricePerKwh:   0.30,
			EffectiveFrom: effectiveFrom,
		},
	}

	svc := NewTOUService(chargerRepo, touRepo)
	rate, err := svc.GetRateAt("charger-001", "2026-04-15", "14:30")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rate == nil {
		t.Fatal("expected rate response, got nil")
	}
	if rate.DefaultApplied {
		t.Fatal("expected non-default rate, got default applied")
	}
	if rate.PricePerKwh != 0.30 || rate.PeriodStart != "14:00" || rate.PeriodEnd != "18:00" {
		t.Fatalf("unexpected rate response: %+v", rate)
	}
	if touRepo.lastGetPeriodMinute != 14*60+30 {
		t.Fatalf("expected minute 870, got %d", touRepo.lastGetPeriodMinute)
	}
}

func TestGetRateAt_DefaultFallbackWhenNoPeriod(t *testing.T) {
	t.Parallel()

	chargerRepo := &fakeChargerRepo{
		chargers: map[string]models.Charger{
			"charger-001": {
				ID:                 "charger-001",
				Timezone:           "UTC",
				DefaultPricePerKwh: 0.18,
			},
		},
	}
	touRepo := &fakeTOURepo{
		getPeriodErr: gorm.ErrRecordNotFound,
	}

	svc := NewTOUService(chargerRepo, touRepo)
	rate, err := svc.GetRateAt("charger-001", "2026-04-15", "14:30")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rate == nil {
		t.Fatal("expected rate response, got nil")
	}
	if !rate.DefaultApplied {
		t.Fatal("expected default fallback to be applied")
	}
	if rate.PricePerKwh != 0.18 {
		t.Fatalf("expected default price 0.18, got %v", rate.PricePerKwh)
	}
}

func TestGetRateAt_InvalidInputsAndErrors(t *testing.T) {
	t.Parallel()

	chargerRepo := &fakeChargerRepo{
		chargers: map[string]models.Charger{
			"charger-001": {ID: "charger-001", Timezone: "UTC"},
		},
	}
	touRepo := &fakeTOURepo{}
	svc := NewTOUService(chargerRepo, touRepo)

	if _, err := svc.GetRateAt("charger-001", "15-04-2026", "14:30"); err == nil || !strings.Contains(err.Error(), "date must be YYYY-MM-DD") {
		t.Fatalf("expected invalid date error, got %v", err)
	}

	if _, err := svc.GetRateAt("charger-001", "2026-04-15", "2:30 PM"); err == nil || !strings.Contains(err.Error(), "time must be HH:MM") {
		t.Fatalf("expected invalid time error, got %v", err)
	}

	touRepo.getPeriodErr = errors.New("db unavailable")
	if _, err := svc.GetRateAt("charger-001", "2026-04-15", "14:30"); err == nil || !strings.Contains(err.Error(), "db unavailable") {
		t.Fatalf("expected repository error passthrough, got %v", err)
	}
}
