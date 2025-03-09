package entities

import "time"

type Profession struct {
	ProfessionID   int       `json:"profession_id" gorm:"primaryKey;column:profession_id"`
	CreatedAt      time.Time `json:"created_at" gorm:"column:created_at"`
	ProfessionName string    `json:"profession_name" gorm:"column:profession_name"`
	MetaPixel      string    `json:"meta_pixel" gorm:"column:meta_pixel"`
	MetaToken      string    `json:"meta_token" gorm:"column:meta_token"`
}
