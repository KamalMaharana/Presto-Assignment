package service

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

func IsNotFoundError(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

type ScheduleRange struct {
	EffectiveFrom string `json:"effective_from"`
	EffectiveTo   string `json:"effective_to,omitempty"`
}

type OverlappingScheduleError struct {
	ChargerID string          `json:"charger_id"`
	Proposed  ScheduleRange   `json:"proposed"`
	Existing  []ScheduleRange `json:"existing"`
}

func (e *OverlappingScheduleError) Error() string {
	return fmt.Sprintf("overlapping effective_from/effective_to already exists for charger %s", e.ChargerID)
}

func IsOverlappingScheduleError(err error) bool {
	var overlapErr *OverlappingScheduleError
	return errors.As(err, &overlapErr)
}

func AsOverlappingScheduleError(err error) *OverlappingScheduleError {
	var overlapErr *OverlappingScheduleError
	if errors.As(err, &overlapErr) {
		return overlapErr
	}

	return nil
}
