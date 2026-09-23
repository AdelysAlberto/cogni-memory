package network

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
)

const (
	saltSize  = 16
	nonceSize = 12
)

// deriveKey derives a 32-byte AES-256 key from the pairing code and random salt using iterated SHA-256
func deriveKey(code string, salt []byte) []byte {
	h := sha256.New()
	h.Write([]byte(code))
	h.Write(salt)
	key := h.Sum(nil)

	// Stretch key
	for i := 0; i < 4096; i++ {
		h.Reset()
		h.Write(key)
		h.Write(salt)
		key = h.Sum(nil)
	}
	return key
}

// Encrypt encrypts data with AES-256-GCM using a key derived from the pairing code.
// Output layout: [16 bytes salt] + [12 bytes nonce] + [ciphertext + 16 bytes GCM tag]
func Encrypt(plaintext []byte, code string) ([]byte, error) {
	if code == "" {
		return nil, errors.New("empty pairing code")
	}

	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate random salt: %w", err)
	}

	key := deriveKey(code, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM block: %w", err)
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate random nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Combine salt + nonce + ciphertext
	out := make([]byte, 0, saltSize+nonceSize+len(ciphertext))
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)

	return out, nil
}

// Decrypt authenticates and decrypts AES-256-GCM ciphertext using the pairing code.
func Decrypt(payload []byte, code string) ([]byte, error) {
	if len(payload) < saltSize+nonceSize {
		return nil, errors.New("payload too short for decryption")
	}

	salt := payload[:saltSize]
	nonce := payload[saltSize : saltSize+nonceSize]
	ciphertext := payload[saltSize+nonceSize:]

	key := deriveKey(code, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM block: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("autenticación fallida: código incorrecto o datos alterados")
	}

	return plaintext, nil
}
