package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

func EncryptData(data []byte, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptData(encryptedData string, key []byte) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func HashData(data string) string {
	hash := sha256.Sum256([]byte(data))
	return base64.StdEncoding.EncodeToString(hash[:])
}

func GenerateHMAC(data, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func VerifyHMAC(data, signature, key string) bool {
	expectedSignature := GenerateHMAC(data, key)
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

func GenerateEncryptionKey() []byte {
	key := make([]byte, 32) // 256-bit key
	rand.Read(key)
	return key
}

func EncryptString(plaintext, keyStr string) (string, error) {
	key := []byte(keyStr)
	if len(key) != 32 {
		// Pad or truncate key to 32 bytes
		newKey := make([]byte, 32)
		copy(newKey, key)
		key = newKey
	}
	
	return EncryptData([]byte(plaintext), key)
}

func DecryptString(ciphertext, keyStr string) (string, error) {
	key := []byte(keyStr)
	if len(key) != 32 {
		// Pad or truncate key to 32 bytes
		newKey := make([]byte, 32)
		copy(newKey, key)
		key = newKey
	}
	
	data, err := DecryptData(ciphertext, key)
	if err != nil {
		return "", err
	}
	
	return string(data), nil
}