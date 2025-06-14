package usecases

import (
	"time"

	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/repositories"
)

// RevenueUseCase interface para casos de uso de faturamento
type RevenueUseCase interface {
	GetUnifiedDataByProfession(from, to time.Time, professionIDs []int) ([]repositories.UnifiedData, error)
	GetUnifiedDataGeneral(from, to time.Time) (repositories.UnifiedData, error)

	// Novos métodos para dados comparativos
	GetRevenueComparisonGeneral(currentFrom, currentTo, previousFrom, previousTo time.Time) (repositories.RevenueComparisonData, error)
	GetRevenueComparisonByProfession(currentFrom, currentTo, previousFrom, previousTo time.Time, professionIDs []int) ([]repositories.RevenueComparisonData, error)

	// Método para dados por hora
	GetHourlyRevenueData(date time.Time, professionIDs []int) (*repositories.HourlyRevenueMetrics, error)
}

type revenueUseCase struct {
	revenueRepo repositories.RevenueRepository
}

func NewRevenueUseCase(revenueRepo repositories.RevenueRepository) RevenueUseCase {
	return &revenueUseCase{
		revenueRepo: revenueRepo,
	}
}

func (uc *revenueUseCase) GetUnifiedDataByProfession(from, to time.Time, professionIDs []int) ([]repositories.UnifiedData, error) {
	return uc.revenueRepo.GetUnifiedDataByProfession(from, to, professionIDs)
}

func (uc *revenueUseCase) GetUnifiedDataGeneral(from, to time.Time) (repositories.UnifiedData, error) {
	return uc.revenueRepo.GetUnifiedDataGeneral(from, to)
}

func (uc *revenueUseCase) GetRevenueComparisonGeneral(currentFrom, currentTo, previousFrom, previousTo time.Time) (repositories.RevenueComparisonData, error) {
	return uc.revenueRepo.GetRevenueComparisonGeneral(currentFrom, currentTo, previousFrom, previousTo)
}

func (uc *revenueUseCase) GetRevenueComparisonByProfession(currentFrom, currentTo, previousFrom, previousTo time.Time, professionIDs []int) ([]repositories.RevenueComparisonData, error) {
	return uc.revenueRepo.GetRevenueComparisonByProfession(currentFrom, currentTo, previousFrom, previousTo, professionIDs)
}

func (uc *revenueUseCase) GetHourlyRevenueData(date time.Time, professionIDs []int) (*repositories.HourlyRevenueMetrics, error) {
	return uc.revenueRepo.GetHourlyRevenueData(date, professionIDs)
}
