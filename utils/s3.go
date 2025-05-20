package utils

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func UploadToS3(data io.ReadSeeker, bucketName string, s3Key string) error {
	awsRegion := "us-east-1"

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(awsRegion))
	if err != nil {
		return fmt.Errorf("unable to load SDK config: %w", err)
	}

	client := s3.NewFromConfig(cfg)

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:        aws.String(bucketName),
		Key:           aws.String(s3Key),
		Body:          data,
		ContentType:   aws.String("application/octet-stream"),
		ContentLength: aws.Int64(getReaderLength(data)),
	})
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	fmt.Printf("✅ Uploaded to: https://%s.s3.%s.amazonaws.com/%s\n", bucketName, awsRegion, s3Key)
	return nil
}

func getReaderLength(r io.ReadSeeker) int64 {
	size, _ := r.Seek(0, io.SeekEnd)
	r.Seek(0, io.SeekStart)
	return size
}

func DownloadFromS3(bucket, key string, writer io.Writer) error {
	// Load AWS config
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return err
	}

	client := s3.NewFromConfig(cfg)

	resp, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Copy with progress
	_, err = io.Copy(writer, resp.Body)
	return err
}
