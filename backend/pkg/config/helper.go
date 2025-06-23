package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

var isGCP = os.Getenv("GOOGLE_CLOUD_PROJECT") != ""

// getSecret retrieves the value of a secret from Google Cloud Secret Manager or environment variables.
func getSecret(key string) (string, error) {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID != "" {
		return accessSecretVersion(fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, key))
	}

	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("environment variable %q not set", key)
	}
	return value, nil
}

// getRequiredSecret is a helper func to get a required secret or fatal log on error.
func getRequiredSecret(key string) string {
	val, err := getSecret(key)
	if err != nil {
		log.Fatalf("FATAL: Cannot get required secret %q: %v", key, err)
	}
	if val == "" {
		log.Fatalf("FATAL: Required secret %q is empty", key)
	}
	return val
}

// getOptionalSecret is a helper func  to get an optional secret with a default value.
func getOptionalSecret(key, defaultValue string) string {
	val, err := getSecret(key)
	if err != nil || val == "" {
		return defaultValue
	}
	return val
}

// parseInt is a helper func  to parse an integer from a secret.
func parseInt(key string) int {
	valStr := getRequiredSecret(key)
	val, err := strconv.Atoi(valStr)
	if err != nil {
		log.Fatalf("FATAL: Invalid integer value for secret %q: %v", key, err)
	}
	return val
}

// parseDuration is a helper func  to parse a duration from a secret (e.g., "15m", "1h").
func parseDuration(key string) time.Duration {
	valStr := getRequiredSecret(key)
	val, err := time.ParseDuration(valStr)
	if err != nil {
		log.Fatalf("FATAL: Invalid duration value for secret %q (e.g. '15m'): %v", key, err)
	}
	return val
}
