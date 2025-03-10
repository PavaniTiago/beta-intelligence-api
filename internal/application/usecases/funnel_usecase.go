package usecases

import (
	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/entities"
	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/repositories"
)

type FunnelUseCase interface {
	GetFunnels(page, limit int, orderBy string) ([]entities.Funnel, int64, error)
}

type funnelUseCase struct {
	funnelRepo repositories.FunnelRepository
}

func NewFunnelUseCase(funnelRepo repositories.FunnelRepository) FunnelUseCase {
	return &funnelUseCase{funnelRepo}
}

func (uc *funnelUseCase) GetFunnels(page, limit int, orderBy string) ([]entities.Funnel, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if orderBy == "" {
		orderBy = "created_at desc"
	}

	return uc.funnelRepo.GetFunnels(page, limit, orderBy)
}
