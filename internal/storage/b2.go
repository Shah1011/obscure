package storage

import (
	"context"
	"io"

	"github.com/kurin/blazer/b2"
	cfg "github.com/shah1011/obscure/internal/config"
)

// B2Client wraps the B2 client for easier use
type B2Client struct {
	client *b2.Client
	bucket *b2.Bucket
}

// NewB2Client creates a new B2 client using the official B2 SDK
func NewB2Client(ctx context.Context, provider string) (*B2Client, error) {
	// Get provider configuration
	providerConfig, err := cfg.GetProviderConfig(provider)
	if err != nil {
		return nil, err
	}

	// Create B2 client using official SDK
	client, err := b2.NewClient(ctx, providerConfig.ApplicationKeyID, providerConfig.ApplicationKey)
	if err != nil {
		return nil, err
	}

	// Get the bucket
	bucket, err := client.Bucket(ctx, providerConfig.Bucket)
	if err != nil {
		return nil, err
	}

	return &B2Client{
		client: client,
		bucket: bucket,
	}, nil
}

// UploadFile uploads a file to B2
func (b *B2Client) UploadFile(ctx context.Context, key string, reader io.Reader, metadata map[string]string) error {
	// Create object writer
	obj := b.bucket.Object(key)
	writer := obj.NewWriter(ctx)

	// Copy data
	if _, err := io.Copy(writer, reader); err != nil {
		return err
	}

	// Close writer to finalize upload
	return writer.Close()
}

// FileExists checks if a file exists in B2
func (b *B2Client) FileExists(ctx context.Context, key string) (bool, error) {
	// Instead of using Attrs which can cause 416 errors, use ListFiles to check existence
	// This is more reliable for B2
	files, err := b.ListFiles(ctx, key)
	if err != nil {
		return false, err
	}
	
	// Check if the exact key exists in the list
	for _, file := range files {
		if file == key {
			return true, nil
		}
	}
	
	return false, nil
}

// ListFiles lists files in B2 with a prefix
func (b *B2Client) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	var files []string

	iter := b.bucket.List(ctx, b2.ListPrefix(prefix))
	for iter.Next() {
		files = append(files, iter.Object().Name())
	}

	if err := iter.Err(); err != nil {
		return nil, err
	}

	return files, nil
}

// GetFileMetadata gets metadata for a file
func (b *B2Client) GetFileMetadata(ctx context.Context, key string) (map[string]string, error) {
	obj := b.bucket.Object(key)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return nil, err
	}

	// Convert B2 custom headers to metadata
	metadata := make(map[string]string)
	for k, v := range attrs.Info {
		// Remove "X-Bz-Info-" prefix
		if len(k) > 11 && k[:11] == "X-Bz-Info-" {
			metadata[k[11:]] = v
		}
	}

	return metadata, nil
}

// DeleteFile deletes a file from B2
func (b *B2Client) DeleteFile(ctx context.Context, key string) error {
	obj := b.bucket.Object(key)
	return obj.Delete(ctx)
}
