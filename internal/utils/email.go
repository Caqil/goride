package utils

import (
	"fmt"
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func NormalizeEmail(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))

	// Handle Gmail aliases (ignore dots and + aliases)
	if strings.Contains(email, "@gmail.com") {
		parts := strings.Split(email, "@")
		if len(parts) == 2 {
			localPart := parts[0]

			// Remove dots
			localPart = strings.ReplaceAll(localPart, ".", "")

			// Remove + aliases
			if plusIndex := strings.Index(localPart, "+"); plusIndex != -1 {
				localPart = localPart[:plusIndex]
			}

			email = localPart + "@" + parts[1]
		}
	}

	return email
}

func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}

	localPart := parts[0]
	domain := parts[1]

	if len(localPart) <= 2 {
		return email
	}

	maskedLocal := string(localPart[0]) + strings.Repeat("*", len(localPart)-2) + string(localPart[len(localPart)-1])

	return maskedLocal + "@" + domain
}

func GetEmailDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

func IsDisposableEmail(email string) bool {
	domain := GetEmailDomain(email)

	// List of common disposable email domains
	disposableDomains := []string{
		"10minutemail.com",
		"guerrillamail.com",
		"mailinator.com",
		"tempmail.org",
		"throwaway.email",
		"temp-mail.org",
		"yopmail.com",
	}

	for _, disposable := range disposableDomains {
		if domain == disposable {
			return true
		}
	}

	return false
}

func GenerateEmailVerificationToken() string {
	return GenerateRandomString(32)
}

func GeneratePasswordResetToken() string {
	return GenerateRandomString(32)
}

func CreateEmailVerificationLink(baseURL, token string) string {
	return fmt.Sprintf("%s/verify-email?token=%s", baseURL, token)
}

func CreatePasswordResetLink(baseURL, token string) string {
	return fmt.Sprintf("%s/reset-password?token=%s", baseURL, token)
}
