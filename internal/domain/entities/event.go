package entities

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// UtmData representa os dados UTM para compatibilidade com o frontend
type UtmData struct {
	UtmSource   string `json:"utm_source"`
	UtmMedium   string `json:"utm_medium"`
	UtmCampaign string `json:"utm_campaign"`
	UtmContent  string `json:"utm_content"`
	UtmTerm     string `json:"utm_term"`
}

type Event struct {
	EventID         uuid.UUID       `json:"event_id" gorm:"type:uuid;primary_key;column:event_id"`
	EventName       string          `json:"event_name" gorm:"column:event_name"`
	PageviewID      uuid.UUID       `json:"pageview_id" gorm:"column:pageview_id"`
	SessionID       uuid.UUID       `json:"session_id" gorm:"column:session_id"`
	EventTime       time.Time       `json:"event_time" gorm:"column:event_time"`
	UserID          string          `json:"user_id" gorm:"column:user_id"`
	ProfessionID    int             `json:"profession_id" gorm:"column:profession_id"`
	ProductID       int             `json:"product_id" gorm:"column:product_id"`
	FunnelID        int             `json:"funnel_id" gorm:"column:funnel_id"`
	EventSource     string          `json:"event_source" gorm:"column:event_source"`
	EventType       string          `json:"event_type" gorm:"column:event_type"`
	EventProperties json.RawMessage `json:"event_propeties" gorm:"column:event_propeties;type:jsonb"`
	User            User            `json:"user" gorm:"foreignKey:UserID;references:UserID"`
	Session         Session         `json:"session" gorm:"foreignKey:SessionID;references:ID"`
	Profession      Profession      `json:"profession" gorm:"foreignKey:ProfessionID;references:ProfessionID"`
	Product         Product         `json:"product" gorm:"foreignKey:ProductID;references:ProductID"`
	Funnel          Funnel          `json:"funnel" gorm:"foreignKey:FunnelID;references:FunnelID"`

	// Campos UTM do usuário - mesmos nomes que o frontend espera
	UtmSource   string `json:"utmSource" gorm:"-"`
	UtmMedium   string `json:"utmMedium" gorm:"-"`
	UtmCampaign string `json:"utmCampaign" gorm:"-"`
	UtmContent  string `json:"utmContent" gorm:"-"`
	UtmTerm     string `json:"utmTerm" gorm:"-"`

	// Campo UTM data para compatibilidade com o frontend
	UtmData *UtmData `json:"utm_data,omitempty" gorm:"-"`

	// Campos geográficos do usuário
	InitialCountry     string `json:"initialCountry" gorm:"-"`
	InitialCity        string `json:"initialCity" gorm:"-"`
	InitialRegion      string `json:"initialRegion" gorm:"-"`
	InitialIp          string `json:"initialIp" gorm:"-"`
	InitialCountryCode string `json:"initialCountryCode" gorm:"-"`
	InitialZip         string `json:"initialZip" gorm:"-"`

	// Relações com surveys
	Survey         *Survey         `json:"survey,omitempty" gorm:"-"`
	SurveyResponse *SurveyResponse `json:"survey_response,omitempty" gorm:"-"`
}
