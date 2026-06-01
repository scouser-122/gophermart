package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
)

// GenerateRandomBytes generates random slice of bytes with specified size
func GenerateRandomBytes(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// CountSha256Sum counts SHA-256 hash for specified string
func CountSha256Sum(value string) (string, error) {
	hash := sha256.New()
	_, err := hash.Write([]byte(value))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
