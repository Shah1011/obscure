package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	cfg "github.com/shah1011/obscure/internal/config"
)

// S3CompatibleClient wraps the S3 client configured for any S3-compatible service
type S3CompatibleClient struct {
	client *s3.Client
	bucket string
}

// NewS3CompatibleClient creates a new S3-compatible client using AWS SDK v2
func NewS3CompatibleClient(ctx context.Context, provider string) (*S3CompatibleClient, error) {
	// Get provider configuration
	providerConfig, err := cfg.GetProviderConfig(provider)
	if err != nil {
		return nil, err
	}

	// Determine endpoint
	endpoint := providerConfig.S3CompatibleEndpoint
	if provider == "filebase-ipfs" {
		endpoint = providerConfig.FilebaseEndpoint
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
			URL:               endpoint,
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

	return &S3CompatibleClient{
		client: client,
		bucket: providerConfig.Bucket,
	}, nil
}

// UploadFile uploads a file to S3-compatible storage
func (s *S3CompatibleClient) UploadFile(ctx context.Context, key string, reader io.Reader, metadata map[string]string) error {
	// Convert metadata to AWS format
	awsMetadata := make(map[string]string)
	for k, v := range metadata {
		awsMetadata[k] = v
	}

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(key),
		Body:     reader,
		Metadata: awsMetadata,
	})
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "AccessDenied") {
			return fmt.Errorf("access denied: please check your Filebase credentials, bucket name, and permissions. The bucket must exist and your access key must have write permissions")
		}
		if strings.Contains(errMsg, "NoSuchBucket") {
			return fmt.Errorf("bucket not found: the specified Filebase bucket does not exist. Please create it in the Filebase dashboard and check the name")
		}
		if strings.Contains(errMsg, "InvalidAccessKeyId") || strings.Contains(errMsg, "SignatureDoesNotMatch") {
			return fmt.Errorf("invalid credentials: the provided Filebase access key or secret key is incorrect. Please verify your credentials")
		}
		return fmt.Errorf("filebase upload error: %v", err)
	}
	return nil
}

// FileExists checks if a file exists in S3-compatible storage
func (s *S3CompatibleClient) FileExists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
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

// ListFiles lists files in S3-compatible storage with a prefix
func (s *S3CompatibleClient) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	var files []string
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
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

// DeleteFile deletes a file from S3-compatible storage
func (s *S3CompatibleClient) DeleteFile(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

// DownloadFile downloads a file from S3-compatible storage
func (s *S3CompatibleClient) DownloadFile(ctx context.Context, key string) (io.ReadCloser, error) {
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}
