package db

import "crypto/rand"

func randomCredential(length int) (string, error) {
	if length < 12 {
		length = 12
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%^&*"
	out := make([]byte, length)
	for i, value := range buf {
		out[i] = alphabet[int(value)%len(alphabet)]
	}
	return string(out), nil
}
