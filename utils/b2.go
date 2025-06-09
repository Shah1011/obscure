package utils

import (
	"context"
	"fmt"
	"io"

	"github.com/kurin/blazer/b2"
	cfg "github.com/shah1011/obscure/internal/config"
)

// DownloadFromB2Stream downloads a file from B2 and returns a reader and file size
func DownloadFromB2Stream(objectKey string) (io.ReadCloser, int64, error) {
	ctx := context.Background()

	// Get B2 provider configuration
	providerConfig, err := cfg.GetProviderConfig("b2")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get B2 provider config: %w", err)
	}

	// Create B2 client
	client, err := b2.NewClient(ctx, providerConfig.ApplicationKeyID, providerConfig.ApplicationKey)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create B2 client: %w", err)
	}

	// Get the bucket
	bucket, err := client.Bucket(ctx, providerConfig.Bucket)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get B2 bucket: %w", err)
	}

	// Get the object
	obj := bucket.Object(objectKey)

	// Get object attributes (to retrieve size)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get B2 object attributes: %w", err)
	}

	// Create a streaming reader
	reader := obj.NewReader(ctx)

	return reader, attrs.Size, nil
}
