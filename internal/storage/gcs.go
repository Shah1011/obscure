package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
	cfg "github.com/shah1011/obscure/internal/config"
	"google.golang.org/api/option"
)

// findGCSServiceAccount tries multiple locations to find the GCS service account file
func findGCSServiceAccount(configuredPath string) string {
	// List of paths to try in order
	var pathsToTry []string
	
	// 1. User-configured path (if provided)
	if configuredPath != "" {
		pathsToTry = append(pathsToTry, configuredPath)
	}
	
	// 2. Standard location in user's home directory
	if homeDir, err := os.UserHomeDir(); err == nil {
		pathsToTry = append(pathsToTry, filepath.Join(homeDir, ".obscure", "gcs-service-account.json"))
	}
	
	// 3. Current working directory
	pathsToTry = append(pathsToTry, "./gcs-service-account.json")
	
	// 4. Environment variable GOOGLE_APPLICATION_CREDENTIALS
	if envPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); envPath != "" {
		pathsToTry = append(pathsToTry, envPath)
	}
	
	// Try each path and return the first one that exists
	for _, path := range pathsToTry {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	
	return "" // No valid path found
}

func NewGCSClient(ctx context.Context, provider string) (*storage.Client, error) {
	// Get provider configuration
	providerConfig, err := cfg.GetProviderConfig(provider)
	if err != nil {
		return nil, err
	}

	// Try multiple locations for service account file
	serviceAccountPath := findGCSServiceAccount(providerConfig.ServiceAccount)
	if serviceAccountPath == "" {
		return nil, fmt.Errorf("GCS service account file not found. Please place your service account JSON file in one of these locations:\n" +
			"1. %s (as configured)\n" +
			"2. ~/.obscure/gcs-service-account.json\n" +
			"3. ./gcs-service-account.json\n" +
			"4. Set GOOGLE_APPLICATION_CREDENTIALS environment variable", 
			providerConfig.ServiceAccount)
	}

	// Create client with service account credentials
	client, err := storage.NewClient(ctx,
		option.WithCredentialsFile(serviceAccountPath),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client with service account %s: %w", serviceAccountPath, err)
	}

	return client, nil
}
