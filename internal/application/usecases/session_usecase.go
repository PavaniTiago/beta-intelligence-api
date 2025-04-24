package usecases

import (
	"context"
	"time"

	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/entities"
	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/repositories"
)

// ISessionUseCase define a interface para operações de sessão
type ISessionUseCase interface {
	GetSessions(ctx context.Context, page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string, userID, professionID, productID, funnelID string, isActive *bool, landingPage string) ([]entities.Session, int64, error)
	FindSessionByID(ctx context.Context, id string) (*entities.Session, error)
	CountSessions(from, to time.Time, timeFrom, timeTo string, userID, professionID, productID, funnelID string, isActive *bool, landingPage string) (int64, error)
	CountSessionsByPeriods(periods []string, landingPage string, funnelID string, professionID string) (map[string]int64, error)
	GetSessionsDateRange() (time.Time, time.Time, error)
	FindActiveSessions(page, limit int, orderBy string, landingPage string, funnelID string, professionID string) ([]entities.Session, int64, error)
	CountActiveSessions(professionID string, funnelID string, landingPage string) (int64, error)
}

// SessionUseCase implementa a interface ISessionUseCase
type SessionUseCase struct {
	sessionRepo repositories.ISessionRepository
}

// NewSessionUseCase cria uma nova instância do SessionUseCase
func NewSessionUseCase(sessionRepo repositories.ISessionRepository) *SessionUseCase {
	return &SessionUseCase{
		sessionRepo: sessionRepo,
	}
}

// GetSessions obtém uma lista paginada de sessões com filtros
func (uc *SessionUseCase) GetSessions(ctx context.Context, page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string, userID, professionID, productID, funnelID string, isActive *bool, landingPage string) ([]entities.Session, int64, error) {
	return uc.sessionRepo.GetSessions(ctx, page, limit, orderBy, from, to, timeFrom, timeTo, userID, professionID, productID, funnelID, isActive, landingPage)
}

// FindSessionByID busca uma sessão pelo ID
func (uc *SessionUseCase) FindSessionByID(ctx context.Context, id string) (*entities.Session, error) {
	return uc.sessionRepo.FindSessionByID(ctx, id)
}

// CountSessions conta o número de sessões com filtros
func (uc *SessionUseCase) CountSessions(from, to time.Time, timeFrom, timeTo string, userID, professionID, productID, funnelID string, isActive *bool, landingPage string) (int64, error) {
	return uc.sessionRepo.CountSessions(from, to, timeFrom, timeTo, userID, professionID, productID, funnelID, isActive, landingPage)
}

// CountSessionsByPeriods conta sessões agrupadas por períodos
func (uc *SessionUseCase) CountSessionsByPeriods(periods []string, landingPage string, funnelID string, professionID string) (map[string]int64, error) {
	return uc.sessionRepo.CountSessionsByPeriods(periods, landingPage, funnelID, professionID)
}

// GetSessionsDateRange obtém o intervalo de datas das sessões
func (uc *SessionUseCase) GetSessionsDateRange() (time.Time, time.Time, error) {
	return uc.sessionRepo.GetSessionsDateRange()
}

// FindActiveSessions busca sessões ativas
func (uc *SessionUseCase) FindActiveSessions(page, limit int, orderBy string, landingPage string, funnelID string, professionID string) ([]entities.Session, int64, error) {
	return uc.sessionRepo.FindActiveSessions(page, limit, orderBy, landingPage, funnelID, professionID)
}

// CountActiveSessions conta o número de sessões ativas
func (uc *SessionUseCase) CountActiveSessions(professionID string, funnelID string, landingPage string) (int64, error) {
	return uc.sessionRepo.CountActiveSessions(professionID, funnelID, landingPage)
}
