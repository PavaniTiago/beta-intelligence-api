package entities

import (
	"time"
)

// Survey representa uma pesquisa no sistema
type Survey struct {
	SurveyID  int64     `json:"survey_id" gorm:"primaryKey;column:survey_id;type:int8"`
	Name      string    `json:"survey_name" gorm:"column:survey_name"`
	FunnelID  int       `json:"funnel_id" gorm:"column:funnel_id"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at"`

	// Relações
	Responses []SurveyResponse `json:"responses,omitempty" gorm:"foreignKey:SurveyID"`
}

// SurveyResponse representa uma resposta de pesquisa
type SurveyResponse struct {
	ID         string    `json:"id" gorm:"primaryKey;column:id;type:uuid"`
	SurveyID   int64     `json:"survey_id" gorm:"column:survey_id;type:int8"`
	EventID    string    `json:"event_id" gorm:"column:event_id;type:uuid"`
	TotalScore int       `json:"total_score" gorm:"column:total_score"`
	Completed  bool      `json:"completed" gorm:"column:completed"`
	CreatedAt  time.Time `json:"created_at" gorm:"column:created_at"`
	Faixa      string    `json:"faixa" gorm:"column:faixa"`

	// Relações
	Survey  Survey         `json:"survey,omitempty" gorm:"foreignKey:SurveyID"`
	Answers []SurveyAnswer `json:"answers,omitempty" gorm:"foreignKey:SurveyResponseID"`
}

// SurveyAnswer representa uma resposta individual a uma pergunta da pesquisa
type SurveyAnswer struct {
	ID               string    `json:"id" gorm:"primaryKey;column:id;type:uuid"`
	SurveyResponseID string    `json:"survey_response_id" gorm:"column:survey_response_id;type:uuid"`
	QuestionID       string    `json:"question_id" gorm:"column:question_id"`
	QuestionText     string    `json:"question_text" gorm:"column:question_text"`
	Value            string    `json:"value" gorm:"column:value"`
	Score            int       `json:"score" gorm:"column:score"`
	TimeToAnswer     float64   `json:"time_to_answer" gorm:"column:time_to_answer"`
	Changed          bool      `json:"changed" gorm:"column:changed"`
	Timestamp        time.Time `json:"timestamp" gorm:"column:timestamp"`
}
