// Package service contains core business logic for the application.
package service

import (
	"errors"
	"strings"
	"time"

	"gin-app/internal/bo"
	"gin-app/internal/mapper"
	"gin-app/internal/models"
	"gin-app/internal/repository"

	"github.com/google/uuid"
)

type ChargerService interface {
	Create(input bo.CreateChargerInput) (*bo.Charger, error)
	GetByID(chargerID string) (*bo.Charger, error)
	List() ([]bo.Charger, error)
}

type chargerService struct {
	repo repository.ChargerRepository
}

const defaultChargerPricePerKWh = 0.20

func NewChargerService(repo repository.ChargerRepository) ChargerService {
	return &chargerService{repo: repo}
}

func (s *chargerService) Create(input bo.CreateChargerInput) (*bo.Charger, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}

	timezone := strings.TrimSpace(input.Timezone)
	if timezone == "" {
		timezone = "UTC"
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return nil, errors.New("invalid timezone")
	}

	defaultPrice := input.DefaultPricePerKWh
	if defaultPrice < 0 {
		return nil, errors.New("default_price_per_kwh cannot be negative")
	}
	if defaultPrice == 0 {
		defaultPrice = defaultChargerPricePerKWh
	}

	chargerID := strings.TrimSpace(input.ID)
	if chargerID == "" {
		chargerID = uuid.NewString()
	}

	charger := &models.Charger{
		ID:                 chargerID,
		Name:               name,
		Location:           strings.TrimSpace(input.Location),
		Timezone:           timezone,
		DefaultPricePerKwh: defaultPrice,
	}

	if err := s.repo.Create(charger); err != nil {
		return nil, err
	}

	mapped := mapper.ToBOCharger(*charger)
	return &mapped, nil
}

func (s *chargerService) GetByID(chargerID string) (*bo.Charger, error) {
	chargerID = strings.TrimSpace(chargerID)
	if chargerID == "" {
		return nil, errors.New("charger_id is required")
	}

	charger, err := s.repo.GetByID(chargerID)
	if err != nil {
		return nil, err
	}

	mapped := mapper.ToBOCharger(*charger)
	return &mapped, nil
}

func (s *chargerService) List() ([]bo.Charger, error) {
	chargers, err := s.repo.List()
	if err != nil {
		return nil, err
	}

	result := make([]bo.Charger, 0, len(chargers))
	for _, charger := range chargers {
		result = append(result, mapper.ToBOCharger(charger))
	}

	return result, nil
}
