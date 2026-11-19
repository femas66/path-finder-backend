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
