package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

const (
	letterBytes  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numberBytes  = "0123456789"
	alphanumeric = letterBytes + numberBytes
	specialChars = "!@#$%^&*()_+-=[]{}|;:,.<>?"
	allChars     = alphanumeric + specialChars
)

func GenerateRandomString(length int) string {
	return generateRandom(length, alphanumeric)
}

func GenerateRandomNumericString(length int) string {
	return generateRandom(length, numberBytes)
}

func GenerateRandomAlphaString(length int) string {
	return generateRandom(length, letterBytes)
}

func GenerateRandomPassword(length int) string {
	return generateRandom(length, allChars)
}

func generateRandom(length int, charset string) string {
	result := make([]byte, length)
	charsetLength := big.NewInt(int64(len(charset)))

	for i := range result {
		num, _ := rand.Int(rand.Reader, charsetLength)
		result[i] = charset[num.Int64()]
	}

	return string(result)
}

func SecureRandomInt(max int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return int(n.Int64())
}

func SecureRandomFloat() float64 {
	max := big.NewInt(1 << 53)
	n, _ := rand.Int(rand.Reader, max)
	return float64(n.Int64()) / float64(1<<53)
}

func GenerateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)

	// Set version (4) and variant bits
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return strings.ToLower(fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]))
}

func GenerateReferralCode() string {
	// Generate 8 character alphanumeric code
	code := strings.ToUpper(GenerateRandomString(ReferralCodeLength))

	// Ensure no confusing characters (0, O, I, L)
	code = strings.ReplaceAll(code, "0", "2")
	code = strings.ReplaceAll(code, "O", "3")
	code = strings.ReplaceAll(code, "I", "4")
	code = strings.ReplaceAll(code, "L", "5")

	return code
}

func GenerateShareToken() string {
	return GenerateRandomString(32)
}

func GenerateSessionID() string {
	return GenerateRandomString(64)
}

func GenerateAPIKey() string {
	prefix := "uber_"
	key := GenerateRandomString(32)
	return prefix + key
}

func ShuffleSlice(slice []interface{}) {
	for i := len(slice) - 1; i > 0; i-- {
		j := SecureRandomInt(i + 1)
		slice[i], slice[j] = slice[j], slice[i]
	}
}
