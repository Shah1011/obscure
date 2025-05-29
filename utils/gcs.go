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
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")))
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
	credPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credPath == "" {
		return nil, fmt.Errorf("missing GOOGLE_APPLICATION_CREDENTIALS environment variable")
	}

	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}
	return client, nil
}

func UploadToGCSBackend(encryptedData []byte, username, tag, version, backendURL, authToken string, isDirect bool) error {
	// Prepare multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add metadata fields
	_ = writer.WriteField("username", username)
	_ = writer.WriteField("tag", tag)
	_ = writer.WriteField("version", version)
	_ = writer.WriteField("is_direct", fmt.Sprintf("%v", isDirect))

	// Add file field with appropriate extension
	extension := "obscure"
	if isDirect {
		extension = "tar"
	}
	part, err := writer.CreateFormFile("file", fmt.Sprintf("%s_v%s.%s", tag, version, extension))
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
	req.Header.Set("Authorization", "Bearer "+authToken)

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

var gcsBucketName = "obscure-open"

func DownloadFromGCSStream(objectKey string) (io.ReadCloser, int64, error) {
	ctx := context.Background()

	// 🧠 Initialize GCS client (assumes ADC or service account key)
	client, err := storage.NewClient(ctx, option.WithCredentialsFile("service-account.json"))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create GCS client: %w", err)
	}

	bucket := client.Bucket(gcsBucketName)
	obj := bucket.Object(objectKey)

	// 🔍 Get object attributes (to retrieve size)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get GCS object attributes: %w", err)
	}

	// 📥 Create a streaming reader
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open GCS object reader: %w", err)
	}

	return reader, attrs.Size, nil
}
