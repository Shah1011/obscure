package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
)

func EncryptBuffer(plainBuf *bytes.Buffer, password string) (*bytes.Buffer, error) {
	salt, err := GenerateSalt()
	if err != nil {
		return nil, err
	}

	key, err := DeriveKey(password, salt)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plainBuf.Bytes(), nil)

	finalBuf := new(bytes.Buffer)
	finalBuf.Write(salt)
	finalBuf.Write(nonce)
	finalBuf.Write(ciphertext)

	return finalBuf, nil
}

// DecryptStream decrypts a streamed .obscure backup from S3.
// Format assumed: [16-byte salt][12-byte nonce][ciphertext...]
func DecryptStream(encStream io.Reader, password string) (io.Reader, error) {
	// Read salt (16 bytes)
	salt := make([]byte, 16)
	if _, err := io.ReadFull(encStream, salt); err != nil {
		return nil, fmt.Errorf("failed to read salt: %w", err)
	}

	// Derive key
	key, err := DeriveKey(password, salt)
	if err != nil {
		return nil, fmt.Errorf("key derivation failed: %w", err)
	}

	// Read nonce (12 bytes)
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(encStream, nonce); err != nil {
		return nil, fmt.Errorf("failed to read nonce: %w", err)
	}

	// Read remaining ciphertext from stream
	ciphertext, err := ioutil.ReadAll(encStream)
	if err != nil {
		return nil, fmt.Errorf("failed to read ciphertext: %w", err)
	}

	// Decrypt
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM cipher: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	// Return decrypted bytes as io.Reader
	return bytes.NewReader(plaintext), nil
}
