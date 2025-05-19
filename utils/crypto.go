package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
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

func DecryptFile(inputPath, outputPath string, key []byte) error {
	inFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open encrypted file: %w", err)
	}
	defer inFile.Close()

	// Read salt (16 bytes) — already used for key derivation externally
	salt := make([]byte, 16)
	if _, err := io.ReadFull(inFile, salt); err != nil {
		return fmt.Errorf("failed to read salt: %w", err)
	}

	// Read nonce (12 bytes for GCM)
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(inFile, nonce); err != nil {
		return fmt.Errorf("failed to read nonce: %w", err)
	}

	// Read the rest (ciphertext + tag)
	ciphertext, err := io.ReadAll(inFile)
	if err != nil {
		return fmt.Errorf("failed to read ciphertext: %w", err)
	}

	// Prepare AES-GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	// Write decrypted data to .zip
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	_, err = outFile.Write(plaintext)
	if err != nil {
		return fmt.Errorf("failed to write decrypted data: %w", err)
	}

	return nil
}
