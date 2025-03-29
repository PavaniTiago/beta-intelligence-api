package usecases

import (
	"context"
	"time"

	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/entities"
	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/repositories"
)

type EventUseCase interface {
	GetEvents(ctx context.Context, page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string, professionIDs, funnelIDs []int, advancedFilters []repositories.AdvancedFilter, filterCondition string) ([]entities.Event, int64, error)
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
