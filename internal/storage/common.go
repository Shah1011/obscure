package storage

import (
	cfg "github.com/shah1011/obscure/internal/config"
)

// GetBucketName returns the bucket name for the specified provider
func GetBucketName(provider string) (string, error) {
	providerConfig, err := cfg.GetProviderConfig(provider)
	if err != nil {
		return "", err
	}
	return providerConfig.Bucket, nil
}
