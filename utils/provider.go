package utils

import (
	"time"
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
