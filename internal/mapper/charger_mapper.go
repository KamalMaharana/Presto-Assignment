package mapper

import (
	"gin-app/internal/bo"
	"gin-app/internal/dto"
	"gin-app/internal/models"
)

func ToBOCreateChargerInput(req dto.CreateChargerRequest) bo.CreateChargerInput {
	return bo.CreateChargerInput{
		ID:                 req.ID,
		Name:               req.Name,
		Location:           req.Location,
		Timezone:           req.Timezone,
		DefaultPricePerKWh: req.DefaultPricePerKWh,
	}
}

func ToBOCharger(model models.Charger) bo.Charger {
	return bo.Charger{
		ID:                 model.ID,
		Name:               model.Name,
		Location:           model.Location,
		Timezone:           model.Timezone,
		DefaultPricePerKWh: model.DefaultPricePerKwh,
		CreatedAt:          model.CreatedAt,
		UpdatedAt:          model.UpdatedAt,
	}
}

func ToDTOChargerResponse(charger bo.Charger) dto.ChargerResponse {
	return dto.ChargerResponse{
		ID:                 charger.ID,
		Name:               charger.Name,
		Location:           charger.Location,
		Timezone:           charger.Timezone,
		DefaultPricePerKWh: charger.DefaultPricePerKWh,
		CreatedAt:          charger.CreatedAt,
		UpdatedAt:          charger.UpdatedAt,
	}
}
