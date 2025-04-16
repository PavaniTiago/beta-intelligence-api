package entities

import "time"

type Funnel struct {
	FunnelID   int       `json:"funnel_id" gorm:"primaryKey;column:funnel_id"`
	FunnelName string    `json:"funnel_name" gorm:"column:funnel_name"`
	FunnelTag  string    `json:"funnel_tag" gorm:"column:funnel_tag"`
	CreatedAt  time.Time `json:"created_at" gorm:"column:created_at"`
	ProductID  int       `json:"product_id" gorm:"column:product_id"`
	Global     bool      `json:"global" gorm:"column:global"`
	IsTesting  bool      `json:"is_testing" gorm:"column:is_testing"`
	IsActive   bool      `json:"is_active" gorm:"column:is_active"`
	Product    Product   `json:"product" gorm:"foreignKey:ProductID;references:ProductID"`
}
