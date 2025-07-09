package validators

import (
	"regexp"
	"strings"
	"time"
)

type UserRegistrationRequest struct {
	FirstName   string `json:"first_name" validate:"required,min=2,max=50"`
	LastName    string `json:"last_name" validate:"required,min=2,max=50"`
	Email       string `json:"email" validate:"required,email"`
	Phone       string `json:"phone" validate:"required,phone_number"`
	CountryCode string `json:"country_code" validate:"required,min=2,max=5"`
	Password    string `json:"password" validate:"required,strong_password"`
	UserType    string `json:"user_type" validate:"required,oneof=rider driver"`
	Language    string `json:"language" validate:"omitempty,language_code"`
}

type UserLoginRequest struct {
	Email    string `json:"email" validate:"required_without=Phone,omitempty,email"`
	Phone    string `json:"phone" validate:"required_without=Email,omitempty,phone_number"`
	Password string `json:"password" validate:"required,min=8"`
}

type UserUpdateRequest struct {
	FirstName      string     `json:"first_name" validate:"omitempty,min=2,max=50"`
	LastName       string     `json:"last_name" validate:"omitempty,min=2,max=50"`
	Email          string     `json:"email" validate:"omitempty,email"`
	Phone          string     `json:"phone" validate:"omitempty,phone_number"`
	DateOfBirth    *time.Time `json:"date_of_birth" validate:"omitempty,past_date"`
	Gender         string     `json:"gender" validate:"omitempty,oneof=male female other"`
	Language       string     `json:"language" validate:"omitempty,language_code"`
	ProfilePicture string     `json:"profile_picture" validate:"omitempty,url"`
}

type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,strong_password"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,strong_password"`
}

type VerifyOTPRequest struct {
	Phone string `json:"phone" validate:"required,phone_number"`
	OTP   string `json:"otp" validate:"required,len=6,numeric"`
}

func ValidateUserRegistration(req *UserRegistrationRequest) ValidationErrors {
	errors := ValidateStruct(req)

	// Additional business logic validation
	if req.Email != "" {
		req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	}

	if req.Phone != "" {
		req.Phone = normalizePhoneNumber(req.Phone, req.CountryCode)
	}

	// Validate age for drivers (must be 18+)
	if req.UserType == "driver" {
		// This would need DateOfBirth in registration for proper validation
		// For now, we'll validate it during driver profile completion
	}

	return errors
}

func ValidateUserLogin(req *UserLoginRequest) ValidationErrors {
	errors := ValidateStruct(req)

	// Ensure either email or phone is provided
	if req.Email == "" && req.Phone == "" {
		errors = append(errors, ValidationError{
			Field:   "email_or_phone",
			Message: "Either email or phone number is required",
		})
	}

	if req.Email != "" {
		req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	}

	return errors
}

func ValidateUserUpdate(req *UserUpdateRequest) ValidationErrors {
	errors := ValidateStruct(req)

	// Validate date of birth if provided
	if req.DateOfBirth != nil {
		age := time.Since(*req.DateOfBirth).Hours() / (24 * 365.25)
		if age < 13 {
			errors = append(errors, ValidationError{
				Field:   "date_of_birth",
				Message: "User must be at least 13 years old",
			})
		}
		if age > 120 {
			errors = append(errors, ValidationError{
				Field:   "date_of_birth",
				Message: "Invalid date of birth",
			})
		}
	}

	return errors
}

func ValidatePasswordChange(req *PasswordChangeRequest) ValidationErrors {
	errors := ValidateStruct(req)

	// Ensure new password is different from current
	if req.CurrentPassword == req.NewPassword {
		errors = append(errors, ValidationError{
			Field:   "new_password",
			Message: "New password must be different from current password",
		})
	}

	return errors
}

func normalizePhoneNumber(phone, countryCode string) string {
	// Remove all non-digit characters except +
	cleaned := regexp.MustCompile(`[^\d+]`).ReplaceAllString(phone, "")

	if !strings.HasPrefix(cleaned, "+") {
		if !strings.HasPrefix(cleaned, strings.TrimPrefix(countryCode, "+")) {
			cleaned = countryCode + cleaned
		} else {
			cleaned = "+" + cleaned
		}
	}

	return cleaned
}
