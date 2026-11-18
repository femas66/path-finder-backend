package entity

import "time"

// Positions adalah delapan kategori posisi resmi yang dipakai kontrak ML.
// Nilai category pada tabel jobs wajib salah satu dari daftar ini.
var Positions = []string{
	"Software Engineer Intern",
	"Backend Developer Intern",
	"Mobile Developer Intern",
	"Frontend Developer Intern",
	"Data Analyst Intern",
	"UI/UX Designer Intern",
	"QA Engineer Intern",
	"Network & IT Support Intern",
}

// IsValidPosition memeriksa apakah sebuah kategori termasuk posisi resmi.
func IsValidPosition(category string) bool {
	for _, position := range Positions {
		if position == category {
			return true
		}
	}
	return false
}

// Job adalah katalog lowongan yang dipakai mesin rekomendasi.
//
// Daftar skill disimpan sebagai JSON agar tetap terbaca sebagai array
// tanpa memerlukan tabel relasi tambahan.
type Job struct {
	ID             string    `gorm:"column:id;primaryKey"`
	Title          string    `gorm:"column:title;not null"`
	Category       string    `gorm:"column:category;not null;index"`
	Company        string    `gorm:"column:company"`
	Description    string    `gorm:"column:description;type:text"`
	Location       string    `gorm:"column:location"`
	WorkType       string    `gorm:"column:work_type"`
	Source         string    `gorm:"column:source"`
	SourceURL      string    `gorm:"column:source_url"`
	RequiredSkills []string  `gorm:"column:required_skills;serializer:json;type:text"`
	OptionalSkills []string  `gorm:"column:optional_skills;serializer:json;type:text"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoCreateTime;autoUpdateTime"`
}

func (j *Job) TableName() string {
	return "jobs"
}
