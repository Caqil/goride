package utils

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
	
	// Register custom validators
	validate.RegisterValidation("phone", validatePhone)
	validate.RegisterValidation("strong_password", validateStrongPassword)
	validate.RegisterValidation("coordinates", validateCoordinates)
	validate.RegisterValidation("currency_code", validateCurrencyCode)
	validate.RegisterValidation("language_code", validateLanguageCode)
	validate.RegisterValidation("rating", validateRating)
}

func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}

func validatePhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	return IsValidPhone(phone)
}

func validateStrongPassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	return ValidatePasswordStrength(password) == nil
}

func validateCoordinates(fl validator.FieldLevel) bool {
	coords := fl.Field().Interface().([]float64)
	if len(coords) != 2 {
		return false
	}
	
	lng, lat := coords[0], coords[1]
	return lng >= -180 && lng <= 180 && lat >= -90 && lat <= 90
}

func validateCurrencyCode(fl validator.FieldLevel) bool {
	code := fl.Field().String()
	validCurrencies := []string{"USD", "EUR", "GBP", "CAD", "AUD", "JPY", "CNY", "INR", "BRL", "MXN"}
	
	for _, currency := range validCurrencies {
		if code == currency {
			return true
		}
	}
	return false
}

func validateLanguageCode(fl validator.FieldLevel) bool {
	code := fl.Field().String()
	validLanguages := []string{"en", "es", "fr", "de", "it", "pt", "ru", "zh", "ja", "ko", "ar", "hi"}
	
	for _, lang := range validLanguages {
		if code == lang {
			return true
		}
	}
	return false
}

func validateRating(fl validator.FieldLevel) bool {
	rating := fl.Field().Float()
	return rating >= MinDriverRating && rating <= MaxDriverRating
}

func IsValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func IsValidName(name string) bool {
	if len(strings.TrimSpace(name)) < 2 {
		return false
	}
	
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsSpace(r) && r != '-' && r != '\'' {
			return false
		}
	}
	return true
}

func IsValidURL(url string) bool {
	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	return urlRegex.MatchString(url)
}

func SanitizeString(input string) string {
	// Remove HTML tags
	htmlRegex := regexp.MustCompile(`<[^>]*>`)
	cleaned := htmlRegex.ReplaceAllString(input, "")
	
	// Trim whitespace
	cleaned = strings.TrimSpace(cleaned)
	
	return cleaned
}

func IsValidLicensePlate(plate string) bool {
	// Basic validation - can be extended based on region
	plateRegex := regexp.MustCompile(`^[A-Z0-9\-\s]{2,10}$`)
	return plateRegex.MatchString(strings.ToUpper(plate))
}

func IsValidVIN(vin string) bool {
	if len(vin) != 17 {
		return false
	}
	
	vinRegex := regexp.MustCompile(`^[A-HJ-NPR-Z0-9]{17}$`)
	return vinRegex.MatchString(strings.ToUpper(vin))
}
