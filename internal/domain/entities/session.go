package entities

import "github.com/google/uuid"

type Session struct {
	ID          uuid.UUID `json:"session_id" gorm:"type:uuid;primary_key;column:session_id"`
	UtmSource   string    `json:"utm_source" gorm:"column:utmSource"`
	UtmMedium   string    `json:"utm_medium" gorm:"column:utmMedium"`
	UtmCampaign string    `json:"utm_campaign" gorm:"column:utmCampaign"`
	UtmContent  string    `json:"utm_content" gorm:"column:utmContent"`
	UtmTerm     string    `json:"utm_term" gorm:"column:utmTerm"`
	Country     string    `json:"country" gorm:"column:country"`
	State       string    `json:"state" gorm:"column:state"`
	City        string    `json:"city" gorm:"column:city"`
}
