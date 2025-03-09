package repositories

import "context"

// Repository interface base para todos os repositórios
type Repository interface {
	Create(ctx context.Context, entity interface{}) error
	Update(ctx context.Context, entity interface{}) error
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (interface{}, error)
}
