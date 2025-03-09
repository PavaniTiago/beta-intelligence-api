package repository

import (
	"github.com/PavaniTiago/beta-intelligence/internal/domain/entity"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// FindLeads retorna todos os usuários que são leads
func (r *UserRepository) FindLeads() ([]entity.User, error) {
	var leads []entity.User
	err := r.db.Where("is_identified = ? AND is_client = ?", true, false).Find(&leads).Error
	return leads, err
}

// FindClients retorna todos os usuários que são clientes
func (r *UserRepository) FindClients() ([]entity.User, error) {
	var clients []entity.User
	err := r.db.Where("is_client = ?", true).Find(&clients).Error
	return clients, err
}

// FindAnonymous retorna todos os usuários anônimos
func (r *UserRepository) FindAnonymous() ([]entity.User, error) {
	var anonymous []entity.User
	err := r.db.Where("is_identified = ? AND is_client = ?", false, false).Find(&anonymous).Error
	return anonymous, err
}

// FindByType retorna usuários baseado no tipo (lead, client, anonymous)
func (r *UserRepository) FindByType(userType string) ([]entity.User, error) {
	var users []entity.User
	query := r.db

	switch userType {
	case "lead":
		query = query.Where("is_identified = ? AND is_client = ?", true, false)
	case "client":
		query = query.Where("is_client = ?", true)
	case "anonymous":
		query = query.Where("is_identified = ? AND is_client = ?", false, false)
	}

	err := query.Find(&users).Error
	return users, err
}
