package main

import (
	"context"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/gin-gonic/gin"
)

type TokenRequest struct {
	UserID string `json:"user_id"` // From CLI
}

type TokenResponse struct {
	AccessKeyId     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	SessionToken    string `json:"session_token"`
	Expiration      string `json:"expiration"`
}

func main() {
	r := gin.Default()

	r.POST("/generate-token", func(c *gin.Context) {
		var req TokenRequest
		if err := c.BindJSON(&req); err != nil || req.UserID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing user_id"})
			return
		}

		// Load AWS credentials and config
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load AWS config"})
			return
		}

		stsClient := sts.NewFromConfig(cfg)

		roleArn := os.Getenv("OBSCURE_BACKUP_ROLE_ARN") // Set this in env
		sessionName := "obscure-session-" + req.UserID

		assumeResp, err := stsClient.AssumeRole(context.TODO(), &sts.AssumeRoleInput{
			RoleArn:         aws.String(roleArn),
			RoleSessionName: aws.String(sessionName),
			ExternalId:      aws.String(req.UserID),
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assume role"})
			return
		}

		c.JSON(http.StatusOK, TokenResponse{
			AccessKeyId:     *assumeResp.Credentials.AccessKeyId,
			SecretAccessKey: *assumeResp.Credentials.SecretAccessKey,
			SessionToken:    *assumeResp.Credentials.SessionToken,
			Expiration:      assumeResp.Credentials.Expiration.String(),
		})
	})

	r.Run(":8080")
}
