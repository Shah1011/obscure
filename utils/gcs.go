package utils

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"bytes"
	"mime/multipart"
	"net/http"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

func UploadToGCS(data io.ReadSeeker, bucketName, objectName string) error {
	ctx := context.Background()

	client, err := GetGCSClient()
	if err != nil {
		return fmt.Errorf("failed to create GCS client: %w", err)
	}
	defer client.Close()

	bucket := client.Bucket(bucketName)
	obj := bucket.Object(objectName)
	w := obj.NewWriter(ctx)

	if _, err := io.Copy(w, data); err != nil {
		return fmt.Errorf("failed to write to GCS: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close GCS writer: %w", err)
	}

	fmt.Println("✅ Uploaded to GCS:", objectName)
	return nil
}

func CheckIfGCSObjectExists(bucket, object string) (bool, error) {
	ctx := context.Background()
	client, err := GetGCSClient()
	if err != nil {
		return false, fmt.Errorf("failed to create GCS client: %w", err)
	}
	defer client.Close()

	_, err = client.Bucket(bucket).Object(object).Attrs(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return false, nil
		}
		// Hide verbose GCS 404 messages for clarity
		if strings.Contains(err.Error(), "Error 404") && strings.Contains(err.Error(), "notFound") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object attributes: %w", err)
	}
	return true, nil
}

func GetGCSClient() (*storage.Client, error) {
	credPath := os.Getenv("OBSCURE_GCP_CREDENTIALS")
	if credPath == "" {
		return nil, fmt.Errorf("missing OBSCURE_GCP_CREDENTIALS environment variable")
	}

	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}
	return client, nil
}

func UploadToGCSBackend(encryptedData []byte, username, tag, version, backendURL string) error {
	// Prepare multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add metadata fields
	_ = writer.WriteField("username", username)
	_ = writer.WriteField("tag", tag)
	_ = writer.WriteField("version", version)

	// Add file field
	part, err := writer.CreateFormFile("file", fmt.Sprintf("%s_v%s.obscure", tag, version))
	if err != nil {
		return err
	}
	_, err = io.Copy(part, bytes.NewReader(encryptedData))
	if err != nil {
		return err
	}
	writer.Close()

	// Send POST request
	req, err := http.NewRequest("POST", backendURL, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}
