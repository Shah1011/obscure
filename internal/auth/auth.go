package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"
)

type User struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"` // You can hash later
	Verified bool   `json:"verified"`
}

var codes = make(map[string]string)

func loadUsers() (map[string]*User, error) {
	users := make(map[string]*User)
	path := getUsersFilePath()
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if len(data) > 0 {
		err = json.Unmarshal(data, &users) // users is map[string]*User here
		if err != nil {
			return nil, err
		}
	}
	return users, nil
}

func saveUsers(users map[string]*User) error {
	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getUsersFilePath(), data, 0600)
}

func SendVerificationCode(email string) (string, error) {
	code := generateCode()
	codes[email] = code
	return code, nil // simulate sending
}

func VerifyCode(email, code string) bool {
	storedCode, exists := codes[email]
	return exists && storedCode == code
}

func SaveUser(email, username, password string) error {
	users, err := loadUsers()
	if err != nil {
		return err
	}

	// Check again for uniqueness
	if _, exists := users[email]; exists {
		return errors.New("user already exists")
	}
	for _, u := range users {
		if u.Username == username {
			return errors.New("username already taken")
		}
	}

	users[email] = &User{
		Email:    email,
		Username: username,
		Password: password,
		Verified: true,
	}

	return saveUsers(users) // write to users.json
}

func IsUserVerified(email, password string) bool {
	users, err := loadUsers()
	if err != nil {
		return false
	}
	user, exists := users[email]
	return exists && user.Verified && user.Password == password
}

func UserExists(email string) bool {
	users, err := loadUsers()
	if err != nil {
		// optionally log error or treat as no users found
		return false
	}
	_, exists := users[email]
	return exists
}

func UsernameExists(username string) bool {
	users, err := loadUsers()
	if err != nil {
		return false
	}
	for _, user := range users {
		if user.Username == username {
			return true
		}
	}
	return false
}

func GetUsernameByEmail(email string) (string, error) {
	users, err := loadUsers()
	if err != nil {
		return "", err
	}
	user, exists := users[email]
	if !exists {
		return "", errors.New("user not found")
	}
	return user.Username, nil
}

func CheckPassword(email, password string) bool {
	users, err := loadUsers()
	if err != nil {
		return false
	}
	user, exists := users[email]
	return exists && user.Password == password
}

func generateCode() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}
