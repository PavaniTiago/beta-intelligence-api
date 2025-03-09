package usecases

import "context"

// UseCase interface base para todos os casos de uso
type UseCase interface {
	Execute(ctx context.Context, input interface{}) (interface{}, error)
}
