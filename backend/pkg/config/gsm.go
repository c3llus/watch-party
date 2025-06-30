package config

import (
	"context"
	"encoding/json"
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// LoadFromSecretManager loads configuration from Google Secret Manager
func LoadFromSecretManager(ctx context.Context, projectID, secretName string) (*Config, error) {
	secretData, err := accessSecretVersion(fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretName))
	if err != nil {
		return nil, fmt.Errorf("failed to access secret: %w", err)
	}

	var config Config
	if err := json.Unmarshal([]byte(secretData), &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret data: %w", err)
	}

	return &config, nil
}

// accessSecretVersion accesses the payload for the given secret version if it exists.
func accessSecretVersion(name string) (string, error) {
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create secretmanager client: %w", err)
	}
	defer client.Close()

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	}

	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to access secret version: %w", err)
	}

	return string(result.Payload.Data), nil
}
