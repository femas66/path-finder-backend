package entity

import "time"

type PasswordReset struct {
	ID        string    `gorm:"column:id;primaryKey"`
	Email     string    `gorm:"column:email;not null;index"`
	Code      string    `gorm:"column:code;not null"`
	Attempts  int       `gorm:"column:attempts;default:0"`
	ExpiredAt time.Time `gorm:"column:expired_at;not null"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (p *PasswordReset) TableName() string {
	return "password_resets"
}
