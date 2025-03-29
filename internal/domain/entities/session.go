package entities

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID                     uuid.UUID  `json:"session_id" gorm:"type:uuid;primary_key;column:session_id"`
	UserID                 uuid.UUID  `json:"user_id" gorm:"type:uuid;column:user_id"`
	SessionStart           time.Time  `json:"session_start" gorm:"column:sessionStart"`
	IsActive               bool       `json:"is_active" gorm:"column:isActive"`
	LastActivity           time.Time  `json:"last_activity" gorm:"column:lastActivity"`
	Country                string     `json:"country" gorm:"column:country"`
	CountryCode            string     `json:"country_code" gorm:"column:countryCode"`
	City                   string     `json:"city" gorm:"column:city"`
	State                  string     `json:"state" gorm:"column:state"`
	StateCode              string     `json:"state_code" gorm:"column:stateCode"`
	Zip                    string     `json:"zip" gorm:"column:zip"`
	IpAddress              string     `json:"ip_address" gorm:"column:ipAddress"`
	UtmSource              string     `json:"utm_source" gorm:"column:utmSource"`
	UtmMedium              string     `json:"utm_medium" gorm:"column:utmMedium"`
	UtmCampaign            string     `json:"utm_campaign" gorm:"column:utmCampaign"`
	UtmContent             string     `json:"utm_content" gorm:"column:utmContent"`
	UtmTerm                string     `json:"utm_term" gorm:"column:utmTerm"`
	UserAgent              string     `json:"user_agent" gorm:"column:userAgent"`
	SessionEnd             *time.Time `json:"session_end" gorm:"column:sessionEnd"`
	Duration               int        `json:"duration" gorm:"column:duration"`
	Fbp                    string     `json:"fbp" gorm:"column:fbp"`
	Fbc                    string     `json:"fbc" gorm:"column:fbc"`
	MarketingChannel       string     `json:"marketing_channel" gorm:"column:marketingChannel"`
	ReferrerPath           string     `json:"referrer_path" gorm:"column:referrerPath"`
	ReferrerHostname       string     `json:"referrer_hostname" gorm:"column:referrerHostname"`
	ReferrerQuery          string     `json:"referrer_query" gorm:"column:referrerQuery"`
	Referrer               string     `json:"referrer" gorm:"column:referrer"`
	LandingPagePath        string     `json:"landing_page_path" gorm:"column:landingPagePath"`
	LandingPageQuery       string     `json:"landing_page_query" gorm:"column:landingPageQuery"`
	LandingPage            string     `json:"landing_page" gorm:"column:landingPage"`
	ProfessionID           *int       `json:"profession_id" gorm:"column:profession_id"`
	ProductID              *int       `json:"product_id" gorm:"column:product_id"`
	FunnelID               *int       `json:"funnel_id" gorm:"column:funnel_id"`
	IsFirstSession         bool       `json:"is_first_session" gorm:"column:is_first_session"`
	IsFirstSessionInFunnel bool       `json:"is_first_session_in_funnel" gorm:"column:is_first_session_in_funnel"`

	// Relações
	User       *User       `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Profession *Profession `json:"profession,omitempty" gorm:"foreignKey:ProfessionID"`
	Product    *Product    `json:"product,omitempty" gorm:"foreignKey:ProductID"`
	Funnel     *Funnel     `json:"funnel,omitempty" gorm:"foreignKey:FunnelID"`
}
