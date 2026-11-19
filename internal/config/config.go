package config

import (
	"fmt"

	"haji-koding/path-finder/internal/authconfig"

	"github.com/spf13/viper"
)

func NewViper() *viper.Viper {
	config := viper.New()
	if err := config.BindEnv("jwt.secret", "JWT_SECRET"); err != nil {
		panic(fmt.Errorf("bind JWT_SECRET: %w", err))
	}

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
