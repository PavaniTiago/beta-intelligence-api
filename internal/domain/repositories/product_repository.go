package repositories

import (
	"fmt"

	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/entities"
	"gorm.io/gorm"
)

type ProductRepository interface {
	GetProductsWithFunnels(page, limit int, orderBy string) ([]entities.Product, int64, error)
	GetFunnelsByProfessionID(professionID int) ([]entities.Funnel, error)
}

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) ProductRepository {
	return &productRepository{db}
}

func (r *productRepository) GetProductsWithFunnels(page, limit int, orderBy string) ([]entities.Product, int64, error) {
	var products []entities.Product
	var total int64

	// Get total count of products
	if err := r.db.Model(&entities.Product{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	fmt.Printf("Total products in database: %d\n", total)

	// Calculate offset for pagination
	offset := (page - 1) * limit

	// Get products with preloaded funnels
	query := r.db.Model(&entities.Product{})

	// Apply ordering if provided
	if orderBy != "" {
		query = query.Order(orderBy)
	}

	// Fetch paginated products and preload funnels with a condition
	// Only preload funnels where is_testing = false
	result := query.Offset(offset).
		Limit(limit).
		Preload("Funnels", "is_testing = ?", false).
		Find(&products)

	if result.Error != nil {
		return nil, 0, result.Error
	}

	fmt.Printf("Retrieved %d products from database\n", len(products))

	return products, total, nil
}

// GetFunnelsByProfessionID retrieves all funnels for products associated with a given profession_id
func (r *productRepository) GetFunnelsByProfessionID(professionID int) ([]entities.Funnel, error) {
	var funnels []entities.Funnel

	// Query to find funnels through products linked to the given profession_id
	// Directly joins products and funnels to efficiently filter
	err := r.db.Table("funnels").
		Select("funnels.*").
		Joins("JOIN products ON funnels.product_id = products.product_id").
		Where("products.profession_id = ? AND funnels.is_testing = ?", professionID, false).
		Find(&funnels).Error

	if err != nil {
		return nil, err
	}

	fmt.Printf("Found %d funnels for profession_id %d\n", len(funnels), professionID)
	return funnels, nil
}
