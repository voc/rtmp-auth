package store

import (
	"crypto/rand"
	"fmt"
)

func makeSecret() ([]byte, error) {
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	if err != nil {
		return nil, fmt.Errorf("generate secret: %w", err)
	}
	return secret, nil
}
