package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJWTSecretConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name       string
		fileSecret string
		envSecret  string
		want       string
	}{
		{"environment overrides file", "file-test-secret", "env-test-secret", "env-test-secret"},
		{"explicit file secret", "file-test-secret", "", "file-test-secret"},
		{"missing", "", "", ""},
		{"whitespace", "   ", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"jwt":{"secret":"`+tc.fileSecret+`"}}`), 0600))
			t.Chdir(dir)
			t.Setenv("JWT_SECRET", tc.envSecret)
			if tc.want == "" {
				require.Panics(t, func() { NewViper() })
				return
			}
			require.Equal(t, tc.want, NewViper().GetString("jwt.secret"))
		})
	}
}

func TestDatabaseAndMailEnvOverride(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{
		"jwt": {"secret": "mysecret"},
		"database": {"host": "localhost", "port": 5432, "username": "localuser", "password": "localpassword", "name": "localdb"},
		"mail": {"host": "localhost", "username": "mailuser", "password": "mailpass"}
	}`), 0600))
	t.Chdir(dir)

	cfg := NewViper()
	require.Equal(t, "localhost", cfg.GetString("database.host"))
	require.Equal(t, "localuser", cfg.GetString("database.username"))
	require.Equal(t, "localpassword", cfg.GetString("database.password"))

	t.Setenv("DB_HOST", "postgres")
	t.Setenv("DB_USER", "customuser")
	t.Setenv("DB_PASSWORD", "customsecret")
	t.Setenv("MAIL_HOST", "smtp.custom.com")
	t.Setenv("MAIL_PASSWORD", "smtpsecret")

	cfgEnv := NewViper()
	require.Equal(t, "postgres", cfgEnv.GetString("database.host"))
	require.Equal(t, "customuser", cfgEnv.GetString("database.username"))
	require.Equal(t, "customsecret", cfgEnv.GetString("database.password"))
	require.Equal(t, "smtp.custom.com", cfgEnv.GetString("mail.host"))
	require.Equal(t, "smtpsecret", cfgEnv.GetString("mail.password"))
}

