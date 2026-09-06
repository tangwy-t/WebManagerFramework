// Package crypto provides password hashing and verification utilities.
package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const saltSize = 16

// HashPassword generates a random salt, then hashes salt+password with bcrypt.
// Returns the bcrypt hash and the hex-encoded salt.
func HashPassword(password string, cost int) (hash string, salt string, err error) {
	saltBytes := make([]byte, saltSize)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", "", fmt.Errorf("generate salt: %w", err)
	}
	salt = hex.EncodeToString(saltBytes)

	salted := salt + password
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(salted), cost)
	if err != nil {
		return "", "", fmt.Errorf("bcrypt hash: %w", err)
	}
	return string(hashBytes), salt, nil
}

// VerifyPassword checks a password against a stored bcrypt hash and salt.
// It prepends the salt to the plaintext password before comparing.
func VerifyPassword(password string, hash string, salt string) bool {
	salted := salt + password
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(salted))
	return err == nil
}

// SHA256Hex returns the hex-encoded SHA-256 hash of the input.
func SHA256Hex(input string) string {
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:])
}
