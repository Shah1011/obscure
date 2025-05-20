package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
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
