package utils

import (
	"context"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	cfg "github.com/shah1011/obscure/internal/config"
)

func CheckIfStorjObjectExists(bucket, object string) (bool, error) {
	ctx := context.Background()

	// Get provider configuration
	providerConfig, err := cfg.GetProviderConfig("storj")
	if err != nil {
		return false, err
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
		return false, err
	}

	client := s3.New(sess)

	_, err = client.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(object),
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

func GetStorjObjectSize(bucket, object string) (int64, error) {
	ctx := context.Background()

	// Get provider configuration
	providerConfig, err := cfg.GetProviderConfig("storj")
	if err != nil {
		return 0, err
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
		return 0, err
	}

	client := s3.New(sess)

	result, err := client.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(object),
	})
	if err != nil {
		return 0, err
	}

	return *result.ContentLength, nil
}

func DownloadFromStorjStream(bucket, object string) (io.ReadCloser, error) {
	ctx := context.Background()

	// Get provider configuration
	providerConfig, err := cfg.GetProviderConfig("storj")
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

	result, err := client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(object),
	})
	if err != nil {
		return nil, err
	}

	return result.Body, nil
}
