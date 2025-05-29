package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const awsRegion = "us-east-1"

func UploadToS3Backend(data []byte, username, tag, version, uploadURL, authToken string, isDirect bool) error {
	// In UploadToS3Backend, right before making the request:
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	// Add file field with appropriate extension
	extension := "obscure"
	if isDirect {
		extension = "tar"
	}
	part, err := writer.CreateFormFile("file", fmt.Sprintf("backup.%s", extension))
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}

	if _, err := io.Copy(part, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("write to form file: %w", err)
	}

	// Add metadata fields
	_ = writer.WriteField("username", username)
	_ = writer.WriteField("tag", tag)
	_ = writer.WriteField("version", version)
	_ = writer.WriteField("is_direct", fmt.Sprintf("%v", isDirect))

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close writer: %w", err)
	}

	// Send request
	req, err := http.NewRequest("POST", uploadURL, &b)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+authToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Println("\n📡 S3 Backend response:", resp.Status)
	fmt.Println(string(respBody))

	if resp.StatusCode == http.StatusUnauthorized && strings.Contains(string(respBody), "Invalid Firebase ID token") {
		return fmt.Errorf("session expired: please run 'obscure login' to authenticate again")
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("S3 backend returned non-201: %s", resp.Status)
	}

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
