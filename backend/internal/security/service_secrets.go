package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	servicesEncryptionSalt = "panel-services-v1"
	servicesEncryptionInfo = "services-encryption"
)

type ServiceSecretsCipher struct {
	aead cipher.AEAD
}

func NewServiceSecretsCipher(jwtSecret string) (*ServiceSecretsCipher, error) {
	key, err := deriveServicesKey(jwtSecret)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM cipher: %w", err)
	}

	return &ServiceSecretsCipher{aead: aead}, nil
}

func deriveServicesKey(jwtSecret string) ([]byte, error) {
	if strings.TrimSpace(jwtSecret) == "" {
		return nil, errors.New("JWT secret cannot be empty")
	}

	key, err := hkdf.Key(
		sha256.New,
		[]byte(jwtSecret),
		[]byte(servicesEncryptionSalt),
		servicesEncryptionInfo,
		32,
	)
	if err != nil {
		return nil, fmt.Errorf("derive services encryption key: %w", err)
	}

	return key, nil
}

func (c *ServiceSecretsCipher) EncryptString(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := c.aead.Seal(nil, nonce, []byte(plaintext), nil)
	encodedNonce := base64.RawStdEncoding.EncodeToString(nonce)
	encodedCiphertext := base64.RawStdEncoding.EncodeToString(ciphertext)

	return encodedNonce + ":" + encodedCiphertext, nil
}

func (c *ServiceSecretsCipher) DecryptString(payload string) (string, error) {
	if payload == "" {
		return "", nil
	}

	parts := strings.SplitN(payload, ":", 2)
	if len(parts) != 2 {
		return "", errors.New("invalid encrypted payload format")
	}

	nonce, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return "", fmt.Errorf("decode nonce: %w", err)
	}

	ciphertext, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}

	plaintext, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt payload: %w", err)
	}

	return string(plaintext), nil
}
