package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shah1011/obscure/internal/auth"
	"gopkg.in/yaml.v3"
)

type User struct {
	Email string `json:"email"`
}

type Config struct {
	Session *struct {
		Email          string `yaml:"email"`
		Username       string `yaml:"username"`
		ActiveProvider string `yaml:"active_provider"`
	} `yaml:"session"`
	User *struct {
		DefaultProvider string `yaml:"default_provider"`
	} `yaml:"user"`
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

	// Initialize nested structs if they're nil
	if cfg.Session == nil {
		cfg.Session = &struct {
			Email          string `yaml:"email"`
			Username       string `yaml:"username"`
			ActiveProvider string `yaml:"active_provider"`
		}{}
	}
	if cfg.User == nil {
		cfg.User = &struct {
			DefaultProvider string `yaml:"default_provider"`
		}{}
	}

	return &cfg, nil
}

func saveConfig(cfg *Config) error {
	// Initialize nested structs if they're nil before saving
	if cfg.Session == nil {
		cfg.Session = &struct {
			Email          string `yaml:"email"`
			Username       string `yaml:"username"`
			ActiveProvider string `yaml:"active_provider"`
		}{}
	}
	if cfg.User == nil {
		cfg.User = &struct {
			DefaultProvider string `yaml:"default_provider"`
		}{}
	}

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
	if cfg.Session == nil || cfg.Session.Email == "" {
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

	// Initialize Session if nil
	if cfg.Session == nil {
		cfg.Session = &struct {
			Email          string `yaml:"email"`
			Username       string `yaml:"username"`
			ActiveProvider string `yaml:"active_provider"`
		}{}
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

	// Initialize Session if nil
	if cfg.Session == nil {
		cfg.Session = &struct {
			Email          string `yaml:"email"`
			Username       string `yaml:"username"`
			ActiveProvider string `yaml:"active_provider"`
		}{}
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
	if cfg.Session == nil || cfg.Session.Username == "" {
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

	// Initialize Session if nil
	if cfg.Session == nil {
		cfg.Session = &struct {
			Email          string `yaml:"email"`
			Username       string `yaml:"username"`
			ActiveProvider string `yaml:"active_provider"`
		}{}
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
	var users map[string]auth.User
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
	var users map[string]auth.User
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

func GetUserDataByEmail(email string) (*UserData, error) {
	filePath := getUsersFilePath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var users map[string]auth.User
	err = json.Unmarshal(data, &users)
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		if user.Email == email {
			// Get provider from local config
			provider, _ := GetUserDefaultProvider()
			return &UserData{
				Username: user.Username,
				Email:    user.Email,
				Provider: provider,
			}, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

type UserData struct {
	Username string `firestore:"username"`
	Email    string `firestore:"email"`
	Provider string `firestore:"defaultProvider"`
}

func SetUserDefaultProvider(provider string) error {
	cfg := &Config{}
	if _, err := os.Stat(configPath); err == nil {
		existing, err := loadConfig()
		if err == nil {
			cfg = existing
		}
	}

	// Initialize User if nil
	if cfg.User == nil {
		cfg.User = &struct {
			DefaultProvider string `yaml:"default_provider"`
		}{}
	}

	cfg.User.DefaultProvider = provider
	return saveConfig(cfg)
}

func SetSessionProvider(provider string) error {
	cfg := &Config{}
	if _, err := os.Stat(configPath); err == nil {
		existing, err := loadConfig()
		if err == nil {
			cfg = existing
		}
	}

	// Initialize Session if nil
	if cfg.Session == nil {
		cfg.Session = &struct {
			Email          string `yaml:"email"`
			Username       string `yaml:"username"`
			ActiveProvider string `yaml:"active_provider"`
		}{}
	}

	cfg.Session.ActiveProvider = provider
	return saveConfig(cfg)
}

func GetUserDefaultProvider() (string, error) {
	cfg, err := loadConfig()
	if err != nil {
		return "", err
	}
	if cfg.User == nil {
		return "", errors.New("no user configuration found")
	}
	return cfg.User.DefaultProvider, nil
}

func GetSessionProvider() (string, error) {
	cfg, err := loadConfig()
	if err != nil {
		return "", err
	}
	if cfg.Session == nil {
		return "", errors.New("no session configuration found")
	}
	return cfg.Session.ActiveProvider, nil
}

func set(key, value string) error {
	cfg := &Config{}
	if _, err := os.Stat(configPath); err == nil {
		existing, err := loadConfig()
		if err == nil {
			cfg = existing
		}
	}

	if cfg.Session == nil {
		cfg.Session = &struct {
			Email          string `yaml:"email"`
			Username       string `yaml:"username"`
			ActiveProvider string `yaml:"active_provider"`
		}{}
	}

	switch key {
	case "session.token":
		// Store token in a separate file for security
		tokenPath := filepath.Join(filepath.Dir(configPath), "token")
		return os.WriteFile(tokenPath, []byte(value), 0600)
	default:
		return fmt.Errorf("unknown key: %s", key)
	}
}

func get(key string) (string, error) {
	switch key {
	case "session.token":
		tokenPath := filepath.Join(filepath.Dir(configPath), "token")
		data, err := os.ReadFile(tokenPath)
		if err != nil {
			return "", err
		}
		return string(data), nil
	default:
		return "", fmt.Errorf("unknown key: %s", key)
	}
}

func SetSessionToken(token string) error {
	return set("session.token", token)
}

func GetSessionToken() (string, error) {
	return get("session.token")
}
