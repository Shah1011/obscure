package utils

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func UploadToS3(keyName, filePath string) error {
	bucketName := os.Getenv("OBSCURE_BUCKET_NAME")
	awsRegion := os.Getenv("AWS_REGION")

	if bucketName == "" || awsRegion == "" {
		return fmt.Errorf("environment variables OBSCURE_BUCKET_NAME and AWS_REGION must be set")
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(awsRegion),
	)
	if err != nil {
		return fmt.Errorf("unable to load SDK config: %w", err)
	}

	client := s3.NewFromConfig(cfg)

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %q: %w", filePath, err)
	}
	defer file.Close()

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(keyName),
		Body:        file,
		ACL:         types.ObjectCannedACLPrivate,
		ContentType: aws.String("application/octet-stream"),
	})
	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}

	fmt.Printf("✅ Successfully uploaded to S3:\n🔗 https://%s.s3.%s.amazonaws.com/%s\n", bucketName, awsRegion, keyName)
	return nil
}

func DownloadFromS3(bucketName, keyName, filePath string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return err
	}

	client := s3.NewFromConfig(cfg)
	output, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: &bucketName,
		Key:    &keyName,
	})
	if err != nil {
		return err
	}
	defer output.Body.Close()

	outFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, output.Body)
	return err
}
