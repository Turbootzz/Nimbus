package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
)

// EncryptionKeyEnv holds the base64 AES-256 key used for secrets at rest.
const EncryptionKeyEnv = "ENCRYPTION_KEY"

// cipherVersion is the first byte of every blob so the format or key can be
// rotated later without guessing what old rows contain.
const cipherVersion byte = 0x01

// ErrDecrypt covers every decrypt failure (wrong key, tampering, bad format)
// so callers can't leak which check failed.
var ErrDecrypt = errors.New("failed to decrypt data")

// Cipher encrypts small secrets with AES-256-GCM.
// Blob format: version (1 byte) || nonce (12 bytes) || ciphertext+tag.
type Cipher struct {
	aead cipher.AEAD
}

// ParseEncryptionKey decodes a base64 key and checks it is 32 bytes.
func ParseEncryptionKey(encoded string) ([]byte, error) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return nil, fmt.Errorf("%s is required (generate with: openssl rand -base64 32)", EncryptionKeyEnv)
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("%s must be 32 bytes encoded as base64 (generate with: openssl rand -base64 32)", EncryptionKeyEnv)
	}
	return key, nil
}

// NewCipher builds a Cipher from a raw 32-byte key.
func NewCipher(key []byte) (*Cipher, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

// NewCipherFromEnv builds a Cipher from ENCRYPTION_KEY.
func NewCipherFromEnv() (*Cipher, error) {
	key, err := ParseEncryptionKey(os.Getenv(EncryptionKeyEnv))
	if err != nil {
		return nil, err
	}
	return NewCipher(key)
}

// Encrypt seals plaintext with a fresh random nonce.
func (c *Cipher) Encrypt(plaintext []byte) ([]byte, error) {
	nonceSize := c.aead.NonceSize()
	out := make([]byte, 1+nonceSize, 1+nonceSize+len(plaintext)+c.aead.Overhead())
	out[0] = cipherVersion
	if _, err := rand.Read(out[1:]); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	return c.aead.Seal(out, out[1:], plaintext, nil), nil
}

// Decrypt opens a blob produced by Encrypt.
func (c *Cipher) Decrypt(blob []byte) ([]byte, error) {
	nonceSize := c.aead.NonceSize()
	if len(blob) < 1+nonceSize+c.aead.Overhead() || blob[0] != cipherVersion {
		return nil, ErrDecrypt
	}
	plaintext, err := c.aead.Open(nil, blob[1:1+nonceSize], blob[1+nonceSize:], nil)
	if err != nil {
		return nil, ErrDecrypt
	}
	return plaintext, nil
}
