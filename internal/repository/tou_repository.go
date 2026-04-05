// Package repository contains database access logic.
package repository

import (
	"time"

	"gin-app/internal/models"

	"gorm.io/gorm"
)

type TOURepository interface {
	ReplaceDailySchedule(chargerID string, effectiveFrom time.Time, effectiveTo *time.Time, periods []models.TOURatePeriod) error
	GetPeriodsByEffectiveFrom(chargerID string, effectiveFrom time.Time) ([]models.TOURatePeriod, error)
	GetApplicablePeriods(chargerID string, date time.Time) ([]models.TOURatePeriod, time.Time, error)
	GetPeriodForMinute(chargerID string, date time.Time, minute int) (*models.TOURatePeriod, error)
}

type touRepository struct {
	db *gorm.DB
}

func NewTOURepository(db *gorm.DB) TOURepository {
	return &touRepository{db: db}
}

func (r *touRepository) ReplaceDailySchedule(chargerID string, effectiveFrom time.Time, effectiveTo *time.Time, periods []models.TOURatePeriod) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("charger_id = ? AND effective_from = ?", chargerID, effectiveFrom).Delete(&models.TOURatePeriod{}).Error; err != nil {
			return err
		}

		if len(periods) == 0 {
			return nil
		}

		return tx.Create(&periods).Error
	})
}

func (r *touRepository) GetPeriodsByEffectiveFrom(chargerID string, effectiveFrom time.Time) ([]models.TOURatePeriod, error) {
	var periods []models.TOURatePeriod
	if err := r.db.
		Where("charger_id = ? AND effective_from = ?", chargerID, effectiveFrom).
		Order("start_minute ASC").
		Find(&periods).Error; err != nil {
		return nil, err
	}

	return periods, nil
}

func (r *touRepository) GetApplicablePeriods(chargerID string, date time.Time) ([]models.TOURatePeriod, time.Time, error) {
	var latest models.TOURatePeriod
	err := r.db.
		Where("charger_id = ? AND effective_from <= ? AND (effective_to IS NULL OR effective_to >= ?)", chargerID, date, date).
		Order("effective_from DESC").
		First(&latest).Error
	if err != nil {
		return nil, time.Time{}, err
	}

	var periods []models.TOURatePeriod
	if err := r.db.
		Where("charger_id = ? AND effective_from = ?", chargerID, latest.EffectiveFrom).
		Order("start_minute ASC").
		Find(&periods).Error; err != nil {
		return nil, time.Time{}, err
	}

	return periods, latest.EffectiveFrom, nil
}

func (r *touRepository) GetPeriodForMinute(chargerID string, date time.Time, minute int) (*models.TOURatePeriod, error) {
	periods, _, err := r.GetApplicablePeriods(chargerID, date)
	if err != nil {
		return nil, err
	}

	for i := range periods {
		if minute >= periods[i].StartMinute && minute < periods[i].EndMinute {
			return &periods[i], nil
		}
	}

	return nil, gorm.ErrRecordNotFound
}
