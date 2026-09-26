package utils

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func testCipher(t *testing.T, fill byte) *Cipher {
	t.Helper()
	c, err := NewCipher(bytes.Repeat([]byte{fill}, 32))
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	return c
}

func TestCipher_RoundTrip(t *testing.T) {
	c := testCipher(t, 1)
	for _, plaintext := range [][]byte{[]byte(`{"api_key":"s3cret"}`), {}} {
		blob, err := c.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encrypt: %v", err)
		}
		got, err := c.Decrypt(blob)
		if err != nil {
			t.Fatalf("Decrypt: %v", err)
		}
		if !bytes.Equal(got, plaintext) {
			t.Errorf("round trip mismatch: got %q, want %q", got, plaintext)
		}
	}
}

func TestCipher_VersionByteAndFreshNonce(t *testing.T) {
	c := testCipher(t, 1)
	a, _ := c.Encrypt([]byte("same"))
	b, _ := c.Encrypt([]byte("same"))

	if a[0] != 0x01 {
		t.Errorf("expected version byte 0x01, got %#x", a[0])
	}
	if bytes.Equal(a, b) {
		t.Error("expected different ciphertexts for the same plaintext (random nonce)")
	}
	if bytes.Contains(a, []byte("same")) {
		t.Error("ciphertext must not contain the plaintext")
	}
}

func TestCipher_DetectsTampering(t *testing.T) {
	c := testCipher(t, 1)
	blob, _ := c.Encrypt([]byte("secret"))

	for i := range blob {
		tampered := bytes.Clone(blob)
		tampered[i] ^= 0xff
		if _, err := c.Decrypt(tampered); !errors.Is(err, ErrDecrypt) {
			t.Errorf("flipping byte %d: expected ErrDecrypt, got %v", i, err)
		}
	}
}

func TestCipher_RejectsUnknownVersionAndShortBlobs(t *testing.T) {
	c := testCipher(t, 1)
	blob, _ := c.Encrypt([]byte("secret"))

	v2 := bytes.Clone(blob)
	v2[0] = 0x02
	for name, input := range map[string][]byte{
		"unknown version": v2,
		"empty":           nil,
		"version only":    {0x01},
		"truncated":       blob[:len(blob)-1],
	} {
		if _, err := c.Decrypt(input); !errors.Is(err, ErrDecrypt) {
			t.Errorf("%s: expected ErrDecrypt, got %v", name, err)
		}
	}
}

func TestCipher_WrongKeyFails(t *testing.T) {
	blob, _ := testCipher(t, 1).Encrypt([]byte("secret"))
	if _, err := testCipher(t, 2).Decrypt(blob); !errors.Is(err, ErrDecrypt) {
		t.Errorf("expected ErrDecrypt with wrong key, got %v", err)
	}
}

func TestParseEncryptionKey(t *testing.T) {
	valid := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32))

	key, err := ParseEncryptionKey("  " + valid + "\n")
	if err != nil || len(key) != 32 {
		t.Fatalf("expected valid 32-byte key, got len=%d err=%v", len(key), err)
	}

	for name, input := range map[string]string{
		"missing":      "",
		"not base64":   "not-base64!!",
		"wrong length": base64.StdEncoding.EncodeToString([]byte("too short")),
	} {
		_, err := ParseEncryptionKey(input)
		if err == nil || !strings.Contains(err.Error(), "ENCRYPTION_KEY") {
			t.Errorf("%s: expected ENCRYPTION_KEY error, got %v", name, err)
		}
	}
}

func TestNewCipherFromEnv(t *testing.T) {
	t.Setenv(EncryptionKeyEnv, "")
	if _, err := NewCipherFromEnv(); err == nil {
		t.Error("expected error when ENCRYPTION_KEY is missing")
	}

	t.Setenv(EncryptionKeyEnv, base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32)))
	if _, err := NewCipherFromEnv(); err != nil {
		t.Errorf("expected cipher from valid env, got %v", err)
	}
}
