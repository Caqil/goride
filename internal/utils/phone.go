package utils

import (
	"regexp"
	"strings"
)

var phoneRegex = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)

func IsValidPhone(phone string) bool {
	// Remove all non-digit characters except +
	cleaned := regexp.MustCompile(`[^\d+]`).ReplaceAllString(phone, "")
	
	// Basic E.164 format validation
	return phoneRegex.MatchString(cleaned)
}

func FormatPhone(phone, countryCode string) string {
	// Remove all non-digit characters
	cleaned := regexp.MustCompile(`[^\d]`).ReplaceAllString(phone, "")
	
	if !strings.HasPrefix(cleaned, strings.TrimPrefix(countryCode, "+")) {
		cleaned = strings.TrimPrefix(countryCode, "+") + cleaned
	}
	
	return "+" + cleaned
}

func NormalizePhone(phone string) string {
	// Remove all spaces, dashes, parentheses, etc.
	normalized := regexp.MustCompile(`[^\d+]`).ReplaceAllString(phone, "")
	
	// Ensure it starts with +
	if !strings.HasPrefix(normalized, "+") {
		normalized = "+" + normalized
	}
	
	return normalized
}

func ExtractCountryCode(phone string) string {
	normalized := NormalizePhone(phone)
	
	// Common country codes
	countryCodes := []string{
		"+1",   // US/Canada
		"+44",  // UK
		"+33",  // France
		"+49",  // Germany
		"+39",  // Italy
		"+34",  // Spain
		"+81",  // Japan
		"+86",  // China
		"+91",  // India
		"+55",  // Brazil
		"+7",   // Russia
		"+61",  // Australia
		"+52",  // Mexico
	}
	
	for _, code := range countryCodes {
		if strings.HasPrefix(normalized, code) {
			return code
		}
	}
	
	// If no match, assume single digit country code
	if len(normalized) > 1 {
		return "+" + string(normalized[1])
	}
	
	return DefaultCountryCode
}

func MaskPhone(phone string) string {
	if len(phone) < 4 {
		return phone
	}
	
	// Show last 4 digits
	masked := strings.Repeat("*", len(phone)-4) + phone[len(phone)-4:]
	return masked
}

func IsValidPhoneForCountry(phone, countryCode string) bool {
	formatted := FormatPhone(phone, countryCode)
	return IsValidPhone(formatted)
}

func GenerateOTP() string {
	return GenerateRandomNumericString(OTPLength)
}

func ValidateOTP(otp string) bool {
	if len(otp) != OTPLength {
		return false
	}
	
	// Check if all characters are digits
	for _, char := range otp {
		if char < '0' || char > '9' {
			return false
		}
	}
	
	return true
}
