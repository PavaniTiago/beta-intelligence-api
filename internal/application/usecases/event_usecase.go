package usecases

import (
	"context"
	"time"

	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/entities"
	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/repositories"
)

type EventUseCase interface {
	GetEvents(ctx context.Context, page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string, professionIDs, funnelIDs []int, advancedFilters []repositories.AdvancedFilter, filterCondition string) ([]entities.Event, int64, error)
	CountEvents(from, to time.Time, timeFrom, timeTo string, eventType string, professionIDs, funnelIDs []int, advancedFilters []repositories.AdvancedFilter, filterCondition string) (int64, error)
	CountEventsByPeriods(periods []string, eventType string, advancedFilters []repositories.AdvancedFilter, funnelID int, professionID int) (map[string]int64, error)
	GetEventsDateRange(eventType string) (time.Time, time.Time, error)
}

type eventUseCase struct {
	eventRepo repositories.EventRepository
}

func NewEventUseCase(eventRepo repositories.EventRepository) EventUseCase {
	return &eventUseCase{eventRepo}
}

func (uc *eventUseCase) GetEvents(ctx context.Context, page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string, professionIDs, funnelIDs []int, advancedFilters []repositories.AdvancedFilter, filterCondition string) ([]entities.Event, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if orderBy == "" {
		orderBy = "event_time desc"
	}

	return uc.eventRepo.GetEvents(ctx, page, limit, orderBy, from, to, timeFrom, timeTo, professionIDs, funnelIDs, advancedFilters, filterCondition)
}

func (uc *eventUseCase) CountEvents(from, to time.Time, timeFrom, timeTo string, eventType string, professionIDs, funnelIDs []int, advancedFilters []repositories.AdvancedFilter, filterCondition string) (int64, error) {
	return uc.eventRepo.CountEvents(from, to, timeFrom, timeTo, eventType, professionIDs, funnelIDs, advancedFilters, filterCondition)
}

func (uc *eventUseCase) CountEventsByPeriods(periods []string, eventType string, advancedFilters []repositories.AdvancedFilter, funnelID int, professionID int) (map[string]int64, error) {
	return uc.eventRepo.CountEventsByPeriods(periods, eventType, advancedFilters, funnelID, professionID)
}

func (uc *eventUseCase) GetEventsDateRange(eventType string) (time.Time, time.Time, error) {
	return uc.eventRepo.GetEventsDateRange(eventType)
}
