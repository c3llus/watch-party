package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// DeploymentMode represents the deployment configuration
type DeploymentMode string

const (
	// DeploymentSaaS - enterprise SaaS with CDN workers
	DeploymentSaaS DeploymentMode = "saas"
	// DeploymentSelfHosted - freemium self-hosted with built-in proxy
	DeploymentSelfHosted DeploymentMode = "self-hosted"
)

// DeploymentConfig holds deployment-specific configuration
type DeploymentConfig struct {
	Mode             DeploymentMode
	CDNConfig        *CDNConfig
	SelfHostedConfig *SelfHostedConfig
	MediaTokenConfig *MediaTokenConfig
}

// CDNConfig holds CDN-specific configuration for SaaS mode
type CDNConfig struct {
	Enabled        bool
	Provider       string // "cloudflare", "aws", "gcp"
	WorkerEndpoint string
	SigningKey     string
	CacheTTL       int // seconds
}

// SelfHostedConfig holds configuration for self-hosted mode
type SelfHostedConfig struct {
	ProxyEnabled   bool
	LocalCacheSize int64 // bytes
	LocalCacheTTL  int   // seconds
	TunnelEnabled  bool
	TunnelEndpoint string
	TunnelAPIKey   string
}

// MediaTokenConfig holds media token configuration
type MediaTokenConfig struct {
	SigningKey    string
	TokenTTL      int    // seconds (default: 60)
	Algorithm     string // "HS256", "RS256"
	PrivateKeyPEM string // for RS256
	PublicKeyPEM  string // for RS256
}

// LoadDeploymentConfig loads deployment configuration from environment
func LoadDeploymentConfig() (*DeploymentConfig, error) {
	mode := DeploymentMode(getEnvOrDefault("DEPLOYMENT_MODE", "self-hosted"))

	if mode != DeploymentSaaS && mode != DeploymentSelfHosted {
		return nil, fmt.Errorf("invalid deployment mode: %s", mode)
	}

	config := &DeploymentConfig{
		Mode: mode,
		MediaTokenConfig: &MediaTokenConfig{
			SigningKey: getEnvOrDefault("MEDIA_TOKEN_SIGNING_KEY", "default-dev-key-change-in-production"),
			TokenTTL:   getEnvIntOrDefault("MEDIA_TOKEN_TTL", 60),
			Algorithm:  getEnvOrDefault("MEDIA_TOKEN_ALGORITHM", "HS256"),
		},
	}

	if mode == DeploymentSaaS {
		config.CDNConfig = &CDNConfig{
			Enabled:        true,
			Provider:       getEnvOrDefault("CDN_PROVIDER", "cloudflare"),
			WorkerEndpoint: getEnvOrDefault("CDN_WORKER_ENDPOINT", ""),
			SigningKey:     getEnvOrDefault("CDN_SIGNING_KEY", ""),
			CacheTTL:       getEnvIntOrDefault("CDN_CACHE_TTL", 3600),
		}
	} else {
		config.SelfHostedConfig = &SelfHostedConfig{
			ProxyEnabled:   true,
			LocalCacheSize: getEnvInt64OrDefault("LOCAL_CACHE_SIZE", 1024*1024*1024), // 1GB default
			LocalCacheTTL:  getEnvIntOrDefault("LOCAL_CACHE_TTL", 3600),
			TunnelEnabled:  getEnvBoolOrDefault("TUNNEL_ENABLED", false),
			TunnelEndpoint: getEnvOrDefault("TUNNEL_ENDPOINT", ""),
			TunnelAPIKey:   getEnvOrDefault("TUNNEL_API_KEY", ""),
		}
	}

	return config, nil
}

// IsSaaS returns true if running in SaaS mode
func (dc *DeploymentConfig) IsSaaS() bool {
	return dc.Mode == DeploymentSaaS
}

// IsSelfHosted returns true if running in self-hosted mode
func (dc *DeploymentConfig) IsSelfHosted() bool {
	return dc.Mode == DeploymentSelfHosted
}

// GetMediaTokenEndpoint returns the appropriate endpoint for media token generation
func (dc *DeploymentConfig) GetMediaTokenEndpoint() string {
	if dc.IsSaaS() && dc.CDNConfig.WorkerEndpoint != "" {
		return dc.CDNConfig.WorkerEndpoint
	}
	return "/api/v1/auth/media_token"
}

// helper functions
func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

func getEnvInt64OrDefault(key string, defaultValue int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return defaultValue
	}
	return intValue
}

func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	value := strings.ToLower(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1" || value == "yes"
}
