package usecase

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"
	"haji-koding/path-finder/internal/model/converter"
	"haji-koding/path-finder/internal/repository"
	"haji-koding/path-finder/internal/service"

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
	DB                      *gorm.DB
	Log                     *logrus.Logger
	Validate                *validator.Validate
	Viper                   *viper.Viper
	UserRepository          *repository.UserRepository
	PasswordResetRepository *repository.PasswordResetRepository
	MailService             service.MailService
}

func NewUserUseCase(
	db *gorm.DB,
	logger *logrus.Logger,
	validate *validator.Validate,
	viperConfig *viper.Viper,
	userRepo *repository.UserRepository,
	passwordResetRepo *repository.PasswordResetRepository,
	mailService service.MailService,
) *UserUseCase {
	return &UserUseCase{
		DB:                      db,
		Log:                     logger,
		Validate:                validate,
		Viper:                   viperConfig,
		UserRepository:          userRepo,
		PasswordResetRepository: passwordResetRepo,
		MailService:             mailService,
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

func (u *UserUseCase) ForgotPassword(ctx context.Context, request *model.ForgotPasswordRequest) (*model.ForgotPasswordResponse, error) {
	if err := u.Validate.Struct(request); err != nil {
		u.Log.Warnf("Invalid forgot password request: %v", err)
		return nil, fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	user := new(entity.User)
	err := u.UserRepository.FindByEmail(u.DB.WithContext(ctx), user, request.Email)
	if err != nil {
		u.Log.Warnf("User not found for email %s: %v", request.Email, err)
		return nil, fiber.NewError(fiber.StatusNotFound, "email tidak ditemukan")
	}

	// Hapus kode reset lama untuk email ini jika ada
	_ = u.PasswordResetRepository.DeleteByEmail(u.DB.WithContext(ctx), request.Email)

	// Buat 6-digit kode OTP
	code, err := generateRandomCode(6)
	if err != nil {
		u.Log.Errorf("Failed to generate OTP code: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "gagal membuat kode verifikasi")
	}

	passwordReset := &entity.PasswordReset{
		ID:        uuid.New().String(),
		Email:     request.Email,
		Code:      code,
		ExpiredAt: time.Now().Add(15 * time.Minute),
	}

	if err := u.PasswordResetRepository.Create(u.DB.WithContext(ctx), passwordReset); err != nil {
		u.Log.Errorf("Failed to save password reset code: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "gagal menyimpan kode verifikasi")
	}

	// Kirim email secara asynchronous agar waktu respons API sangat cepat (< 10ms)
	emailSubject := "Kode Verifikasi Lupa Password - Path Finder"
	emailBody := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; padding: 20px;">
			<h2>Kode Verifikasi Lupa Password</h2>
			<p>Halo <strong>%s</strong>,</p>
			<p>Kami menerima permintaan untuk mereset password akun Anda. Berikut adalah kode verifikasi Anda:</p>
			<div style="font-size: 24px; font-weight: bold; letter-spacing: 5px; color: #2563eb; padding: 10px 0;">
				%s
			</div>
			<p>Kode ini berlaku selama <strong>15 menit</strong>. Jangan bagikan kode ini kepada siapa pun.</p>
			<p>Jika Anda tidak merasa meminta reset password, abaikan email ini.</p>
		</div>
	`, user.Name, code)

	u.MailService.SendEmailAsync(request.Email, emailSubject, emailBody)

	return converter.UserToForgotPasswordResponse(request.Email), nil
}

func generateRandomCode(length int) (string, error) {
	const digits = "0123456789"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = digits[b[i]%byte(len(digits))]
	}
	return string(b), nil
}
