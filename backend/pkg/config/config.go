package config

import (
	"log"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port      string         `json:"port"`
	JWTSecret string         `json:"jwt_secret"`
	Database  DatabaseConfig `json:"database"`
	Log       LogConfig      `json:"log"`
}

type DatabaseConfig struct {
	Name            string        `mapstructure:"db_name"`
	Host            string        `mapstructure:"db_host"`
	Port            string        `mapstructure:"db_port"`
	Username        string        `mapstructure:"db_username"`
	Password        string        `mapstructure:"db_password"`
	Database        string        `mapstructure:"db_database"`
	MaxOpenConns    int           `mapstructure:"db_max_open_conns"`
	MaxIdleConns    int           `mapstructure:"db_max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"db_conn_max_lifetime"`
	SSLMode         string        `mapstructure:"db_ssl_mode"` // e.g., "disable", "require", "verify-ca", "verify-full"
}

type LogConfig struct {
	Level string `mapstructure:"log_level"`
}

func init() {
	if !isGCP {
		err := godotenv.Load()
		if err != nil {
			log.Println("Warning: Could not find or load .env file.")
		}
	}
}

func NewConfig() *Config {
	return &Config{
		Port:      getOptionalSecret("PORT", "8080"),
		JWTSecret: getRequiredSecret("JWT_SECRET"),
		Database: DatabaseConfig{
			Name:            getRequiredSecret("DB_NAME"),
			Host:            getRequiredSecret("DB_HOST"),
			Port:            getRequiredSecret("DB_PORT"),
			Username:        getRequiredSecret("DB_USERNAME"),
			Password:        getRequiredSecret("DB_PASSWORD"),
			Database:        getRequiredSecret("DB_DATABASE"),
			MaxOpenConns:    parseInt("DB_MAX_OPEN_CONNS"),
			MaxIdleConns:    parseInt("DB_MAX_IDLE_CONNS"),
			ConnMaxLifetime: parseDuration("DB_CONN_MAX_LIFETIME"),
			SSLMode:         getOptionalSecret("DB_SSL_MODE", "disable"),
		},
		Log: LogConfig{
			Level: getOptionalSecret("LOG_LEVEL", "info"),
		},
	}
}
