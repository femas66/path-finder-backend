package usecase

import (
	"context"
	"time"

	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"
	"haji-koding/path-finder/internal/model/converter"
	"haji-koding/path-finder/internal/repository"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserUseCase struct {
	DB             *gorm.DB
	Log            *logrus.Logger
	Validate       *validator.Validate
	Viper          *viper.Viper
	UserRepository *repository.UserRepository
}

func NewUserUseCase(
	db *gorm.DB,
	logger *logrus.Logger,
	validate *validator.Validate,
	viperConfig *viper.Viper,
	userRepo *repository.UserRepository,
) *UserUseCase {
	return &UserUseCase{
		DB:             db,
		Log:            logger,
		Validate:       validate,
		Viper:          viperConfig,
		UserRepository: userRepo,
	}
}

func (u *UserUseCase) Register(ctx context.Context, request *model.RegisterUserRequest) (*model.UserResponse, error) {
	tx := u.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := u.Validate.Struct(request); err != nil {
		u.Log.Warnf("Invalid register request: %v", err)
		return nil, fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if request.Password != request.ConfirmPassword {
		return nil, fiber.NewError(fiber.StatusBadRequest, "konfirmasi password tidak cocok")
	}

	total, err := u.UserRepository.CountByEmail(tx, request.Email)
	if err != nil {
		u.Log.Warnf("Failed to count user by email: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "terjadi kesalahan pada server")
	}

	if total > 0 {
		u.Log.Warnf("Email already registered: %s", request.Email)
		return nil, fiber.NewError(fiber.StatusConflict, "email sudah terdaftar")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		u.Log.Warnf("Failed to hash password: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "gagal mengenkripsi password")
	}

	user := &entity.User{
		ID:       uuid.New().String(),
		Name:     request.Name,
		Email:    request.Email,
		Password: string(hashedPassword),
		Role:     entity.RoleMahasiswa,
	}

	if err := u.UserRepository.Create(tx, user); err != nil {
		u.Log.Warnf("Failed to create user: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "gagal menyimpan data user")
	}

	if err := tx.Commit().Error; err != nil {
		u.Log.Warnf("Failed to commit transaction: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "gagal memproses transaksi")
	}

	return converter.UserToResponse(user), nil
}

func (u *UserUseCase) Login(ctx context.Context, request *model.LoginUserRequest) (*model.LoginResponse, error) {
	if err := u.Validate.Struct(request); err != nil {
		u.Log.Warnf("Invalid login request: %v", err)
		return nil, fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	user := new(entity.User)
	err := u.UserRepository.FindByEmail(u.DB.WithContext(ctx), user, request.Email)
	if err != nil {
		u.Log.Warnf("User not found for email %s: %v", request.Email, err)
		return nil, fiber.NewError(fiber.StatusUnauthorized, "email atau password salah")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password))
	if err != nil {
		u.Log.Warnf("Password mismatch for email %s", request.Email)
		return nil, fiber.NewError(fiber.StatusUnauthorized, "email atau password salah")
	}

	secret := u.Viper.GetString("jwt.secret")
	if secret == "" {
		secret = "super-secret-path-finder-key-2026"
	}

	expireSeconds := u.Viper.GetInt("jwt.expire")
	if expireSeconds <= 0 {
		expireSeconds = 86400
	}

	claims := jwt.MapClaims{
		"user_id": user.ID,
		"name":    user.Name,
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Duration(expireSeconds) * time.Second).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		u.Log.Warnf("Failed to generate JWT token: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "gagal membuat token autentikasi")
	}

	return converter.UserToLoginResponse(user, tokenString), nil
}
