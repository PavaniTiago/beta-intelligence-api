package entities

import (
	"time"
)

type User struct {
	UserID                    string    `json:"user_id" gorm:"primaryKey;column:user_id"`
	Fullname                  string    `json:"fullname" gorm:"column:fullname"`
	Email                     string    `json:"email" gorm:"column:email"`
	Phone                     string    `json:"phone" gorm:"column:phone"`
	IsClient                  bool      `json:"isClient" gorm:"column:isClient"`
	Fbc                       string    `json:"fbc" gorm:"type:text"`
	Fbp                       string    `json:"fbp" gorm:"type:text"`
	CreatedAt                 time.Time `json:"created_at"`
	IsRecent                  bool      `json:"is_recent"`
	InitialCountry            string    `json:"initialCountry" gorm:"type:text"`
	InitialCountryCode        string    `json:"initialCountryCode" gorm:"type:text"`
	InitialRegion             string    `json:"initialRegion" gorm:"type:text"`
	InitialCity               string    `json:"initialCity" gorm:"type:text"`
	InitialZip                string    `json:"initialZip" gorm:"type:text"`
	InitialIp                 string    `json:"initialIp" gorm:"type:text"`
	InitialUserAgent          string    `json:"initialUserAgent" gorm:"type:text"`
	InitialReferrer           string    `json:"initialReferrer" gorm:"type:text"`
	InitialTimezone           string    `json:"initialTimezone" gorm:"type:text"`
	IsIdentified              bool      `json:"isIdentified"`
	InitialDeviceType         string    `json:"initialDeviceType" gorm:"type:text"`
	InitialPlatform           string    `json:"initialPlatform" gorm:"type:text"`
	InitialBrowser            string    `json:"initialBrowser" gorm:"type:text"`
	InitialLandingPage        string    `json:"initialLandingPage" gorm:"type:text"`
	InitialMarketingChannel   string    `json:"initialMarketingChannel" gorm:"type:text"`
	InitialProfession         string    `json:"initialProfession" gorm:"type:text"`
	InitialFunnel             string    `json:"initialFunnel" gorm:"type:text"`
	InitialUtmSource          string    `json:"initialUtmSource" gorm:"type:text"`
	InitialUtmMedium          string    `json:"initialUtmMedium" gorm:"type:text"`
	InitialUtmCampaign        string    `json:"initialUtmCampaign" gorm:"type:text"`
	InitialUtmContent         string    `json:"initialUtmContent" gorm:"type:text"`
	InitialUtmTerm            string    `json:"initialUtmTerm" gorm:"type:text"`
	InitialLandingSpecialPath string    `json:"initialLandingSpecialPath" gorm:"type:text"`
	InitialReferrerDomain     string    `json:"initialReferrerDomain" gorm:"type:text"`
	InitialReferrerQuery      string    `json:"initialReferrerQuery" gorm:"type:text"`
	InitialReferrerHostname   string    `json:"initialReferrerHostname" gorm:"type:text"`
	InitialReferrerPath       string    `json:"initialReferrerPath" gorm:"type:text"`
}
