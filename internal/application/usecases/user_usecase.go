package usecases

import (
	"context"
	"time"

	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/entities"
	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/repositories"
)

type IUserRepository interface {
	GetUsers(ctx context.Context, page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string) ([]entities.User, int64, error)
}

type UserUseCase struct {
	userRepo repositories.IUserRepository
}

func NewUserUseCase(userRepo repositories.IUserRepository) *UserUseCase {
	return &UserUseCase{
		userRepo: userRepo,
	}
}

func (uc *UserUseCase) GetUsers(ctx context.Context, page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string) ([]entities.User, int64, error) {
	return uc.userRepo.GetUsers(ctx, page, limit, orderBy, from, to, timeFrom, timeTo)
}
