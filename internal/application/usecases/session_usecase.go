package usecases

import (
	"context"
	"time"

	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/entities"
	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/repositories"
)

type SessionUseCase struct {
	sessionRepo repositories.ISessionRepository
}

func NewSessionUseCase(sessionRepo repositories.ISessionRepository) *SessionUseCase {
	return &SessionUseCase{
		sessionRepo: sessionRepo,
	}
}

func (uc *SessionUseCase) GetSessions(ctx context.Context, page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string, userID, professionID, productID, funnelID string, isActive *bool, landingPage string) ([]entities.Session, int64, error) {
	return uc.sessionRepo.GetSessions(ctx, page, limit, orderBy, from, to, timeFrom, timeTo, userID, professionID, productID, funnelID, isActive, landingPage)
}

func (uc *SessionUseCase) FindSessionByID(ctx context.Context, id string) (*entities.Session, error) {
	return uc.sessionRepo.FindSessionByID(ctx, id)
}

func (uc *SessionUseCase) CountSessions(from, to time.Time, timeFrom, timeTo string, userID, professionID, productID, funnelID string, isActive *bool, landingPage string) (int64, error) {
	return uc.sessionRepo.CountSessions(from, to, timeFrom, timeTo, userID, professionID, productID, funnelID, isActive, landingPage)
}

func (uc *SessionUseCase) CountSessionsByPeriods(periods []string, landingPage string, funnelID string, professionID string) (map[string]int64, error) {
	return uc.sessionRepo.CountSessionsByPeriods(periods, landingPage, funnelID, professionID)
}

func (uc *SessionUseCase) GetSessionsDateRange() (time.Time, time.Time, error) {
	return uc.sessionRepo.GetSessionsDateRange()
}

func (uc *SessionUseCase) FindActiveSessions(page, limit int, orderBy string, landingPage string, funnelID string, professionID string) ([]entities.Session, int64, error) {
	return uc.sessionRepo.FindActiveSessions(page, limit, orderBy, landingPage, funnelID, professionID)
}

func (uc *SessionUseCase) CountActiveSessions(professionID string, funnelID string, landingPage string) (int64, error) {
	return uc.sessionRepo.CountActiveSessions(professionID, funnelID, landingPage)
}
