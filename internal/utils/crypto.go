package utils

import "crypto/rand"

// GenerateRandomBytes generates random slice of bytes with specified size
func GenerateRandomBytes(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}
