package usecases

import (
	"github.com/PavaniTiago/beta-intelligence/internal/domain/entities"
	"github.com/PavaniTiago/beta-intelligence/internal/domain/repositories"
)

type ProfessionUseCase interface {
	GetProfessions(page, limit int, orderBy string) ([]entities.Profession, int64, error)
}

type professionUseCase struct {
	professionRepo repositories.ProfessionRepository
}

func NewProfessionUseCase(professionRepo repositories.ProfessionRepository) ProfessionUseCase {
	return &professionUseCase{professionRepo}
}

func (uc *professionUseCase) GetProfessions(page, limit int, orderBy string) ([]entities.Profession, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if orderBy == "" {
		orderBy = "created_at desc"
	}

	return uc.professionRepo.GetProfessions(page, limit, orderBy)
}
