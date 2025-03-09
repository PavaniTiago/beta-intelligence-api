package entities

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	EventID      uuid.UUID  `json:"event_id" gorm:"type:uuid;primary_key;column:event_id"`
	EventName    string     `json:"event_name" gorm:"column:event_name"`
	PageviewID   uuid.UUID  `json:"pageview_id" gorm:"column:pageview_id"`
	SessionID    uuid.UUID  `json:"session_id" gorm:"column:session_id"`
	EventTime    time.Time  `json:"event_time" gorm:"column:event_time"`
	UserID       string     `json:"user_id" gorm:"column:user_id"`
	ProfessionID int        `json:"profession_id" gorm:"column:profession_id"`
	ProductID    int        `json:"product_id" gorm:"column:product_id"`
	FunnelID     int        `json:"funnel_id" gorm:"column:funnel_id"`
	EventSource  string     `json:"event_source" gorm:"column:event_source"`
	EventType    string     `json:"event_type" gorm:"column:event_type"`
	User         User       `json:"user" gorm:"foreignKey:UserID;references:UserID"`
	Session      Session    `json:"session" gorm:"foreignKey:SessionID;references:ID"`
	Profession   Profession `json:"profession" gorm:"foreignKey:ProfessionID;references:ProfessionID"`
	Product      Product    `json:"product" gorm:"foreignKey:ProductID;references:ProductID"`
	Funnel       Funnel     `json:"funnel" gorm:"foreignKey:FunnelID;references:FunnelID"`
}
