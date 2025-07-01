package config

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Duration is a custom type that wraps time.Duration to support JSON marshaling/unmarshaling
type Duration time.Duration

// MarshalJSON implements the json.Marshaler interface
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	duration, err := time.ParseDuration(s)
	if err != nil {
		return err
	}

	*d = Duration(duration)
	return nil
}

// ToDuration converts the custom Duration to time.Duration
func (d Duration) ToDuration() time.Duration {
	return time.Duration(d)
}

const (
	EnvProduction            = "production"
	EnvStaging               = "staging"
	EnvDevelopment           = "development"
	SecretNameConfig         = "watch-party-config"
	EnvVarEnvironment        = "ENVIRONMENT"
	EnvVarGCPProjectID       = "GCP_PROJECT_ID"
	EnvVarGoogleCloudProject = "GOOGLE_CLOUD_PROJECT"
	GCEMetadataEndpoint      = "http://metadata.google.internal/computeMetadata/v1/"
)

// isCloudEnvironment detects if we're running in a cloud environment that should use Secret Manager
func isCloudEnvironment() bool {
	return isGCP()
}

// getGCPProjectID gets the project ID from environment or GCE metadata
func getGCPProjectID() string {
	if projectID := os.Getenv(EnvVarGCPProjectID); projectID != "" {
		return projectID
	}
	if projectID := os.Getenv(EnvVarGoogleCloudProject); projectID != "" {
		return projectID
	}

	if projectID := getProjectIDFromMetadata(); projectID != "" {
		return projectID
	}

	return ""
}

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
	Name            string   `json:"name" mapstructure:"db_name"`
	Host            string   `json:"host" mapstructure:"db_host"`
	Port            string   `json:"port" mapstructure:"db_port"`
	Username        string   `json:"username" mapstructure:"db_username"`
	Password        string   `json:"password" mapstructure:"db_password"`
	Database        string   `json:"database" mapstructure:"db_database"`
	MaxOpenConns    int      `json:"max_open_conns" mapstructure:"db_max_open_conns"`
	MaxIdleConns    int      `json:"max_idle_conns" mapstructure:"db_max_idle_conns"`
	ConnMaxLifetime Duration `json:"conn_max_lifetime" mapstructure:"db_conn_max_lifetime"`
	SSLMode         string   `json:"ssl_mode" mapstructure:"db_ssl_mode"` // e.g., "disable", "require", "verify-ca", "verify-full"
}

type LogConfig struct {
	Level  string `json:"level" mapstructure:"log_level"`
	Format string `json:"format" mapstructure:"log_format"` // "console" or "json"
}

type StorageConfig struct {
	Provider            string      `json:"provider" mapstructure:"storage_provider"`
	GCSBucket           string      `json:"gcs_bucket" mapstructure:"storage_gcs_bucket"`
	GCSCredentialsPath  string      `json:"gcs_credentials_path" mapstructure:"storage_gcs_credentials_path"`
	GCSServiceAccountID string      `json:"gcs_service_account_id" mapstructure:"storage_gcs_service_account_id"`
	GCSPrivateKey       string      `json:"gcs_private_key" mapstructure:"storage_gcs_private_key"`
	MinIO               MinIOConfig `json:"minio" mapstructure:"minio"`
	VideoProcessing     VideoConfig `json:"video_processing" mapstructure:"video_processing"`
}

type MinIOConfig struct {
	Endpoint       string `json:"endpoint" mapstructure:"endpoint"`
	AccessKey      string `json:"access_key" mapstructure:"access_key"`
	SecretKey      string `json:"secret_key" mapstructure:"secret_key"`
	Bucket         string `json:"bucket" mapstructure:"bucket"`
	UseSSL         bool   `json:"use_ssl" mapstructure:"use_ssl"`
	PublicEndpoint string `json:"public_endpoint" mapstructure:"public_endpoint"` // For public URLs (if different from endpoint)
}

type VideoConfig struct {
	TempDir     string `json:"temp_dir" mapstructure:"temp_dir"`
	HLSBaseURL  string `json:"hls_base_url" mapstructure:"hls_base_url"`
	FFmpegPath  string `json:"ffmpeg_path" mapstructure:"ffmpeg_path"`
	FFprobePath string `json:"ffprobe_path" mapstructure:"ffprobe_path"`
}

type EmailConfig struct {
	Provider  string              `json:"provider" mapstructure:"email_provider"`
	SMTP      SMTPConfig          `json:"smtp" mapstructure:"smtp"`
	SendGrid  SendGridConfig      `json:"sendgrid" mapstructure:"sendgrid"`
	Templates EmailTemplateConfig `json:"templates" mapstructure:"templates"`
}

