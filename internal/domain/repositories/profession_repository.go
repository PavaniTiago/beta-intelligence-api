package repositories

import (
	"fmt"

	"github.com/PavaniTiago/beta-intelligence/internal/domain/entities"
	"gorm.io/gorm"
)

type ProfessionRepository interface {
	GetProfessions(page, limit int, orderBy string) ([]entities.Profession, int64, error)
}

type professionRepository struct {
	db *gorm.DB
}

func NewProfessionRepository(db *gorm.DB) ProfessionRepository {
	return &professionRepository{db}
}

func (r *professionRepository) GetProfessions(page, limit int, orderBy string) ([]entities.Profession, int64, error) {
	var professions []entities.Profession
	var total int64

	// Get total count
	if err := r.db.Model(&entities.Profession{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	fmt.Printf("Total professions in database: %d\n", total)

	// Get results
	query := r.db.Model(&entities.Profession{})

	if orderBy != "" {
		query = query.Order(orderBy)
	}

	// Se limit for -1, não aplicar paginação
	if limit == -1 {
		result := query.Find(&professions)
		if result.Error != nil {
			return nil, 0, result.Error
		}
	} else {
		// Calculate offset for pagination
		offset := (page - 1) * limit

		result := query.Offset(offset).
			Limit(limit).
			Find(&professions)

		if result.Error != nil {
			return nil, 0, result.Error
		}
	}

	fmt.Printf("Retrieved %d professions from database\n", len(professions))
	for i, p := range professions {
		fmt.Printf("Profession %d: ID=%d, Name=%s\n", i+1, p.ProfessionID, p.ProfessionName)
	}

	return professions, total, nil
}
