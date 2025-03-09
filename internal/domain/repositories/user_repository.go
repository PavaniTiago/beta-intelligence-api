package repositories

import (
	"context"

	"github.com/PavaniTiago/beta-intelligence/internal/domain/entities"

	"gorm.io/gorm"
)

type IUserRepository interface {
	GetUsers(ctx context.Context, page, limit int, orderBy string) ([]entities.User, int64, error)
	FindLeads(page, limit int, orderBy string) ([]entities.User, int64, error)
	FindClients(page, limit int, orderBy string) ([]entities.User, int64, error)
	FindAnonymous(page, limit int, orderBy string) ([]entities.User, int64, error)
}

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetUsers(ctx context.Context, page, limit int, orderBy string) ([]entities.User, int64, error) {
	var users []entities.User
	var total int64

	offset := (page - 1) * limit

	query := r.db.Model(&entities.User{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if orderBy != "" {
		query = query.Order(orderBy)
	}

	if err := query.Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// FindLeads retorna todos os usuários que são leads com paginação e ordenação
func (r *UserRepository) FindLeads(page, limit int, orderBy string) ([]entities.User, int64, error) {
	var leads []entities.User
	var total int64
	offset := (page - 1) * limit

	query := r.db.Where(`"isIdentified" = ? AND "isClient" = ?`, true, false)

	if err := query.Model(&entities.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Atualizado para usar user_id como padrão
	if orderBy == "" {
		orderBy = "user_id asc"
	}
	query = query.Order(orderBy)

	err := query.Offset(offset).Limit(limit).Find(&leads).Error
	if err != nil {
		return nil, 0, err
	}

	return leads, total, nil
}

// FindClients retorna todos os usuários que são clientes com paginação e ordenação
func (r *UserRepository) FindClients(page, limit int, orderBy string) ([]entities.User, int64, error) {
	var clients []entities.User
	var total int64
	offset := (page - 1) * limit

	query := r.db.Where(`"isClient" = ?`, true)

	if err := query.Model(&entities.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if orderBy != "" {
		query = query.Order(orderBy)
	}

	err := query.Offset(offset).Limit(limit).Find(&clients).Error
	return clients, total, err
}

// FindAnonymous retorna todos os usuários anônimos com paginação e ordenação
func (r *UserRepository) FindAnonymous(page, limit int, orderBy string) ([]entities.User, int64, error) {
	var anonymous []entities.User
	var total int64
	offset := (page - 1) * limit

	query := r.db.Where(`"isIdentified" = ? AND "isClient" = ?`, false, false)

	if err := query.Model(&entities.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if orderBy != "" {
		query = query.Order(orderBy)
	}

	err := query.Offset(offset).Limit(limit).Find(&anonymous).Error
	return anonymous, total, err
}
