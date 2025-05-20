package utils

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// GetUserID retrieves the AWS user ID using STS
func GetUserID() (string, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}
	client := sts.NewFromConfig(cfg)
	resp, err := client.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", fmt.Errorf("failed to get AWS user identity: %w", err)
	}
	return *resp.UserId, nil
}
