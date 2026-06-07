// Package service contains the crypto utilities used to protect sensitive
// credentials stored in MongoDB. All GitHub OAuth access tokens are encrypted
// before persistence and decrypted only when a GitHub API call is needed.
//
// Algorithm: AES-256-GCM (authenticated encryption).
//   - AES block size: 256 bits (32-byte key).
//   - GCM provides confidentiality AND integrity: tampered ciphertext is
//     detected and rejected during decryption.
//   - A fresh random 96-bit (12-byte) nonce is generated for every encryption
//     so that encrypting the same token twice produces different ciphertext.
//   - The nonce is prepended to the ciphertext before base64 encoding, so the
//     stored string is self-contained and can be decrypted without extra state.
package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// EncryptToken encrypts plaintext using AES-256-GCM and returns a
// base64-URL-encoded string of the form: base64(nonce || ciphertext || tag).
//
// The key must be exactly 32 bytes; any other length is rejected to prevent
// accidentally using AES-128 or AES-192 without realising it.
//
// The nonce is generated using crypto/rand, making it cryptographically random
// and extremely unlikely to repeat within the lifetime of the application.
func EncryptToken(plaintext, key string) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("EncryptToken: key must be exactly 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", fmt.Errorf("EncryptToken aes.NewCipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("EncryptToken cipher.NewGCM: %w", err)
	}

	// Generate a cryptographically random nonce of the size required by GCM.
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("EncryptToken nonce generation: %w", err)
	}

	// Seal appends the encrypted ciphertext and GCM authentication tag to nonce.
	// The resulting byte slice layout is: [nonce | ciphertext | tag].
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode as URL-safe base64 so the result is safe to store in MongoDB and
	// print in logs (though the value itself is never logged).
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// DecryptToken reverses EncryptToken: it base64-decodes the stored string,
// extracts the nonce from the leading bytes, and decrypts the remainder.
//
// An error is returned if:
//   - The key is not 32 bytes.
//   - The base64 encoding is corrupt.
//   - The ciphertext is too short to contain a valid nonce.
//   - The GCM authentication tag does not match (i.e. the ciphertext has been
//     tampered with or the wrong key was provided).
func DecryptToken(ciphertext, key string) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("DecryptToken: key must be exactly 32 bytes, got %d", len(key))
	}

	data, err := base64.URLEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("DecryptToken base64 decode: %w", err)
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", fmt.Errorf("DecryptToken aes.NewCipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("DecryptToken cipher.NewGCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("DecryptToken: ciphertext too short to contain nonce")
	}

	// Split the stored bytes back into nonce and actual ciphertext+tag.
	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		// GCM authentication failure — the token may have been tampered with
		// or the key has been rotated since the token was stored.
		return "", fmt.Errorf("DecryptToken gcm.Open (authentication failure or wrong key): %w", err)
	}

	return string(plaintext), nil
}
