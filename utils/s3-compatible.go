package utils

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	cfg "github.com/shah1011/obscure/internal/config"
)

// GetS3CompatibleClient creates an S3 client configured for any S3-compatible service
func GetS3CompatibleClient() (*s3.Client, error) {
	// Get S3-compatible provider configuration
	providerConfig, err := cfg.GetProviderConfig("s3-compatible")
	if err != nil {
		return nil, fmt.Errorf("failed to get S3-compatible provider config: %w", err)
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
			URL:               providerConfig.S3CompatibleEndpoint,
			SigningRegion:     providerConfig.Region,
			HostnameImmutable: true,
		}, nil
	})

	// Load AWS configuration with custom settings
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(providerConfig.Region),
		config.WithCredentialsProvider(customCredentials),
		config.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for S3-compatible: %w", err)
	}

	return s3.NewFromConfig(awsCfg), nil
}

// DownloadFromS3CompatibleStream downloads a file from S3-compatible storage and returns a reader
func DownloadFromS3CompatibleStream(bucket, key string) (io.ReadCloser, error) {
	client, err := GetS3CompatibleClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create S3-compatible client: %w", err)
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	resp, err := client.GetObject(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to get S3-compatible object: %w", err)
	}

	return resp.Body, nil // stream — remember to defer Close()
}

// CheckIfS3CompatibleObjectExists checks if an object exists in S3-compatible storage
func CheckIfS3CompatibleObjectExists(bucket, key string) (bool, error) {
	client, err := GetS3CompatibleClient()
	if err != nil {
		return false, fmt.Errorf("failed to create S3-compatible client: %w", err)
	}

	_, err = client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return false, nil // doesn't exist
		}
		return false, fmt.Errorf("failed to check S3-compatible object existence: %w", err)
	}

	return true, nil // exists
}

// GetS3CompatibleObjectSize gets the size of an object in S3-compatible storage
func GetS3CompatibleObjectSize(bucket, key string) (int64, error) {
	client, err := GetS3CompatibleClient()
	if err != nil {
		return 0, fmt.Errorf("failed to create S3-compatible client: %w", err)
	}

	headResp, err := client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get S3-compatible object size: %w", err)
	}

	return *headResp.ContentLength, nil
}

// UploadToS3Compatible uploads a file to S3-compatible storage
func UploadToS3Compatible(bucket, key string, reader io.Reader, metadata map[string]string) error {
	client, err := GetS3CompatibleClient()
	if err != nil {
		return fmt.Errorf("failed to create S3-compatible client: %w", err)
	}

	// Convert metadata to AWS format
	awsMetadata := make(map[string]string)
	for k, v := range metadata {
		awsMetadata[k] = v
	}

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		Body:     reader,
		Metadata: awsMetadata,
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3-compatible: %w", err)
	}

	return nil
}

// DeleteFromS3Compatible deletes an object from S3-compatible storage
func DeleteFromS3Compatible(bucket, key string) error {
	client, err := GetS3CompatibleClient()
	if err != nil {
		return fmt.Errorf("failed to create S3-compatible client: %w", err)
	}

	_, err = client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from S3-compatible: %w", err)
	}

	return nil
}

// ListS3CompatibleObjects lists objects in S3-compatible storage with a prefix
func ListS3CompatibleObjects(bucket, prefix string) ([]string, error) {
	client, err := GetS3CompatibleClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create S3-compatible client: %w", err)
	}

	var objects []string
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("failed to list S3-compatible objects: %w", err)
		}

		for _, obj := range page.Contents {
			objects = append(objects, *obj.Key)
		}
	}

	return objects, nil
}
