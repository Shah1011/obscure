package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/manifoldco/promptui"
)

type BackupInfo struct {
	Key      string
	Size     int64
	Modified time.Time
	Metadata map[string]string
}

type StorageProvider interface {
	Upload(filePath string, metadata map[string]string) (string, error)
	Download(objectKey string, destinationPath string) error
	List(prefix string) ([]BackupInfo, error)
}

func FetchUserDefaultProvider(email string) (string, error) {
	url := fmt.Sprintf("http://localhost:8080/api/users/default-provider?email=%s", email)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return "", nil // No default, fallback to prompt
	}

	if resp.StatusCode != 200 {
		return "", errors.New("failed to get default provider")
	}

	var data struct {
		DefaultProvider string `json:"default_provider"`
	}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return "", err
	}

	return data.DefaultProvider, nil
}

func PromptForCloudProvider() (string, error) {
	providers := []string{"Amazon S3", "Google Cloud Storage", "Backblaze B2"}

	prompt := promptui.Select{
		Label:  "Select Cloud Provider",
		Items:  providers,
		Stdout: os.Stderr,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		return "", err
	}

	switch idx {
	case 0:
		return "Amazon S3", nil
	case 1:
		return "Google Cloud Storage", nil
	case 2:
		return "Backblaze B2", nil
	default:
		return "", errors.New("invalid provider selected")
	}
}
