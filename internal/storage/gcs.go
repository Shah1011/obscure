package storage

import (
	"context"

	"cloud.google.com/go/storage"
	cfg "github.com/shah1011/obscure/internal/config"
	"google.golang.org/api/option"
)

func NewGCSClient(ctx context.Context, provider string) (*storage.Client, error) {
	// Get provider configuration
	providerConfig, err := cfg.GetProviderConfig(provider)
	if err != nil {
		return nil, err
	}

	// Create client with service account credentials
	client, err := storage.NewClient(ctx,
		option.WithCredentialsFile(providerConfig.ServiceAccount),
	)
	if err != nil {
		return nil, err
	}

	return client, nil
}
