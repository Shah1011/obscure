package auth

import (
	"os"
	"path/filepath"
)

func getUsersFilePath() string {
	dirname, err := os.UserHomeDir()
	if err != nil {
		panic("❌ Failed to get user home directory")
	}
	return filepath.Join(dirname, ".obscure", "users.json")
}
