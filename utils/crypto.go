package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
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

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM cipher: %w", err)
	}

	// Read all ciphertext
	var ciphertextBuf bytes.Buffer
	if _, err := io.Copy(&ciphertextBuf, encStream); err != nil {
		return nil, fmt.Errorf("failed to read ciphertext: %w", err)
	}

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertextBuf.Bytes(), nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	// Return decrypted bytes as io.Reader
	return bytes.NewReader(plaintext), nil
}

// EncryptStream creates a writer that encrypts data using AES-GCM
func EncryptStream(w io.Writer, password string) (io.WriteCloser, error) {
	salt, err := GenerateSalt()
	if err != nil {
		return nil, err
	}

	// Write salt to output
	if _, err := w.Write(salt); err != nil {
		return nil, fmt.Errorf("failed to write salt: %w", err)
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

	// Write nonce to output
	if _, err := w.Write(nonce); err != nil {
		return nil, fmt.Errorf("failed to write nonce: %w", err)
	}

	// Create a buffer to store all plaintext
	var plaintextBuf bytes.Buffer

	return &encryptWriter{
		writer:       w,
		gcm:          gcm,
		nonce:        nonce,
		plaintextBuf: &plaintextBuf,
	}, nil
}

type encryptWriter struct {
	writer       io.Writer
	gcm          cipher.AEAD
	nonce        []byte
	plaintextBuf *bytes.Buffer
}

func (w *encryptWriter) Write(p []byte) (int, error) {
	// Write to plaintext buffer
	n, err := w.plaintextBuf.Write(p)
	if err != nil {
		return n, err
	}
	return n, nil
}

func (w *encryptWriter) Close() error {
	// Encrypt the entire plaintext at once
	ciphertext := w.gcm.Seal(nil, w.nonce, w.plaintextBuf.Bytes(), nil)

	// Write the ciphertext
	_, err := w.writer.Write(ciphertext)
	return err
}

// NewCompressWriter creates a zstd compression writer
func NewCompressWriter(w io.Writer) io.WriteCloser {
	encoder, _ := zstd.NewWriter(w)
	return encoder
}
