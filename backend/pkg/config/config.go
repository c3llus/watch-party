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
	Storage   StorageConfig  `json:"storage"`
	Email     EmailConfig    `json:"email"`
	Redis     RedisConfig    `json:"redis"`
	CORS      CORSConfig     `json:"cors"`
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
	Level  string `mapstructure:"log_level"`
	Format string `mapstructure:"log_format"` // "console" or "json"
}

type StorageConfig struct {
	Provider           string      `mapstructure:"storage_provider"`
	GCSBucket          string      `mapstructure:"storage_gcs_bucket"`
	GCSCredentialsPath string      `mapstructure:"storage_gcs_credentials_path"`
	MinIO              MinIOConfig `mapstructure:"minio"`
	VideoProcessing    VideoConfig `mapstructure:"video_processing"`
}

type MinIOConfig struct {
	Endpoint       string `mapstructure:"endpoint"`
	AccessKey      string `mapstructure:"access_key"`
	SecretKey      string `mapstructure:"secret_key"`
	Bucket         string `mapstructure:"bucket"`
	UseSSL         bool   `mapstructure:"use_ssl"`
	PublicEndpoint string `mapstructure:"public_endpoint"` // For public URLs (if different from endpoint)
}

type VideoConfig struct {
	TempDir     string `mapstructure:"temp_dir"`
	HLSBaseURL  string `mapstructure:"hls_base_url"`
	FFmpegPath  string `mapstructure:"ffmpeg_path"`
	FFprobePath string `mapstructure:"ffprobe_path"`
}

type EmailConfig struct {
	Provider  string              `mapstructure:"email_provider"`
	SMTP      SMTPConfig          `mapstructure:"smtp"`
	SendGrid  SendGridConfig      `mapstructure:"sendgrid"`
	Templates EmailTemplateConfig `mapstructure:"templates"`
}

type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	UseTLS   bool   `mapstructure:"use_tls"`
}

type SendGridConfig struct {
	APIKey    string `mapstructure:"api_key"`
	FromEmail string `mapstructure:"from_email"`
	FromName  string `mapstructure:"from_name"`
}

type EmailTemplateConfig struct {
	BaseURL string `mapstructure:"base_url"`
	AppName string `mapstructure:"app_name"`
}

type RedisConfig struct {
	Host     string `mapstructure:"redis_host"`
	Port     string `mapstructure:"redis_port"`
	Password string `mapstructure:"redis_password"`
	DB       int    `mapstructure:"redis_db"`
}

type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"cors_allowed_origins"`
	AllowedMethods []string `mapstructure:"cors_allowed_methods"`
	AllowedHeaders []string `mapstructure:"cors_allowed_headers"`
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
			SSLMode:         getOptionalSecret("DB_SSL_MODE", "disable"), // Default to "disable" if not set
		},
		Log: LogConfig{
			Level: getOptionalSecret("LOG_LEVEL", "info"),
		},
		Storage: StorageConfig{
			Provider:           getOptionalSecret("STORAGE_PROVIDER", "minio"),
			GCSBucket:          getOptionalSecret("STORAGE_GCS_BUCKET", ""),
			GCSCredentialsPath: getOptionalSecret("STORAGE_GCS_CREDENTIALS_PATH", ""),
			MinIO: MinIOConfig{
				Endpoint:       getOptionalSecret("MINIO_ENDPOINT", "localhost:9000"),
				AccessKey:      getOptionalSecret("MINIO_ACCESS_KEY", "minioadmin"),
				SecretKey:      getOptionalSecret("MINIO_SECRET_KEY", "minioadmin"),
				Bucket:         getOptionalSecret("MINIO_BUCKET", "watch-party"),
				UseSSL:         parseBool("MINIO_USE_SSL"),
				PublicEndpoint: getOptionalSecret("MINIO_PUBLIC_ENDPOINT", ""),
			},
			VideoProcessing: VideoConfig{
				TempDir:     getOptionalSecret("VIDEO_PROCESSING_TEMP_DIR", "/tmp/watch-party-processing"),
				HLSBaseURL:  getOptionalSecret("VIDEO_HLS_BASE_URL", "http://localhost:8080/api/v1/files"),
				FFmpegPath:  getOptionalSecret("FFMPEG_PATH", "ffmpeg"),
				FFprobePath: getOptionalSecret("FFPROBE_PATH", "ffprobe"),
			},
		},
		Email: EmailConfig{
			Provider: getOptionalSecret("EMAIL_PROVIDER", "smtp"),
			SMTP: SMTPConfig{
				Host:     getOptionalSecret("EMAIL_SMTP_HOST", ""),
				Port:     parseOptionalInt("EMAIL_SMTP_PORT", 587),
				Username: getOptionalSecret("EMAIL_SMTP_USERNAME", ""),
				Password: getOptionalSecret("EMAIL_SMTP_PASSWORD", ""),
				UseTLS:   parseBool("EMAIL_SMTP_USE_TLS"),
			},
			SendGrid: SendGridConfig{
				APIKey:    getOptionalSecret("EMAIL_SENDGRID_API_KEY", ""),
				FromEmail: getOptionalSecret("EMAIL_SENDGRID_FROM_EMAIL", ""),
				FromName:  getOptionalSecret("EMAIL_SENDGRID_FROM_NAME", ""),
			},
			Templates: EmailTemplateConfig{
				BaseURL: getOptionalSecret("EMAIL_TEMPLATE_BASE_URL", "http://localhost:3000"),
				AppName: getOptionalSecret("EMAIL_TEMPLATE_APP_NAME", "WatchParty"),
			},
		},
		Redis: RedisConfig{
			Host:     getOptionalSecret("REDIS_HOST", "localhost"),
			Port:     getOptionalSecret("REDIS_PORT", "6379"),
			Password: getOptionalSecret("REDIS_PASSWORD", ""),
			DB:       parseOptionalInt("REDIS_DB", 0),
		},
		CORS: CORSConfig{
			AllowedOrigins: parseOptionalStringSlice("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173,http://localhost:5174"),
			AllowedMethods: parseOptionalStringSlice("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS"),
			AllowedHeaders: parseOptionalStringSlice("CORS_ALLOWED_HEADERS", "Content-Type,Authorization"),
		},
	}
}
