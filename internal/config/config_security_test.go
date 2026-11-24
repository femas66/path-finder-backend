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
	for _, k := range []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "MAIL_HOST", "MAIL_PORT", "MAIL_USERNAME", "MAIL_PASSWORD", "MAIL_FROM", "FIREBASE_CREDENTIALS_FILE", "FIREBASE_CREDENTIALS_JSON", "FIREBASE_PROJECT_ID"} {
		t.Setenv(k, "")
	}
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

func TestValidatorURLSecurity(t *testing.T) {
	validate := NewValidator()

	type URLTest struct {
		URL string `validate:"omitempty,http_url"`
	}

	testCases := []struct {
		url   string
		valid bool
	}{
		{"https://example.com/photo.png", true},
		{"http://example.com/cv.pdf", true},
		{"https://subdomain.domain.org/portfolio", true},
		{"http://localhost/admin", false},
		{"http://127.0.0.1:8080/secret", false},
		{"http://0.0.0.0/test", false},
		{"http://169.254.169.254/latest/meta-data/", false},
		{"javascript:alert(1)", false},
		{"data:text/html,test", false},
		{"ftp://ftp.example.com", false},
		{"", true},
	}

	for _, tc := range testCases {
		t.Run(tc.url, func(t *testing.T) {
			err := validate.Struct(&URLTest{URL: tc.url})
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestDatabaseSSLModeConfig(t *testing.T) {
	t.Setenv("DB_SSLMODE", "require")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{
		"jwt": {"secret": "mysecret"},
		"database": {"host": "localhost", "port": 5432, "username": "localuser", "password": "localpassword", "name": "localdb", "sslmode": "disable"}
	}`), 0600))
	t.Chdir(dir)

	cfg := NewViper()
	require.Equal(t, "require", cfg.GetString("database.sslmode"))
}


