package utils

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const awsRegion = "us-east-1"

func UploadToS3(data io.ReadSeeker, bucketName string, s3Key string) error {

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

func GetS3Client() *s3.Client {
	cfg, _ := config.LoadDefaultConfig(context.TODO(), config.WithRegion(awsRegion))
	return s3.NewFromConfig(cfg)
}

func DownloadFromS3Stream(bucket, key string) (io.ReadCloser, error) {
	client := GetS3Client()

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	resp, err := client.GetObject(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to get S3 object: %w", err)
	}

	return resp.Body, nil // stream — remember to defer Close()
}

func CheckIfS3ObjectExists(bucket, key string) (bool, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return false, err
	}

	client := s3.NewFromConfig(cfg)

	_, err = client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return false, nil // doesn't exist
		}
		return false, err // some other error
	}

	return true, nil // exists
}
