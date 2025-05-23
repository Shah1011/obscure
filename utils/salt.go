package utils

import (
	"crypto/rand"
	"errors"
	"os"

	"golang.org/x/crypto/scrypt"
)

const (
	KeyLength = 32
)

func GenerateSalt() ([]byte, error) {
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, err
	}
	return salt, nil
}

func DeriveKey(password string, salt []byte) ([]byte, error) {
	if len(salt) == 0 {
		return nil, errors.New("salt is required")
	}
	return scrypt.Key([]byte(password), salt, 1<<15, 8, 1, KeyLength)
}

func ExtractSaltFromEncryptedFile(filepath string) ([]byte, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	salt := make([]byte, 16) // assuming 16-byte salt
	_, err = f.Read(salt)
	if err != nil {
		return nil, err
	}

	return salt, nil
}
