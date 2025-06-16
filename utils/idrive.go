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

// GetIDriveClient creates an S3 client configured for IDrive E2
func GetIDriveClient() (*s3.Client, error) {
	// Get IDrive provider configuration
	providerConfig, err := cfg.GetProviderConfig("idrive")
	if err != nil {
		return nil, fmt.Errorf("failed to get IDrive provider config: %w", err)
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
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(providerConfig.Region),
		config.WithCredentialsProvider(customCredentials),
		config.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for IDrive: %w", err)
	}

	return s3.NewFromConfig(awsCfg), nil
}

// DownloadFromIDriveStream downloads a file from IDrive E2 and returns a reader
func DownloadFromIDriveStream(bucket, key string) (io.ReadCloser, error) {
	client, err := GetIDriveClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create IDrive client: %w", err)
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	resp, err := client.GetObject(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to get IDrive object: %w", err)
	}

	return resp.Body, nil // stream — remember to defer Close()
}

// CheckIfIDriveObjectExists checks if an object exists in IDrive E2
func CheckIfIDriveObjectExists(bucket, key string) (bool, error) {
	client, err := GetIDriveClient()
	if err != nil {
		return false, fmt.Errorf("failed to create IDrive client: %w", err)
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
		return false, fmt.Errorf("failed to check IDrive object existence: %w", err)
	}

	return true, nil // exists
}

// GetIDriveObjectSize gets the size of an object in IDrive E2
func GetIDriveObjectSize(bucket, key string) (int64, error) {
	client, err := GetIDriveClient()
	if err != nil {
		return 0, fmt.Errorf("failed to create IDrive client: %w", err)
	}

	headResp, err := client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get IDrive object size: %w", err)
	}

	return *headResp.ContentLength, nil
}

// UploadToIDrive uploads a file to IDrive E2
func UploadToIDrive(bucket, key string, reader io.Reader, metadata map[string]string) error {
	client, err := GetIDriveClient()
	if err != nil {
		return fmt.Errorf("failed to create IDrive client: %w", err)
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
		return fmt.Errorf("failed to upload to IDrive: %w", err)
	}

	return nil
}

// DeleteFromIDrive deletes an object from IDrive E2
func DeleteFromIDrive(bucket, key string) error {
	client, err := GetIDriveClient()
	if err != nil {
		return fmt.Errorf("failed to create IDrive client: %w", err)
	}

	_, err = client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from IDrive: %w", err)
	}

	return nil
}

// ListIDriveObjects lists objects in IDrive E2 with a prefix
func ListIDriveObjects(bucket, prefix string) ([]string, error) {
	client, err := GetIDriveClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create IDrive client: %w", err)
	}

	var objects []string
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("failed to list IDrive objects: %w", err)
		}

		for _, obj := range page.Contents {
			objects = append(objects, *obj.Key)
		}
	}

	return objects, nil
}
