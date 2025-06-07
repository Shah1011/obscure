package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type CloudProviderConfig struct {
	Provider string `json:"provider"` // "s3" or "gcs"
	Enabled  bool   `json:"enabled"`
	// S3 specific fields
	Bucket          string `json:"bucket,omitempty"`
	Region          string `json:"region,omitempty"`
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
	// GCS specific fields
	ProjectID      string `json:"project_id,omitempty"`
	ServiceAccount string `json:"service_account,omitempty"` // Path to service account key file
}

type UserProviders struct {
	Providers map[string]*CloudProviderConfig `json:"providers"` // key is provider name (s3/gcs)
}

func getProvidersFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".obscure", "providers.json")
}

func LoadUserProviders() (*UserProviders, error) {
	filePath := getProvidersFilePath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &UserProviders{
				Providers: make(map[string]*CloudProviderConfig),
			}, nil
		}
		return nil, err
	}

	var providers UserProviders
	if err := json.Unmarshal(data, &providers); err != nil {
		return nil, err
	}

	if providers.Providers == nil {
		providers.Providers = make(map[string]*CloudProviderConfig)
	}

	return &providers, nil
}

func SaveUserProviders(providers *UserProviders) error {
	data, err := json.MarshalIndent(providers, "", "  ")
	if err != nil {
		return err
	}

	filePath := getProvidersFilePath()
	if err := os.MkdirAll(filepath.Dir(filePath), 0700); err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0600)
}

func AddProviderConfig(config *CloudProviderConfig) error {
	providers, err := LoadUserProviders()
	if err != nil {
		return err
	}

	providers.Providers[config.Provider] = config
	return SaveUserProviders(providers)
}

func GetProviderConfig(provider string) (*CloudProviderConfig, error) {
	providers, err := LoadUserProviders()
	if err != nil {
		return nil, err
	}

	config, exists := providers.Providers[provider]
	if !exists || !config.Enabled {
		return nil, fmt.Errorf("provider %s not configured or not enabled", provider)
	}

	return config, nil
}

func RemoveProviderConfig(provider string) error {
	providers, err := LoadUserProviders()
	if err != nil {
		return err
	}

	delete(providers.Providers, provider)
	return SaveUserProviders(providers)
}

func ListConfiguredProviders() ([]string, error) {
	providers, err := LoadUserProviders()
	if err != nil {
		return nil, err
	}

	var enabled []string
	for provider, config := range providers.Providers {
		if config.Enabled {
			enabled = append(enabled, provider)
		}
	}
	return enabled, nil
}
