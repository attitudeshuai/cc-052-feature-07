package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string

	RedisHost     string
	RedisPort     int
	RedisPassword string

	MinIOEndpoint       string
	MinIOPublicEndpoint string
	MinIOAccessKey      string
	MinIOSecretKey      string
	MinIOBucket         string
	MinIOUseSSL         bool

	ServerPort string
}

func Load() *Config {
	godotenv.Load()

	cfg := &Config{
		DBHost:              getEnv("DB_HOST", "localhost"),
		DBPort:              getEnvInt("DB_PORT", 5432),
		DBUser:              getEnv("DB_USER", "farm"),
		DBPassword:          getEnv("DB_PASSWORD", "farm_secret"),
		DBName:              getEnv("DB_NAME", "farm_trace"),
		RedisHost:           getEnv("REDIS_HOST", "localhost"),
		RedisPort:           getEnvInt("REDIS_PORT", 6379),
		RedisPassword:       getEnv("REDIS_PASSWORD", ""),
		MinIOEndpoint:       getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOPublicEndpoint: getEnv("MINIO_PUBLIC_ENDPOINT", "localhost:9000"),
		MinIOAccessKey:      getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey:      getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOBucket:         getEnv("MINIO_BUCKET", "farm-trace"),
		MinIOUseSSL:         getEnvBool("MINIO_USE_SSL", false),
		ServerPort:          getEnv("SERVER_PORT", "8080"),
	}
	return cfg
}

func (c *Config) DSN() string {
	return "host=" + c.DBHost + " port=" + strconv.Itoa(c.DBPort) +
		" user=" + c.DBUser + " password=" + c.DBPassword +
		" dbname=" + c.DBName + " sslmode=disable"
}

func (c *Config) RedisAddr() string {
	return c.RedisHost + ":" + strconv.Itoa(c.RedisPort)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}