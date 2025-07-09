package validators

import (
	"fmt"
	"time"
	"unicode"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DriverRegistrationRequest struct {
	UserID            primitive.ObjectID        `json:"user_id" validate:"required,object_id"`
	LicenseNumber     string                    `json:"license_number" validate:"required,min=5,max=20"`
	LicenseExpiry     time.Time                 `json:"license_expiry" validate:"required,future_date"`
	LicenseDocument   string                    `json:"license_document" validate:"required,url"`
	InsuranceNumber   string                    `json:"insurance_number" validate:"omitempty,min=5,max=30"`
	InsuranceExpiry   time.Time                 `json:"insurance_expiry" validate:"omitempty,future_date"`
	InsuranceDocument string                    `json:"insurance_document" validate:"omitempty,url"`
	DateOfBirth       time.Time                 `json:"date_of_birth" validate:"required,past_date"`
	EmergencyContacts []EmergencyContactRequest `json:"emergency_contacts" validate:"required,min=1,max=3,dive"`
}

type DriverUpdateRequest struct {
	LicenseNumber     string                    `json:"license_number" validate:"omitempty,min=5,max=20"`
	LicenseExpiry     *time.Time                `json:"license_expiry" validate:"omitempty,future_date"`
	LicenseDocument   string                    `json:"license_document" validate:"omitempty,url"`
	InsuranceNumber   string                    `json:"insurance_number" validate:"omitempty,min=5,max=30"`
	InsuranceExpiry   *time.Time                `json:"insurance_expiry" validate:"omitempty,future_date"`
	InsuranceDocument string                    `json:"insurance_document" validate:"omitempty,url"`
	EmergencyContacts []EmergencyContactRequest `json:"emergency_contacts" validate:"omitempty,max=3,dive"`
	PreferredAreas    []string                  `json:"preferred_areas" validate:"omitempty,max=10"`
	WorkingHours      *WorkingHoursRequest      `json:"working_hours" validate:"omitempty"`
}

type EmergencyContactRequest struct {
	Name         string `json:"name" validate:"required,min=2,max=100"`
	Phone        string `json:"phone" validate:"required,phone_number"`
	Relationship string `json:"relationship" validate:"required,oneof=family friend spouse parent sibling other"`
}

type WorkingHoursRequest struct {
	Monday    []TimeSlotRequest `json:"monday" validate:"omitempty,max=3,dive"`
	Tuesday   []TimeSlotRequest `json:"tuesday" validate:"omitempty,max=3,dive"`
	Wednesday []TimeSlotRequest `json:"wednesday" validate:"omitempty,max=3,dive"`
	Thursday  []TimeSlotRequest `json:"thursday" validate:"omitempty,max=3,dive"`
	Friday    []TimeSlotRequest `json:"friday" validate:"omitempty,max=3,dive"`
	Saturday  []TimeSlotRequest `json:"saturday" validate:"omitempty,max=3,dive"`
	Sunday    []TimeSlotRequest `json:"sunday" validate:"omitempty,max=3,dive"`
}

type TimeSlotRequest struct {
	StartTime string `json:"start_time" validate:"required,business_hours"`
	EndTime   string `json:"end_time" validate:"required,business_hours"`
}

type DriverLocationUpdateRequest struct {
	Latitude  float64 `json:"latitude" validate:"required,min=-90,max=90"`
	Longitude float64 `json:"longitude" validate:"required,min=-180,max=180"`
	Bearing   float64 `json:"bearing" validate:"omitempty,min=0,max=360"`
	Speed     float64 `json:"speed" validate:"omitempty,min=0,max=200"` // km/h
	Accuracy  float64 `json:"accuracy" validate:"omitempty,min=0"`
}

type DriverStatusUpdateRequest struct {
	Status      string `json:"status" validate:"required,oneof=online offline busy break"`
	IsAvailable bool   `json:"is_available"`
}

type BankAccountRequest struct {
	AccountNumber string `json:"account_number" validate:"required,min=8,max=20"`
	RoutingNumber string `json:"routing_number" validate:"required,min=8,max=15"`
	AccountName   string `json:"account_name" validate:"required,min=2,max=100"`
	BankName      string `json:"bank_name" validate:"required,min=2,max=100"`
	AccountType   string `json:"account_type" validate:"required,oneof=checking savings"`
}

func ValidateDriverRegistration(req *DriverRegistrationRequest) ValidationErrors {
	errors := ValidateStruct(req)

	// Validate minimum age (18 years)
	age := time.Since(req.DateOfBirth).Hours() / (24 * 365.25)
	if age < 18 {
		errors = append(errors, ValidationError{
			Field:   "date_of_birth",
			Message: "Driver must be at least 18 years old",
		})
	}

	// Validate license expiry (must be at least 6 months in future)
	if req.LicenseExpiry.Before(time.Now().AddDate(0, 6, 0)) {
		errors = append(errors, ValidationError{
			Field:   "license_expiry",
			Message: "License must be valid for at least 6 months",
		})
	}

	// Validate insurance expiry if provided
	if !req.InsuranceExpiry.IsZero() && req.InsuranceExpiry.Before(time.Now().AddDate(0, 1, 0)) {
		errors = append(errors, ValidationError{
			Field:   "insurance_expiry",
			Message: "Insurance must be valid for at least 1 month",
		})
	}

	return errors
}

func ValidateDriverUpdate(req *DriverUpdateRequest) ValidationErrors {
	errors := ValidateStruct(req)

	// Validate license expiry if provided
	if req.LicenseExpiry != nil && req.LicenseExpiry.Before(time.Now().AddDate(0, 1, 0)) {
		errors = append(errors, ValidationError{
			Field:   "license_expiry",
			Message: "License must be valid for at least 1 month",
		})
	}

	// Validate insurance expiry if provided
	if req.InsuranceExpiry != nil && req.InsuranceExpiry.Before(time.Now().AddDate(0, 1, 0)) {
		errors = append(errors, ValidationError{
			Field:   "insurance_expiry",
			Message: "Insurance must be valid for at least 1 month",
		})
	}

	// Validate working hours
	if req.WorkingHours != nil {
		errors = append(errors, validateWorkingHours(req.WorkingHours)...)
	}

	return errors
}

func ValidateDriverLocationUpdate(req *DriverLocationUpdateRequest) ValidationErrors {
	return ValidateStruct(req)
}

func ValidateDriverStatusUpdate(req *DriverStatusUpdateRequest) ValidationErrors {
	errors := ValidateStruct(req)

	// Business logic: driver cannot be available if offline
	if req.Status == "offline" && req.IsAvailable {
		errors = append(errors, ValidationError{
			Field:   "is_available",
			Message: "Driver cannot be available when offline",
		})
	}

	return errors
}

func ValidateBankAccount(req *BankAccountRequest) ValidationErrors {
	errors := ValidateStruct(req)

	// Additional validation for bank account numbers (simplified)
	if !isValidBankAccount(req.AccountNumber) {
		errors = append(errors, ValidationError{
			Field:   "account_number",
			Message: "Invalid bank account number format",
		})
	}

	return errors
}

func validateWorkingHours(hours *WorkingHoursRequest) ValidationErrors {
	var errors ValidationErrors

	days := map[string][]TimeSlotRequest{
		"monday":    hours.Monday,
		"tuesday":   hours.Tuesday,
		"wednesday": hours.Wednesday,
		"thursday":  hours.Thursday,
		"friday":    hours.Friday,
		"saturday":  hours.Saturday,
		"sunday":    hours.Sunday,
	}

	for day, slots := range days {
		for i, slot := range slots {
			// Validate start time is before end time
			startTime, err1 := time.Parse("15:04", slot.StartTime)
			endTime, err2 := time.Parse("15:04", slot.EndTime)

			if err1 != nil || err2 != nil {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("%s[%d]", day, i),
					Message: "Invalid time format (use HH:MM)",
				})
				continue
			}

			if !endTime.After(startTime) {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("%s[%d]", day, i),
					Message: "End time must be after start time",
				})
			}
		}
	}

	return errors
}

func isValidBankAccount(accountNumber string) bool {
	// Simplified bank account validation
	if len(accountNumber) < 8 || len(accountNumber) > 20 {
		return false
	}

	// Check if all characters are digits
	for _, char := range accountNumber {
		if !unicode.IsDigit(char) {
			return false
		}
	}

	return true
}
