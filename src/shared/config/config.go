package config

import (
	"os"
	"strconv"
	"time"
)

// Config estrutura de configuração da aplicação
type Config struct {
	// Configurações da aplicação
	AppPort string
	AppEnv  string
	GinMode string

	// Configurações do banco de dados
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Configurações do Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	// Configurações de JWT
	JWTSecret          string
	JWTExpirationHours int

	// Configurações de Log
	LogLevel  string
	LogFormat string

	// Configurações de Multi-tenant
	DefaultTenantID string

	// Configurações de CORS
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string

	// Configurações B3 (OAuth2 + mTLS)
	B3URLOptIn      string
	B3URLData        string
	B3OAuthTokenURL  string
	B3ClientID       string
	B3ClientSecret   string
	B3Scope          string
	B3CertP12Path    string
	B3CertPassphrase string
	B3LegacyCertPath string
	B3TimeoutSeconds int
	B3MaxRetries     int
	B3InitialBackoff int // ms
	B3MaxBackoff     int // ms
}

// Load carrega as configurações do ambiente
func Load() *Config {
	return &Config{
		AppPort: getEnv("APP_PORT", "8080"),
		AppEnv:  getEnv("APP_ENV", "development"),
		GinMode: getEnv("GIN_MODE", "debug"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", ""),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "suno_wallets"),
		DBSSLMode:  getEnv("DB_SSL_MODE", "disable"),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		JWTSecret:          getEnv("JWT_SECRET", "your-super-secret-jwt-key"),
		JWTExpirationHours: getEnvInt("JWT_EXPIRATION_HOURS", 24),

		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "json"),

		DefaultTenantID: getEnv("DEFAULT_TENANT_ID", "00000000-0000-0000-0000-000000000001"),

		AllowedOrigins: []string{"http://localhost:3000", "http://localhost:8080"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Tenant-ID"},

		// B3
		B3URLOptIn:      getEnv("B3_URL_OPTIN", ""),
		B3URLData:        getEnv("B3_URL_DATA", ""),
		B3OAuthTokenURL:  getEnv("B3_OAUTH_TOKEN_URL", ""),
		B3ClientID:       getEnv("B3_CLIENT_ID", ""),
		B3ClientSecret:   getEnv("B3_CLIENT_SECRET", ""),
		B3Scope:          getEnv("B3_SCOPE", ""),
		B3CertP12Path:    getEnv("B3_CERT_P12_PATH", "certs/b3_certificate12filepath.p12"),
		B3CertPassphrase: getEnv("B3_CERT_PASSPHRASE", ""),
		B3LegacyCertPath: getEnv("B3_LEGACY_CERT_PATH", "certs/b3_certificatefilepath.cer"),
		B3TimeoutSeconds: getEnvInt("B3_TIMEOUT_SECONDS", 10),
		B3MaxRetries:     getEnvInt("B3_MAX_RETRIES", 3),
		B3InitialBackoff: getEnvInt("B3_INITIAL_BACKOFF_MS", 200),
		B3MaxBackoff:     getEnvInt("B3_MAX_BACKOFF_MS", 2000),
	}
}

// getEnv busca variável de ambiente ou retorna valor padrão
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt busca variável de ambiente como inteiro ou retorna valor padrão
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetJWTDuration retorna a duração do token JWT
func (c *Config) GetJWTDuration() time.Duration {
	return time.Duration(c.JWTExpirationHours) * time.Hour
}
