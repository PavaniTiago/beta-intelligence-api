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
	InitialCountry            string    `json:"initialCountry" gorm:"column:initialCountry;type:text"`
	InitialCountryCode        string    `json:"initialCountryCode" gorm:"column:initialCountryCode;type:text"`
	InitialRegion             string    `json:"initialRegion" gorm:"column:initialRegion;type:text"`
	InitialCity               string    `json:"initialCity" gorm:"column:initialCity;type:text"`
	InitialZip                string    `json:"initialZip" gorm:"column:initialZip;type:text"`
	InitialIp                 string    `json:"initialIp" gorm:"column:initialIp;type:text"`
	InitialUserAgent          string    `json:"initialUserAgent" gorm:"column:initialUserAgent;type:text"`
	InitialReferrer           string    `json:"initialReferrer" gorm:"column:initialReferrer;type:text"`
	InitialTimezone           string    `json:"initialTimezone" gorm:"column:initialTimezone;type:text"`
	IsIdentified              bool      `json:"isIdentified" gorm:"column:isIdentified"`
	InitialDeviceType         string    `json:"initialDeviceType" gorm:"column:initialDeviceType;type:text"`
	InitialPlatform           string    `json:"initialPlatform" gorm:"column:initialPlatform;type:text"`
	InitialBrowser            string    `json:"initialBrowser" gorm:"column:initialBrowser;type:text"`
	InitialLandingPage        string    `json:"initialLandingPage" gorm:"column:initialLandingPage;type:text"`
	InitialMarketingChannel   string    `json:"initialMarketingChannel" gorm:"column:initialMarketingChannel;type:text"`
	InitialProfession         string    `json:"initialProfession" gorm:"column:initialProfession;type:text"`
	InitialFunnel             string    `json:"initialFunnel" gorm:"column:initialFunnel;type:text"`
	InitialUtmSource          string    `json:"initialUtmSource" gorm:"column:initialUtmSource;type:text"`
	InitialUtmMedium          string    `json:"initialUtmMedium" gorm:"column:initialUtmMedium;type:text"`
	InitialUtmCampaign        string    `json:"initialUtmCampaign" gorm:"column:initialUtmCampaign;type:text"`
	InitialUtmContent         string    `json:"initialUtmContent" gorm:"column:initialUtmContent;type:text"`
	InitialUtmTerm            string    `json:"initialUtmTerm" gorm:"column:initialUtmTerm;type:text"`
	InitialLandingSpecialPath string    `json:"initialLandingSpecialPath" gorm:"column:initialLandingSpecialPath;type:text"`
	InitialReferrerDomain     string    `json:"initialReferrerDomain" gorm:"column:initialReferrerDomain;type:text"`
	InitialReferrerQuery      string    `json:"initialReferrerQuery" gorm:"column:initialReferrerQuery;type:text"`
	InitialReferrerHostname   string    `json:"initialReferrerHostname" gorm:"column:initialReferrerHostname;type:text"`
	InitialReferrerPath       string    `json:"initialReferrerPath" gorm:"column:initialReferrerPath;type:text"`
}
