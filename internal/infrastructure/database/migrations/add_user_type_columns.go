package migrations

import (
	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/entity"

	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&entity.User{})
}
