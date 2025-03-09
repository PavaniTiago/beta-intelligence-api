package usecases

import (
	"time"

	"github.com/PavaniTiago/beta-intelligence/internal/domain/entities"
	"github.com/PavaniTiago/beta-intelligence/internal/domain/repositories"
)

type EventUseCase interface {
	GetEvents(page, limit int, orderBy string, from, to time.Time, professionIDs, funnelIDs []int) ([]entities.Event, int64, error)
}

type eventUseCase struct {
	eventRepo repositories.EventRepository
}

func NewEventUseCase(eventRepo repositories.EventRepository) EventUseCase {
	return &eventUseCase{eventRepo}
}

func (uc *eventUseCase) GetEvents(page, limit int, orderBy string, from, to time.Time, professionIDs, funnelIDs []int) ([]entities.Event, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if orderBy == "" {
		orderBy = "event_time desc"
	}

	return uc.eventRepo.GetEvents(page, limit, orderBy, from, to, professionIDs, funnelIDs)
}