type SMTPConfig struct {
	Host     string `json:"host" mapstructure:"host"`
	Port     int    `json:"port" mapstructure:"port"`
	Username string `json:"username" mapstructure:"username"`
	Password string `json:"password" mapstructure:"password"`
	UseTLS   bool   `json:"use_tls" mapstructure:"use_tls"`
}

type SendGridConfig struct {
	APIKey    string `json:"api_key" mapstructure:"api_key"`
	FromEmail string `json:"from_email" mapstructure:"from_email"`
	FromName  string `json:"from_name" mapstructure:"from_name"`
}

type EmailTemplateConfig struct {
	BaseURL string `json:"base_url" mapstructure:"base_url"`
	AppName string `json:"app_name" mapstructure:"app_name"`
}

type RedisConfig struct {
	Host     string `json:"host" mapstructure:"redis_host"`
	Port     string `json:"port" mapstructure:"redis_port"`
	Password string `json:"password" mapstructure:"redis_password"`
	DB       int    `json:"db" mapstructure:"redis_db"`
}

type CORSConfig struct {
	AllowedOrigins []string `json:"allowed_origins" mapstructure:"cors_allowed_origins"`
	AllowedMethods []string `json:"allowed_methods" mapstructure:"cors_allowed_methods"`
	AllowedHeaders []string `json:"allowed_headers" mapstructure:"cors_allowed_headers"`
}

func init() {
	if !isCloudEnvironment() {
		err := godotenv.Load()
		if err != nil {
			log.Println("Warning: Could not find or load .env file.")
		}
	}
}

func NewConfig() *Config {
	if isCloudEnvironment() {
		ctx := context.Background()
		projectID := getGCPProjectID()
		environment := os.Getenv(EnvVarEnvironment)
		if environment == "" {
			environment = "cloud"
		}
		if projectID == "" {
			log.Fatalf("Failed to load from Secret Manager in %s environment", environment)
		}
		config, err := LoadFromSecretManager(ctx, projectID, SecretNameConfig)
		if err != nil {
			log.Fatalf("failed to load configuration from Google Secret Manager for %s environment: %v", environment, err)
		}

		log.Printf("Configuration loaded from Google Secret Manager for %s environment", environment)
		return config
	}

	environment := os.Getenv(EnvVarEnvironment)
	if environment == "" {
		environment = "development"
	}

	log.Printf("Loading configuration from environment variables for %s environment", environment)
	return loadFromEnvironment()
}

func loadFromEnvironment() *Config {
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
			ConnMaxLifetime: Duration(parseDuration("DB_CONN_MAX_LIFETIME")),
			SSLMode:         getOptionalSecret("DB_SSL_MODE", "disable"), // Default to "disable" if not set
		},
		Log: LogConfig{
			Level: getOptionalSecret("LOG_LEVEL", "info"),
		},
		Storage: StorageConfig{
			Provider:            getOptionalSecret("STORAGE_PROVIDER", "minio"),
			GCSBucket:           getOptionalSecret("STORAGE_GCS_BUCKET", ""),
			GCSCredentialsPath:  getOptionalSecret("STORAGE_GCS_CREDENTIALS_PATH", ""),
			GCSServiceAccountID: getOptionalSecret("STORAGE_GCS_SERVICE_ACCOUNT_ID", ""),
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
			AllowedHeaders: parseOptionalStringSlice("CORS_ALLOWED_HEADERS", "Content-Type,Authorization,x-guest-token,User-Agent,Sec-Ch-Ua,Sec-Ch-Ua-Mobile,Sec-Ch-Ua-Platform,Accept,Accept-Language,Accept-Encoding,Cache-Control,Connection,Host,Origin,Referer,Sec-Fetch-Dest,Sec-Fetch-Mode,Sec-Fetch-Site,X-Requested-With"),
		},
	}
}

// isRunningOnGCE checks if the application is running on Google Compute Engine (GCE).
func isGCP() bool {
	isCloudRun := isCloudRun()
	if isCloudRun {
		log.Println("Detected GCP environment")
		return true
	}

	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", GCEMetadataEndpoint+"instance/hostname", nil)
	if err != nil {
		return false
	}

	// required header for GCE metadata server
	req.Header.Add("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// GCE metadata server returns 200 OK for valid requests
	return resp.StatusCode == http.StatusOK
}

func isCloudRun() bool {
	log.Printf("service name: %s", os.Getenv("K_SERVICE"))
	return os.Getenv("K_SERVICE") != ""
}

// getProjectIDFromMetadata retrieves the Google Cloud project ID from the GCE metadata server.
func getProjectIDFromMetadata() string {
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", GCEMetadataEndpoint+"project/project-id", nil)
	if err != nil {
		return ""
	}

	// required header for GCE metadata server
	req.Header.Add("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	projectID, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(projectID))
}
