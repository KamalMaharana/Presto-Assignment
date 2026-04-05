// Package repository contains database access logic.
package repository

import (
	"gin-app/internal/models"

	"gorm.io/gorm"
)

type ChargerRepository interface {
	Create(charger *models.Charger) error
	GetByID(chargerID string) (*models.Charger, error)
	List() ([]models.Charger, error)
}

type chargerRepository struct {
	db *gorm.DB
}

func NewChargerRepository(db *gorm.DB) ChargerRepository {
	return &chargerRepository{db: db}
}

func (r *chargerRepository) Create(charger *models.Charger) error {
	return r.db.Create(charger).Error
}

func (r *chargerRepository) GetByID(chargerID string) (*models.Charger, error) {
	var charger models.Charger
	if err := r.db.Where("id = ?", chargerID).First(&charger).Error; err != nil {
		return nil, err
	}

	return &charger, nil
}

func (r *chargerRepository) List() ([]models.Charger, error) {
	var chargers []models.Charger
	if err := r.db.Order("created_at DESC").Find(&chargers).Error; err != nil {
		return nil, err
	}

	return chargers, nil
}
