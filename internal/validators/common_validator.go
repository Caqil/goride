package validators

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var validate *validator.Validate

func init() {
	validate = validator.New()

	// Register custom validation functions
	validate.RegisterValidation("object_id", validateObjectID)
	validate.RegisterValidation("phone_number", validatePhoneNumber)
	validate.RegisterValidation("strong_password", validateStrongPassword)
	validate.RegisterValidation("coordinates", validateCoordinates)
	validate.RegisterValidation("currency_code", validateCurrencyCode)
	validate.RegisterValidation("language_code", validateLanguageCode)
	validate.RegisterValidation("rating_value", validateRatingValue)
	validate.RegisterValidation("license_plate", validateLicensePlate)
	validate.RegisterValidation("vin_number", validateVIN)
	validate.RegisterValidation("future_date", validateFutureDate)
	validate.RegisterValidation("past_date", validatePastDate)
	validate.RegisterValidation("business_hours", validateBusinessHours)
	validate.RegisterValidation("fare_amount", validateFareAmount)
	validate.RegisterValidation("distance", validateDistance)
	validate.RegisterValidation("duration", validateDuration)
}

// Common validation errors
var (
	ErrInvalidObjectID     = errors.New("invalid object ID format")
	ErrInvalidPhoneNumber  = errors.New("invalid phone number format")
	ErrWeakPassword        = errors.New("password does not meet strength requirements")
	ErrInvalidCoordinates  = errors.New("invalid GPS coordinates")
	ErrInvalidCurrency     = errors.New("invalid currency code")
	ErrInvalidLanguage     = errors.New("invalid language code")
	ErrInvalidRating       = errors.New("rating must be between 1.0 and 5.0")
	ErrInvalidLicensePlate = errors.New("invalid license plate format")
	ErrInvalidVIN          = errors.New("invalid VIN number")
	ErrInvalidDate         = errors.New("invalid date")
	ErrInvalidFareAmount   = errors.New("invalid fare amount")
	ErrInvalidDistance     = errors.New("invalid distance value")
	ErrInvalidDuration     = errors.New("invalid duration value")
)

// ValidationError represents a field validation error
type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	var messages []string
	for _, err := range v {
		messages = append(messages, fmt.Sprintf("%s: %s", err.Field, err.Message))
	}
	return strings.Join(messages, "; ")
}

// ValidateStruct validates a struct and returns detailed errors
func ValidateStruct(s interface{}) ValidationErrors {
	var validationErrors ValidationErrors

	err := validate.Struct(s)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			validationError := ValidationError{
				Field:   err.Field(),
				Tag:     err.Tag(),
				Value:   fmt.Sprintf("%v", err.Value()),
				Message: getErrorMessage(err),
			}
			validationErrors = append(validationErrors, validationError)
		}
	}

	return validationErrors
}

func getErrorMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", err.Field())
	case "email":
		return "Invalid email format"
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", err.Field(), err.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", err.Field(), err.Param())
	case "object_id":
		return "Invalid ID format"
	case "phone_number":
		return "Invalid phone number format"
	case "strong_password":
		return "Password must contain uppercase, lowercase, number, and special character"
	case "coordinates":
		return "Invalid GPS coordinates"
	case "currency_code":
		return "Invalid currency code"
	case "language_code":
		return "Invalid language code"
	case "rating_value":
		return "Rating must be between 1.0 and 5.0"
	case "license_plate":
		return "Invalid license plate format"
	case "vin_number":
		return "Invalid VIN number"
	case "future_date":
		return "Date must be in the future"
	case "past_date":
		return "Date must be in the past"
	case "fare_amount":
		return "Invalid fare amount"
	case "distance":
		return "Invalid distance value"
	case "duration":
		return "Invalid duration value"
	default:
		return fmt.Sprintf("Validation failed for %s", err.Field())
	}
}

// Custom validation functions
func validateObjectID(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // Let required tag handle empty values
	}
	_, err := primitive.ObjectIDFromHex(value)
	return err == nil
}

func validatePhoneNumber(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	if phone == "" {
		return true
	}

	// E.164 format validation
	phoneRegex := regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	return phoneRegex.MatchString(phone)
}

func validateStrongPassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	if len(password) < 8 || len(password) > 128 {
		return false
	}

	var (
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasNumber && hasSpecial
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

func validateRatingValue(fl validator.FieldLevel) bool {
	rating := fl.Field().Float()
	return rating >= 1.0 && rating <= 5.0
}

func validateLicensePlate(fl validator.FieldLevel) bool {
	plate := fl.Field().String()
	if plate == "" {
		return true
	}

	// Basic license plate validation - can be customized per region
	plateRegex := regexp.MustCompile(`^[A-Z0-9\-\s]{2,10}$`)
	return plateRegex.MatchString(strings.ToUpper(plate))
}

func validateVIN(fl validator.FieldLevel) bool {
	vin := fl.Field().String()
	if vin == "" {
		return true
	}

	if len(vin) != 17 {
		return false
	}

	vinRegex := regexp.MustCompile(`^[A-HJ-NPR-Z0-9]{17}$`)
	return vinRegex.MatchString(strings.ToUpper(vin))
}

func validateFutureDate(fl validator.FieldLevel) bool {
	date := fl.Field().Interface().(time.Time)
	return date.After(time.Now())
}

func validatePastDate(fl validator.FieldLevel) bool {
	date := fl.Field().Interface().(time.Time)
	return date.Before(time.Now())
}

func validateBusinessHours(fl validator.FieldLevel) bool {
	timeStr := fl.Field().String()
	if timeStr == "" {
		return true
	}

	// Validate HH:MM format
	timeRegex := regexp.MustCompile(`^([0-1]?[0-9]|2[0-3]):[0-5][0-9]$`)
	return timeRegex.MatchString(timeStr)
}

func validateFareAmount(fl validator.FieldLevel) bool {
	amount := fl.Field().Float()
	return amount >= 0 && amount <= 10000 // Max $10,000 per ride
}

func validateDistance(fl validator.FieldLevel) bool {
	distance := fl.Field().Float()
	return distance >= 0 && distance <= 1000 // Max 1000 km
}

func validateDuration(fl validator.FieldLevel) bool {
	duration := fl.Field().Int()
	return duration >= 0 && duration <= 86400 // Max 24 hours in seconds
}

// Helper functions for common validations
func IsValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func IsValidObjectID(id string) bool {
	_, err := primitive.ObjectIDFromHex(id)
	return err == nil
}

func SanitizeInput(input string) string {
	// Remove HTML tags and trim whitespace
	htmlRegex := regexp.MustCompile(`<[^>]*>`)
	cleaned := htmlRegex.ReplaceAllString(input, "")
	return strings.TrimSpace(cleaned)
}
