package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"bytes"
	"mime/multipart"
	"net/http"

	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

func CheckIfGCSObjectExists(bucket, object string) (bool, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(os.Getenv("OBSCURE_GCP_CREDENTIALS")))
	if err != nil {
		return false, fmt.Errorf("failed to create GCS client: %w", err)
	}
	defer client.Close()

	_, err = client.Bucket(bucket).Object(object).Attrs(ctx)
	if err != nil {
		var gErr *googleapi.Error
		if errors.As(err, &gErr) {
			if gErr.Code == 404 {
				// Not found, so backup does NOT exist
				return false, nil
			}
		}
		// Other error
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
		return fmt.Errorf("%s", string(respBody))
	}

	return nil
}
