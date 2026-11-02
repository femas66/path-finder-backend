package config

import (
	"haji-koding/path-finder/internal/delivery/http"
	"haji-koding/path-finder/internal/delivery/http/middleware"
	"haji-koding/path-finder/internal/delivery/http/route"
	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/repository"
	"haji-koding/path-finder/internal/service"
	"haji-koding/path-finder/internal/usecase"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type BootstrapConfig struct {
	DB       *gorm.DB
	App      *fiber.App
	Log      *logrus.Logger
	Validate *validator.Validate
	Config   *viper.Viper
}

func Bootstrap(config *BootstrapConfig) {
	// Auto migrate entities
	if err := config.DB.AutoMigrate(&entity.User{}, &entity.PasswordReset{}); err != nil {
		config.Log.Fatalf("Failed to auto migrate database: %v", err)
	}

	// Setup repositories
	userRepository := repository.NewUserRepository(config.Log)
	passwordResetRepository := repository.NewPasswordResetRepository(config.Log)

	// Setup services
	mailService := service.NewMailService(config.Config, config.Log)

	// Setup use cases
	userUseCase := usecase.NewUserUseCase(
		config.DB,
		config.Log,
		config.Validate,
		config.Config,
		userRepository,
		passwordResetRepository,
		mailService,
	)
	mahasiswaUseCase := usecase.NewMahasiswaUseCase(config.DB, config.Log, config.Validate, userRepository)

	// Setup controllers
	userController := http.NewUserController(userUseCase, config.Log)
	adminMahasiswaController := http.NewAdminMahasiswaController(mahasiswaUseCase, config.Log)
	hajiCodingController := http.NewHajiCodingController(config.Log)

	// Setup middleware
	authMiddleware := middleware.NewAuth(config.Config)
	adminMiddleware := middleware.OnlyAdmin()
	mitraMiddleware := middleware.OnlyMitra()
	mahasiswaMiddleware := middleware.OnlyMahasiswa()

	// Setup routes
	routeConfig := route.RouteConfig{
		App:                      config.App,
		UserController:           userController,
		AdminMahasiswaController: adminMahasiswaController,
		HajiCodingController:     hajiCodingController,
		AuthMiddleware:           authMiddleware,
		AdminMiddleware:          adminMiddleware,
		MitraMiddleware:          mitraMiddleware,
		MahasiswaMiddleware:      mahasiswaMiddleware,
	}
	routeConfig.Setup()
}
