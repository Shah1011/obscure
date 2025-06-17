package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	cfg "github.com/shah1011/obscure/internal/config"
)

// StorjClient wraps the S3 client configured for Storj
type StorjClient struct {
	client *s3.S3
	bucket string
}

// NewStorjClient creates a new Storj client using AWS SDK v1
func NewStorjClient(ctx context.Context, provider string) (*StorjClient, error) {
	// Get provider configuration
	providerConfig, err := cfg.GetProviderConfig(provider)
	if err != nil {
		return nil, err
	}

	// Create AWS session with custom configuration
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String(providerConfig.Region),
		Credentials:      credentials.NewStaticCredentials(providerConfig.AccessKeyID, providerConfig.SecretAccessKey, ""),
		Endpoint:         aws.String(providerConfig.StorjEndpoint),
		S3ForcePathStyle: aws.Bool(true),
		DisableSSL:       aws.Bool(false),
	})
	if err != nil {
		return nil, err
	}

	client := s3.New(sess)

	return &StorjClient{
		client: client,
		bucket: providerConfig.Bucket,
	}, nil
}

// UploadFile uploads a file to Storj
func (s *StorjClient) UploadFile(ctx context.Context, key string, reader io.Reader, metadata map[string]string) error {
	// Convert metadata to S3 format
	s3Metadata := make(map[string]*string)
	for k, v := range metadata {
		s3Metadata[k] = aws.String(v)
	}

	// Read all content into memory to create a ReadSeeker
	content, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read content: %w", err)
	}
	seekReader := bytes.NewReader(content)

	// Upload to S3
	_, err = s.client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(key),
		Body:     seekReader,
		Metadata: s3Metadata,
	})
	return err
}

// FileExists checks if a file exists in Storj
func (s *StorjClient) FileExists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check for file not found error
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ListFiles lists files in Storj with a prefix
func (s *StorjClient) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	var files []string

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	}

	err := s.client.ListObjectsV2PagesWithContext(ctx, input, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, obj := range page.Contents {
			files = append(files, *obj.Key)
		}
		return !lastPage
	})

	return files, err
}

// DownloadFile downloads a file from Storj
func (s *StorjClient) DownloadFile(ctx context.Context, key string) (io.ReadCloser, error) {
	result, err := s.client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	return result.Body, nil
}

// GetFileSize gets the size of a file in Storj
func (s *StorjClient) GetFileSize(ctx context.Context, key string) (int64, error) {
	result, err := s.client.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return 0, err
	}

	return *result.ContentLength, nil
}

// DeleteFile deletes a file from Storj
func (s *StorjClient) DeleteFile(ctx context.Context, key string) error {
	_, err := s.client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}
