package entity

import "time"

const (
	ApplicationStatusPending  = "pending"
	ApplicationStatusAccepted = "accepted"
	ApplicationStatusRejected = "rejected"
)

func IsValidApplicationStatus(status string) bool {
	return status == ApplicationStatusPending || status == ApplicationStatusAccepted || status == ApplicationStatusRejected
}

type Application struct {
	ID          string    `gorm:"column:id;primaryKey"`
	JobID       string    `gorm:"column:job_id;not null;index"`
	Job         *Job      `gorm:"foreignKey:JobID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	MahasiswaID string    `gorm:"column:mahasiswa_id;not null;index"`
	Mahasiswa   *User     `gorm:"foreignKey:MahasiswaID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Status      string    `gorm:"column:status;not null;default:'pending';index"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoCreateTime;autoUpdateTime"`
}

func (a *Application) TableName() string {
	return "applications"
}
