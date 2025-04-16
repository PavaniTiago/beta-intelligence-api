package usecases

import (
	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/entities"
	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/repositories"
)

type ProductUseCase interface {
	GetProductsWithFunnels(page, limit int, orderBy string) ([]entities.Product, int64, error)
	GetFunnelsByProfessionID(professionID int) ([]entities.Funnel, error)
}

type productUseCase struct {
	productRepo repositories.ProductRepository
}

func NewProductUseCase(productRepo repositories.ProductRepository) ProductUseCase {
	return &productUseCase{productRepo}
}

func (uc *productUseCase) GetProductsWithFunnels(page, limit int, orderBy string) ([]entities.Product, int64, error) {
	if page < 1 {
		page = 1
	}

	if limit < 1 {
		limit = 10
	}

	if orderBy == "" {
		orderBy = "created_at desc"
	}

	return uc.productRepo.GetProductsWithFunnels(page, limit, orderBy)
}

// GetFunnelsByProfessionID retrieves all funnels for products associated with a given profession_id
func (uc *productUseCase) GetFunnelsByProfessionID(professionID int) ([]entities.Funnel, error) {
	return uc.productRepo.GetFunnelsByProfessionID(professionID)
}
