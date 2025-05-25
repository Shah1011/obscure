package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/shah1011/obscure/internal/auth"
	"gopkg.in/yaml.v3"
)

type User struct {
	Email string `json:"email"`
}

type Config struct {
	Session struct {
		Email    string `yaml:"email"`
		Username string `yaml:"username"`
	} `yaml:"session"`
}

var configPath = filepath.Join(os.Getenv("HOME"), ".obscure", "config.yaml")

func loadConfig() (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func saveConfig(cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

func GetSessionEmail() (string, error) {
	cfg, err := loadConfig()
	if err != nil {
		return "", err
	}
	if cfg.Session.Email == "" {
		return "", errors.New("no email is logged in")
	}
	return cfg.Session.Email, nil
}

func SetSessionEmail(email string) error {
	cfg := &Config{}
	if _, err := os.Stat(configPath); err == nil {
		existing, err := loadConfig()
		if err == nil {
			cfg = existing
		}
	}
	cfg.Session.Email = email
	return saveConfig(cfg)
}

func ClearSessionEmail() error {
	cfg := &Config{}
	if _, err := os.Stat(configPath); err == nil {
		existing, err := loadConfig()
		if err == nil {
			cfg = existing
		}
	}
	cfg.Session.Email = ""
	cfg.Session.Username = ""
	return saveConfig(cfg)
}

func GetSessionUsername() (string, error) {
	cfg, err := loadConfig()
	if err != nil {
		return "", err
	}
	if cfg.Session.Username == "" {
		return "", errors.New("no username is logged in")
	}
	return cfg.Session.Username, nil
}

func SetSessionUsername(username string) error {
	cfg := &Config{}
	if _, err := os.Stat(configPath); err == nil {
		existing, err := loadConfig()
		if err == nil {
			cfg = existing
		}
	}
	cfg.Session.Username = username
	return saveConfig(cfg)
}

func getUsersFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".obscure", "users.json")
}

func IsUserSignedUp(email string) (bool, error) {
	filePath := getUsersFilePath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	var users map[string]*auth.User
	err = json.Unmarshal(data, &users)
	if err != nil {
		return false, err
	}

	for _, user := range users {
		if user.Email == email {
			return true, nil
		}
	}
	return false, nil
}

func GetUsernameByEmail(email string) (string, error) {
	filePath := getUsersFilePath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	var users map[string]*auth.User
	err = json.Unmarshal(data, &users)
	if err != nil {
		return "", err
	}

	for _, user := range users {
		if user.Email == email {
			return user.Username, nil
		}
	}
	return "", errors.New("username not found for email")
}
