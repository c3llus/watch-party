package config

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// SecretManagerConfig represents the JSON structure stored in Secret Manager
type SecretManagerConfig struct {
	Application SecretApplicationConfig `json:"application"`
	Database    SecretDatabaseConfig    `json:"database"`
	Redis       SecretRedisConfig       `json:"redis"`
	Storage     SecretStorageConfig     `json:"storage"`
	Video       SecretVideoConfig       `json:"video"`
	Email       SecretEmailConfig       `json:"email"`
}

// SecretApplicationConfig holds application-specific settings from Secret Manager
type SecretApplicationConfig struct {
	Port               string `json:"port"`
	JWTSecret          string `json:"jwt_secret"`
	LogLevel           string `json:"log_level"`
	LogFormat          string `json:"log_format"`
	CORSAllowedOrigins string `json:"cors_allowed_origins"`
}

// SecretDatabaseConfig holds database connection settings from Secret Manager
type SecretDatabaseConfig struct {
	Name            string `json:"name"`
	Host            string `json:"host"`
	Port            string `json:"port"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	MaxOpenConns    string `json:"max_open_conns"`
	MaxIdleConns    string `json:"max_idle_conns"`
	ConnMaxLifetime string `json:"conn_max_lifetime"`
	SSLMode         string `json:"ssl_mode"`
}

// SecretRedisConfig holds Redis connection settings from Secret Manager
type SecretRedisConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Password string `json:"password"`
	DB       string `json:"db"`
}

// SecretStorageConfig holds storage provider settings from Secret Manager
type SecretStorageConfig struct {
	Provider           string `json:"provider"`
	GCSBucket          string `json:"gcs_bucket"`
	GCSCredentialsPath string `json:"gcs_credentials_path"`
}

// SecretVideoConfig holds video processing settings from Secret Manager
type SecretVideoConfig struct {
	ProcessingTempDir string `json:"processing_temp_dir"`
	HLSBaseURL        string `json:"hls_base_url"`
	FFmpegPath        string `json:"ffmpeg_path"`
	FFprobePath       string `json:"ffprobe_path"`
}

// SecretEmailConfig holds email service settings from Secret Manager
type SecretEmailConfig struct {
	Provider        string `json:"provider"`
	SMTPHost        string `json:"smtp_host"`
	SMTPPort        string `json:"smtp_port"`
	SMTPUsername    string `json:"smtp_username"`
	SMTPPassword    string `json:"smtp_password"`
	SMTPUseTLS      string `json:"smtp_use_tls"`
	TemplateBaseURL string `json:"template_base_url"`
	TemplateAppName string `json:"template_app_name"`
	FromEmail       string `json:"from_email"`
	FromName        string `json:"from_name"`
}

// LoadFromSecretManager loads configuration from Google Secret Manager
func LoadFromSecretManager(ctx context.Context, projectID, secretName string) (*Config, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create secretmanager client: %v", err)
	}
	defer client.Close()

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretName),
	}

	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to access secret version: %v", err)
	}

	var secretConfig SecretManagerConfig
	if err := json.Unmarshal(result.Payload.Data, &secretConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config JSON: %v", err)
	}

	// Convert to the existing Config structure
	return convertSecretToConfig(&secretConfig)
}

// convertSecretToConfig converts SecretManagerConfig to the existing Config structure
func convertSecretToConfig(secret *SecretManagerConfig) (*Config, error) {
	maxOpenConns, err := strconv.Atoi(secret.Database.MaxOpenConns)
	if err != nil {
		return nil, fmt.Errorf("invalid max_open_conns: %v", err)
	}

	maxIdleConns, err := strconv.Atoi(secret.Database.MaxIdleConns)
	if err != nil {
		return nil, fmt.Errorf("invalid max_idle_conns: %v", err)
	}

	connMaxLifetime, err := time.ParseDuration(secret.Database.ConnMaxLifetime)
	if err != nil {
		return nil, fmt.Errorf("invalid conn_max_lifetime: %v", err)
	}

	redisDB, err := strconv.Atoi(secret.Redis.DB)
	if err != nil {
		return nil, fmt.Errorf("invalid redis db: %v", err)
	}

	smtpPort, err := strconv.Atoi(secret.Email.SMTPPort)
	if err != nil {
		return nil, fmt.Errorf("invalid smtp_port: %v", err)
	}

	smtpUseTLS, err := strconv.ParseBool(secret.Email.SMTPUseTLS)
	if err != nil {
		return nil, fmt.Errorf("invalid smtp_use_tls: %v", err)
	}

	// Parse CORS allowed origins from comma-separated string
	corsOrigins := strings.Split(secret.Application.CORSAllowedOrigins, ",")
	for i, origin := range corsOrigins {
		corsOrigins[i] = strings.TrimSpace(origin)
	}

	return &Config{
		Port:      secret.Application.Port,
		JWTSecret: secret.Application.JWTSecret,
		Database: DatabaseConfig{
			Name:            secret.Database.Name,
			Host:            secret.Database.Host,
			Port:            secret.Database.Port,
			Username:        secret.Database.Username,
			Password:        secret.Database.Password,
			Database:        secret.Database.Name, // Use name as database
			MaxOpenConns:    maxOpenConns,
			MaxIdleConns:    maxIdleConns,
			ConnMaxLifetime: connMaxLifetime,
			SSLMode:         secret.Database.SSLMode,
		},
		Log: LogConfig{
			Level:  secret.Application.LogLevel,
			Format: secret.Application.LogFormat,
		},
		Storage: StorageConfig{
			Provider:           secret.Storage.Provider,
			GCSBucket:          secret.Storage.GCSBucket,
			GCSCredentialsPath: secret.Storage.GCSCredentialsPath,
			VideoProcessing: VideoConfig{
				TempDir:     secret.Video.ProcessingTempDir,
				HLSBaseURL:  secret.Video.HLSBaseURL,
				FFmpegPath:  secret.Video.FFmpegPath,
				FFprobePath: secret.Video.FFprobePath,
			},
		},
		Email: EmailConfig{
			Provider: secret.Email.Provider,
			SMTP: SMTPConfig{
				Host:     secret.Email.SMTPHost,
				Port:     smtpPort,
				Username: secret.Email.SMTPUsername,
				Password: secret.Email.SMTPPassword,
				UseTLS:   smtpUseTLS,
			},
			Templates: EmailTemplateConfig{
				BaseURL: secret.Email.TemplateBaseURL,
				AppName: secret.Email.TemplateAppName,
			},
		},
		Redis: RedisConfig{
			Host:     secret.Redis.Host,
			Port:     secret.Redis.Port,
			Password: secret.Redis.Password,
			DB:       redisDB,
		},
		CORS: CORSConfig{
			AllowedOrigins: corsOrigins,
		},
	}, nil
}
