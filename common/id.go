package common

import (
	"crypto/rand"
	"fmt"
)

// RandomID generates a random hexadecimal string of the requested len
func RandomID(len int) (string, error) {
	b, err := RandomIDBytes(len/2 + 1)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b)[0:len], nil
}

// RandomIDBytes generates a random byte array requested len
func RandomIDBytes(len int) ([]byte, error) {
	b := make([]byte, len)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}
