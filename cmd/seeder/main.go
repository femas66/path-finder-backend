package main

import (
	"fmt"
	"haji-koding/path-finder/internal/config"
	"haji-koding/path-finder/internal/entity"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type SeedUser struct {
	Name           string
	Email          string
	Password       string
	Role           string
	NIM            string
	Location       string
	ProfilePicture string
	CV             string
	PortfolioURL   string
}

func main() {
	viperConfig := config.NewViper()
	log := config.NewLogger()
	db := config.NewDatabase(viperConfig, log)

	// Pastikan migrasi tabel siap
	if err := db.AutoMigrate(&entity.User{}, &entity.PasswordReset{}); err != nil {
		log.Fatalf("Gagal melakukan auto migrate: %v", err)
	}

	seedUsers := []SeedUser{
		{
			Name:           "Administrator PathFinder",
			Email:          "admin@pathfinder.com",
			Password:       "admin12345",
			Role:           entity.RoleAdmin,
			ProfilePicture: "https://ui-avatars.com/api/?name=Admin+PathFinder&background=0D8ABC&color=fff",
		},
		{
			Name:           "Mitra Mandiri Sejahtera (PT Solusi Teknologi)",
			Email:          "mitra@pathfinder.com",
			Password:       "mitra12345",
			Role:           entity.RoleMitra,
			Location:       "Jakarta Selatan, DKI Jakarta",
			ProfilePicture: "https://ui-avatars.com/api/?name=Mitra+Mandiri&background=059669&color=fff",
		},
		{
			Name:           "Budi Santoso",
			Email:          "mahasiswa@pathfinder.com",
			NIM:            "2201010001",
			Password:       "2201010001", // Default password mahasiswa adalah NIM
			Role:           entity.RoleMahasiswa,
			Location:       "Semarang, Jawa Tengah",
			ProfilePicture: "https://ui-avatars.com/api/?name=Budi+Santoso&background=D97706&color=fff",
			CV:             "https://example.com/cv/budi-santoso.pdf",
			PortfolioURL:   "https://budi-santoso.dev",
		},
	}

	fmt.Println("==================================================")
	fmt.Println("           SEEDING USER UNTUK TESTING             ")
	fmt.Println("==================================================")

	for _, u := range seedUsers {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("Gagal mengenkripsi password untuk %s: %v", u.Email, err)
		}

		var existing entity.User
		err = db.Where("email = ?", u.Email).First(&existing).Error
		if err != nil {
			// Buat user baru
			newUser := entity.User{
				ID:             uuid.New().String(),
				NIM:            u.NIM,
				Name:           u.Name,
				Email:          u.Email,
				Password:       string(hashedPassword),
				Role:           u.Role,
				Location:       u.Location,
				ProfilePicture: u.ProfilePicture,
				CV:             u.CV,
				PortfolioURL:   u.PortfolioURL,
			}
			if err := db.Create(&newUser).Error; err != nil {
				log.Fatalf("Gagal membuat user %s: %v", u.Email, err)
			}
			fmt.Printf("✅ [DIBUAT] Role: %-11s | Email: %-26s | Password: %s\n", u.Role, u.Email, u.Password)
		} else {
			// Update user yang sudah ada agar password & data sinkron
			existing.NIM = u.NIM
			existing.Name = u.Name
			existing.Password = string(hashedPassword)
			existing.Role = u.Role
			existing.Location = u.Location
			existing.ProfilePicture = u.ProfilePicture
			existing.CV = u.CV
			existing.PortfolioURL = u.PortfolioURL

			if err := db.Save(&existing).Error; err != nil {
				log.Fatalf("Gagal memperbarui user %s: %v", u.Email, err)
			}
			fmt.Printf("🔄 [DIPERBARUI] Role: %-11s | Email: %-26s | Password: %s\n", u.Role, u.Email, u.Password)
		}
	}

	fmt.Println("==================================================")
	fmt.Println("Seeding berhasil! Semua akun siap digunakan.")
}
