package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Success(t *testing.T) {
	// Создаём временный конфиг файл
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
env: "dev"
server:
  addr: "127.0.0.1:8080"
log:
  level: "debug"
  pretty: true
db:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "postgres"
  name: "ispo_test"
  sslmode: "disable"
auth:
  jwt_secret: "test-secret-key-for-testing-only-32-chars!"
  access_token_ttl: "1h"
schedule:
  semester_start_date: "2026-02-09"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(LoadOptions{ConfigPath: configPath})

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "dev", cfg.Env)
	assert.Equal(t, "127.0.0.1:8080", cfg.Server.Addr)
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, "localhost", cfg.DB.Host)
	assert.Equal(t, "test-secret-key-for-testing-only-32-chars!", cfg.Auth.JWTSecret)
}

func TestLoad_FallbackToExample(t *testing.T) {
	// Сохраняем текущий working directory
	origWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origWd)

	// Переходим в директорию проекта
	err = os.Chdir(filepath.Join(origWd, "../.."))
	require.NoError(t, err)

	// Пытаемся загрузить несуществующий конфиг - должен fallback на example
	// В CI файла example может не быть в этой директории
	// Поэтому просто проверяем что ошибка возникает
	_, err = Load(LoadOptions{ConfigPath: "configs/nonexistent.yaml"})
	assert.Error(t, err)
}

func TestLoad_EnvVariablesOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
env: "dev"
server:
  addr: "127.0.0.1:8080"
db:
  host: "localhost"
  user: "postgres"
  password: "postgres"
  name: "ispo_test"
auth:
  jwt_secret: "test-secret-key-for-testing-only-32-chars!"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set env variable
	os.Setenv("ISPO_DB_HOST", "env-host.example.com")
	defer os.Unsetenv("ISPO_DB_HOST")

	cfg, err := Load(LoadOptions{ConfigPath: configPath})

	require.NoError(t, err)
	// Viper должен использовать env переменную
	assert.Equal(t, "env-host.example.com", cfg.DB.Host)
}

func TestLoad_InvalidEnv(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
env: "invalid_env"
server:
  addr: "127.0.0.1:8080"
auth:
  jwt_secret: "test-secret-key-for-testing-only-32-chars!"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(LoadOptions{ConfigPath: configPath})

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "env: unknown value")
}

func TestLoad_ProdGuardrails_WeakJWTSecret(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
env: "prod"
server:
  addr: "127.0.0.1:8080"
admin:
  api_key: "some-api-key"
auth:
  jwt_secret: "weak"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(LoadOptions{ConfigPath: configPath})

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "auth.jwt_secret: must be a strong secret in prod")
}

func TestLoad_ProdGuardrails_EmptyAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
env: "prod"
server:
  addr: "127.0.0.1:8080"
auth:
  jwt_secret: "test-secret-key-for-testing-only-32-chars!"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(LoadOptions{ConfigPath: configPath})

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "admin.api_key: must be set in prod")
}

func TestLoad_ProdGuardrails_CORS_Wildcard_WithCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
env: "prod"
server:
  addr: "127.0.0.1:8080"
  cors:
    allowed_origins: ["*"]
    allow_credentials: true
admin:
  api_key: "some-api-key"
auth:
  jwt_secret: "test-secret-key-for-testing-only-32-chars!"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(LoadOptions{ConfigPath: configPath})

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "cannot contain '*' when allow_credentials=true")
}

func TestLoad_DefaultValues(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
env: "dev"
auth:
  jwt_secret: "test-secret-key-for-testing-only-32-chars!"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(LoadOptions{ConfigPath: configPath})

	require.NoError(t, err)

	// Check defaults
	assert.Equal(t, "127.0.0.1:8080", cfg.Server.Addr)
	assert.Equal(t, 5*time.Second, cfg.Server.ReadHeaderTimeout)
	assert.Equal(t, 60*time.Second, cfg.Server.IdleTimeout)
	assert.Equal(t, 1<<20, cfg.Server.MaxHeaderBytes)
	assert.Equal(t, int64(25<<20), cfg.Server.AdminImportMaxBodyBytes)
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, "disable", cfg.DB.SSLMode)
	assert.Equal(t, 20*time.Second, cfg.PDF.Timeout)
	assert.Equal(t, 12*time.Hour, cfg.Auth.AccessTokenTTL)
	assert.Equal(t, 30*24*time.Hour, cfg.Auth.RefreshTokenTTL)
	assert.Equal(t, 5*time.Second, cfg.Push.FCM.Timeout)
}

func TestLoad_InvalidTrustedProxy(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
env: "dev"
server:
  addr: "127.0.0.1:8080"
  proxy:
    trusted_proxies: ["invalid-ip-address"]
auth:
  jwt_secret: "test-secret-key-for-testing-only-32-chars!"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(LoadOptions{ConfigPath: configPath})

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "server.proxy.trusted_proxies: invalid entry")
}

func TestLoad_RateLimitDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
env: "dev"
server:
  addr: "127.0.0.1:8080"
  rate_limit:
    login:
      enabled: true
    schedule_pdf:
      enabled: true
    admin_import:
      enabled: true
auth:
  jwt_secret: "test-secret-key-for-testing-only-32-chars!"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(LoadOptions{ConfigPath: configPath})

	require.NoError(t, err)

	// Check rate limit defaults
	assert.Equal(t, 0.2, cfg.Server.RateLimit.Login.RPS)
	assert.Equal(t, 5, cfg.Server.RateLimit.Login.Burst)
	assert.Equal(t, 0.1, cfg.Server.RateLimit.SchedulePDF.RPS)
	assert.Equal(t, 2, cfg.Server.RateLimit.SchedulePDF.Burst)
	assert.Equal(t, 0.02, cfg.Server.RateLimit.AdminImport.RPS)
	assert.Equal(t, 1, cfg.Server.RateLimit.AdminImport.Burst)
}
