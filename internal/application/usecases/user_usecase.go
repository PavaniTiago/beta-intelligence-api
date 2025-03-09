package usecases

import (
	"context"

	"github.com/PavaniTiago/beta-intelligence/internal/domain/entities"
)

type IUserRepository interface {
	GetUsers(ctx context.Context, page, limit int, orderBy string) ([]entities.User, int64, error)
}

type UserUseCase struct {
	userRepo IUserRepository
}

func NewUserUseCase(userRepo IUserRepository) *UserUseCase {
	return &UserUseCase{
		userRepo: userRepo,
	}
}

func (uc *UserUseCase) GetUsers(ctx context.Context, page, limit int, orderBy string) ([]entities.User, int64, error) {
	return uc.userRepo.GetUsers(ctx, page, limit, orderBy)
}
