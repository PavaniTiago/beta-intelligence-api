package entities

import "time"

type Product struct {
	ProductID    int       `json:"product_id" gorm:"primaryKey;column:product_id"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at"`
	ProductName  string    `json:"product_name" gorm:"column:product_name"`
	ProfessionID int       `json:"profession_id" gorm:"column:profession_id"`
}
