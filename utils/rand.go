package utils

import "math/rand"

const letters = "abcdefghijklmnopqrstuvwxyz1234567890"

func RandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
