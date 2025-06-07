package storage

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	cfg "github.com/shah1011/obscure/internal/config"
)

func NewAWSClient(ctx context.Context, provider string) (*aws.Config, error) {
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

	// Load AWS configuration with custom credentials
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(providerConfig.Region),
		config.WithCredentialsProvider(customCredentials),
	)
	if err != nil {
		return nil, err
	}

	return &awsCfg, nil
}
