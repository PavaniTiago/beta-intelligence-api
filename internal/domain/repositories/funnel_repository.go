package repositories

import (
	"github.com/PavaniTiago/beta-intelligence/internal/domain/entities"
	"gorm.io/gorm"
)

type FunnelRepository interface {
	GetFunnels(page, limit int, orderBy string) ([]entities.Funnel, int64, error)
}

type funnelRepository struct {
	db *gorm.DB
}

func NewFunnelRepository(db *gorm.DB) FunnelRepository {
	return &funnelRepository{db}
}

func (r *funnelRepository) GetFunnels(page, limit int, orderBy string) ([]entities.Funnel, int64, error) {
	var funnels []entities.Funnel
	var total int64

	// Get total count
	if err := r.db.Model(&entities.Funnel{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Get paginated results with product relation
	query := r.db.Model(&entities.Funnel{}).
		Select(`
			funnels.*,
			products.product_name,
			products.created_at as product_created_at,
			products.profession_id as product_profession_id
		`).
		Joins("LEFT JOIN products ON funnels.product_id = products.product_id").
		Preload("Product")

	if orderBy != "" {
		query = query.Order(orderBy)
	}

	result := query.Offset(offset).
		Limit(limit).
		Find(&funnels)

	if result.Error != nil {
		return nil, 0, result.Error
	}

	return funnels, total, nil
}
