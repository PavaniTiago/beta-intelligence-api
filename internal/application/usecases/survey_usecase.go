package usecases

import (
	"time"

	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/repositories"
)

// SurveyUseCase implementa os casos de uso relacionados a pesquisas
type SurveyUseCase struct {
	surveyRepo *repositories.SurveyRepository
}

// NewSurveyUseCase cria uma nova instância de SurveyUseCase
func NewSurveyUseCase(surveyRepo *repositories.SurveyRepository) *SurveyUseCase {
	return &SurveyUseCase{
		surveyRepo: surveyRepo,
	}
}

// GetSurveys retorna todas as pesquisas com opção de filtros
func (u *SurveyUseCase) GetSurveys(page, limit int, funnelID, surveyID int, includeFunnel bool) (interface{}, int64, error) {
	// Preparar parâmetros
	params := map[string]interface{}{
		"page":           page,
		"limit":          limit,
		"include_funnel": includeFunnel,
	}

	if funnelID > 0 {
		params["funnel_id"] = funnelID
	}

	if surveyID > 0 {
		params["survey_id"] = int64(surveyID)
	}

	// Buscar pesquisas
	return u.surveyRepo.GetSurveys(params)
}

// GetSurveyMetrics retorna métricas agregadas de pesquisas com base nos filtros fornecidos
func (u *SurveyUseCase) GetSurveyMetrics(params map[string]interface{}) (interface{}, error) {
	return u.surveyRepo.GetSurveyMetrics(params)
}

// GetSurveyDetails retorna detalhes de uma pesquisa específica, incluindo análise por pergunta e resposta
func (u *SurveyUseCase) GetSurveyDetails(surveyID int64, params map[string]interface{}) (interface{}, error) {
	return u.surveyRepo.GetSurveyDetails(surveyID, params)
}

// ParseDateParam converte uma string de data para time.Time
func (u *SurveyUseCase) ParseDateParam(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, nil
	}

	// Tentar formato ISO8601 com timezone
	t, err := time.Parse(time.RFC3339, dateStr)
	if err == nil {
		return t, nil
	}

	// Tentar formato de data simples
	t, err = time.Parse("2006-01-02", dateStr)
	if err == nil {
		// Definir hora para início do dia (00:00:00)
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()), nil
	}

	// Tentar formato de data e hora sem timezone
	t, err = time.Parse("2006-01-02T15:04:05", dateStr)
	if err == nil {
		return t, nil
	}

	return time.Time{}, err
}
