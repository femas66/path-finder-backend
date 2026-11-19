package authconfig

import (
	"errors"
	"strings"

	"github.com/spf13/viper"
)

// JWTSecret is shared by token signing and verification so neither can fall back
// to a hardcoded secret when configuration is missing.
func JWTSecret(config *viper.Viper) (string, error) {
	if config == nil {
		return "", errors.New("jwt.secret wajib dikonfigurasi melalui JWT_SECRET atau config.json")
	}
	secret := config.GetString("jwt.secret")
	if strings.TrimSpace(secret) == "" {
		return "", errors.New("jwt.secret wajib dikonfigurasi melalui JWT_SECRET atau config.json")
	}
	return secret, nil
}
