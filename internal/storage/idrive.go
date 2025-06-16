package storage

import (
	"context"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	cfg "github.com/shah1011/obscure/internal/config"
)

// IDriveClient wraps the S3 client configured for IDrive E2
type IDriveClient struct {
	client *s3.Client
	bucket string
}

// NewIDriveClient creates a new IDrive E2 client using AWS SDK v2
func NewIDriveClient(ctx context.Context, provider string) (*IDriveClient, error) {
	// Get provider configuration
	providerConfig, err := cfg.GetProviderConfig(provider)
	if err != nil {
		return nil, err
	}

	// Create custom credentials
	customCredentials := credentials.NewStaticCredentialsProvider(
		providerConfig.AccessKeyID,
		providerConfig.SecretAccessKey,
		"", // session token (optional)
	)

	// Create custom endpoint resolver
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               providerConfig.IDriveEndpoint,
			SigningRegion:     providerConfig.Region,
			HostnameImmutable: true,
		}, nil
	})

	// Load AWS configuration with custom settings
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(providerConfig.Region),
		config.WithCredentialsProvider(customCredentials),
		config.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg)

	return &IDriveClient{
		client: client,
		bucket: providerConfig.Bucket,
	}, nil
}

// UploadFile uploads a file to IDrive E2
func (i *IDriveClient) UploadFile(ctx context.Context, key string, reader io.Reader, metadata map[string]string) error {
	// Convert metadata to AWS format
	awsMetadata := make(map[string]string)
	for k, v := range metadata {
		awsMetadata[k] = v
	}

	_, err := i.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:   aws.String(i.bucket),
		Key:      aws.String(key),
		Body:     reader,
		Metadata: awsMetadata,
	})
	return err
}

// FileExists checks if a file exists in IDrive E2
func (i *IDriveClient) FileExists(ctx context.Context, key string) (bool, error) {
	_, err := i.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(i.bucket),
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

// ListFiles lists files in IDrive E2 with a prefix
func (i *IDriveClient) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	var files []string
	paginator := s3.NewListObjectsV2Paginator(i.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(i.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, obj := range page.Contents {
			files = append(files, *obj.Key)
		}
	}

	return files, nil
}

// DeleteFile deletes a file from IDrive E2
func (i *IDriveClient) DeleteFile(ctx context.Context, key string) error {
	_, err := i.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(i.bucket),
		Key:    aws.String(key),
	})
	return err
}

// DownloadFile downloads a file from IDrive E2
func (i *IDriveClient) DownloadFile(ctx context.Context, key string) (io.ReadCloser, error) {
	resp, err := i.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(i.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}
