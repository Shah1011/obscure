package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type CloudProviderConfig struct {
	Provider string `json:"provider"` // "s3", "gcs", "b2", "idrive", or "s3-compatible"
	Enabled  bool   `json:"enabled"`
	// S3 specific fields
	Bucket          string `json:"bucket,omitempty"`
	Region          string `json:"region,omitempty"`
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
	// GCS specific fields
	ProjectID      string `json:"project_id,omitempty"`
	ServiceAccount string `json:"service_account,omitempty"` // Path to service account key file
	// Backblaze B2 specific fields
	ApplicationKeyID string `json:"application_key_id,omitempty"`
	ApplicationKey   string `json:"application_key,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"` // B2 endpoint URL
	// IDrive E2 specific fields (S3-compatible)
	IDriveEndpoint string `json:"idrive_endpoint,omitempty"` // IDrive E2 endpoint URL
	// S3-compatible generic fields
	S3CompatibleEndpoint string `json:"s3_compatible_endpoint,omitempty"` // Custom S3-compatible endpoint URL
	CustomName           string `json:"custom_name,omitempty"`            // Custom name for S3-compatible provider
}

type UserProviders struct {
	Providers map[string]*CloudProviderConfig `json:"providers"` // key is provider name (s3/gcs)
}

// Check if provider configuration is complete
func IsProviderConfigComplete(config *CloudProviderConfig) (bool, []string) {
	var missing []string

	switch config.Provider {
	case "s3":
		if strings.TrimSpace(config.Bucket) == "" {
			missing = append(missing, "bucket name")
		}
		if strings.TrimSpace(config.Region) == "" {
			missing = append(missing, "region")
		}
		if strings.TrimSpace(config.AccessKeyID) == "" {
			missing = append(missing, "access key ID")
		}
		if strings.TrimSpace(config.SecretAccessKey) == "" {
			missing = append(missing, "secret access key")
		}
	case "gcs":
		if strings.TrimSpace(config.ProjectID) == "" {
			missing = append(missing, "project ID")
		}
		if strings.TrimSpace(config.ServiceAccount) == "" {
			missing = append(missing, "service account path")
		}
		// Check if service account file exists
		if strings.TrimSpace(config.ServiceAccount) != "" {
			if _, err := os.Stat(config.ServiceAccount); os.IsNotExist(err) {
				missing = append(missing, "service account file not found")
			}
		}
	case "b2":
		if strings.TrimSpace(config.Bucket) == "" {
			missing = append(missing, "bucket name")
		}
		if strings.TrimSpace(config.Endpoint) == "" {
			missing = append(missing, "endpoint")
		}
		if strings.TrimSpace(config.ApplicationKeyID) == "" {
			missing = append(missing, "application key ID")
		}
		if strings.TrimSpace(config.ApplicationKey) == "" {
			missing = append(missing, "application key")
		}
	case "idrive":
		if strings.TrimSpace(config.Bucket) == "" {
			missing = append(missing, "bucket name")
		}
		if strings.TrimSpace(config.Region) == "" {
			missing = append(missing, "region")
		}
		if strings.TrimSpace(config.AccessKeyID) == "" {
			missing = append(missing, "access key ID")
		}
		if strings.TrimSpace(config.SecretAccessKey) == "" {
			missing = append(missing, "secret access key")
		}
		if strings.TrimSpace(config.IDriveEndpoint) == "" {
			missing = append(missing, "IDrive E2 endpoint")
		}
	case "s3-compatible":
		if strings.TrimSpace(config.Bucket) == "" {
			missing = append(missing, "bucket name")
		}
		if strings.TrimSpace(config.Region) == "" {
			missing = append(missing, "region")
		}
		if strings.TrimSpace(config.AccessKeyID) == "" {
			missing = append(missing, "access key ID")
		}
		if strings.TrimSpace(config.SecretAccessKey) == "" {
			missing = append(missing, "secret access key")
		}
		if strings.TrimSpace(config.S3CompatibleEndpoint) == "" {
			missing = append(missing, "S3-compatible endpoint")
		}
		if strings.TrimSpace(config.CustomName) == "" {
			missing = append(missing, "custom name")
		}
	}

	return len(missing) == 0, missing
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
	if !exists {
		return nil, fmt.Errorf("provider %s not configured", provider)
	}

	if !config.Enabled {
		return nil, fmt.Errorf("provider %s is disabled", provider)
	}

	// Check if configuration is complete
	isComplete, missing := IsProviderConfigComplete(config)
	if !isComplete {
		return nil, fmt.Errorf("provider %s configuration incomplete (missing: %s)",
			provider, strings.Join(missing, ", "))
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
			isComplete, _ := IsProviderConfigComplete(config)
			if isComplete {
				enabled = append(enabled, provider)
			}
		}
	}
	return enabled, nil
}
