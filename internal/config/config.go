package config

import (
	"fmt"

	"haji-koding/path-finder/internal/authconfig"

	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

func NewViper() *viper.Viper {
	_ = gotenv.Load(".env")
	_ = gotenv.Load("../.env")

	config := viper.New()
	_ = config.BindEnv("ml.service_url", "ML_SERVICE_URL")
	_ = config.BindEnv("ml.service_token", "ML_SERVICE_TOKEN")
	_ = config.BindEnv("ml.request_timeout_seconds", "ML_REQUEST_TIMEOUT_SECONDS")
	config.SetDefault("ml.request_timeout_seconds", 60)
	if err := config.BindEnv("jwt.secret", "JWT_SECRET"); err != nil {
		panic(fmt.Errorf("bind JWT_SECRET: %w", err))
	}

	_ = config.BindEnv("database.host", "DB_HOST")
	_ = config.BindEnv("database.port", "DB_PORT")
	_ = config.BindEnv("database.username", "DB_USER")
	_ = config.BindEnv("database.password", "DB_PASSWORD")
	_ = config.BindEnv("database.name", "DB_NAME")
	_ = config.BindEnv("database.sslmode", "DB_SSLMODE")

	_ = config.BindEnv("mail.host", "MAIL_HOST")
	_ = config.BindEnv("mail.port", "MAIL_PORT")
	_ = config.BindEnv("mail.username", "MAIL_USERNAME")
	_ = config.BindEnv("mail.password", "MAIL_PASSWORD")
	_ = config.BindEnv("mail.from", "MAIL_FROM")

	_ = config.BindEnv("firebase.credentials_file", "FIREBASE_CREDENTIALS_FILE")
	_ = config.BindEnv("firebase.credentials_json", "FIREBASE_CREDENTIALS_JSON")
	_ = config.BindEnv("firebase.project_id", "FIREBASE_PROJECT_ID")

	config.SetConfigName("config")
	config.SetConfigType("json")
	config.AddConfigPath("./../")
	config.AddConfigPath("./")
	err := config.ReadInConfig()

	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}

	if _, err := authconfig.JWTSecret(config); err != nil {
		panic(err)
	}

	return config
}
