package entity

type User struct {
	ID           uint `json:"id" gorm:"primaryKey"`
	IsIdentified bool `json:"isIdentified" gorm:"column:isIdentified"`
	IsClient     bool `json:"isClient" gorm:"column:isClient"`
	// ... outros campos existentes ...
}

// IsLead verifica se o usuário é um lead
func (u *User) IsLead() bool {
	return u.IsIdentified && !u.IsClient
}

// CheckIsClient verifica se o usuário é um cliente
func (u *User) CheckIsClient() bool {
	return u.IsClient
}

// IsAnonymous verifica se o usuário é anônimo
func (u *User) IsAnonymous() bool {
	return !u.IsIdentified && !u.IsClient
}
