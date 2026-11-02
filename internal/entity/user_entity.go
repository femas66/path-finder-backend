package entity

import "time"

const (
	RoleAdmin      = "admin"
	RoleMitra      = "mitra"
	RolePerusahaan = "perusahaan"
	RoleMahasiswa  = "mahasiswa"
)

type User struct {
	ID             string    `gorm:"column:id;primaryKey"`
	NIM            string    `gorm:"column:nim;index"`
	Name           string    `gorm:"column:name;not null"`
	Email          string    `gorm:"column:email;unique;not null"`
	Password       string    `gorm:"column:password;not null"`
	Role           string    `gorm:"column:role;not null;default:'mahasiswa'"`
	ProfilePicture string    `gorm:"column:profile_picture"`
	Location       string    `gorm:"column:location"`
	CV             string    `gorm:"column:cv"`
	PortfolioURL   string    `gorm:"column:portfolio_url"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoCreateTime;autoUpdateTime"`
}

func (u *User) TableName() string {
	return "users"
}
