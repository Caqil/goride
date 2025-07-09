package validators

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RiderRegistrationRequest struct {
	UserID primitive.ObjectID `json:"user_id" validate:"required,object_id"`
}

type RiderUpdateRequest struct {
	FavoriteLocations  []FavoriteLocationRequest `json:"favorite_locations" validate:"omitempty,max=10,dive"`
	EmergencyContacts  []EmergencyContactRequest `json:"emergency_contacts" validate:"omitempty,max=5,dive"`
	AccessibilityNeeds []string                  `json:"accessibility_needs" validate:"omitempty,max=5"`
	RidePreferences    *RidePreferencesRequest   `json:"ride_preferences" validate:"omitempty"`
}

type FavoriteLocationRequest struct {
	Name      string  `json:"name" validate:"required,min=1,max=100"`
	Address   string  `json:"address" validate:"required,min=5,max=255"`
	Latitude  float64 `json:"latitude" validate:"required,min=-90,max=90"`
	Longitude float64 `json:"longitude" validate:"required,min=-180,max=180"`
	Type      string  `json:"type" validate:"required,oneof=home work other"`
}

type RidePreferencesRequest struct {
	PreferredRideTypes  []string `json:"preferred_ride_types" validate:"omitempty,max=5"`
	Temperature         int      `json:"temperature" validate:"omitempty,min=16,max=30"`
	MusicPreference     string   `json:"music_preference" validate:"omitempty,oneof=none pop rock jazz classical country hip-hop"`
	ConversationLevel   string   `json:"conversation_level" validate:"omitempty,oneof=none minimal normal chatty"`
	AllowPetFriendly    bool     `json:"allow_pet_friendly"`
	AllowSharedRides    bool     `json:"allow_shared_rides"`
	PreferFemaleDrivers bool     `json:"prefer_female_drivers"`
}

type PaymentMethodRequest struct {
	Type           string                 `json:"type" validate:"required,oneof=credit_card debit_card paypal apple_pay google_pay"`
	Token          string                 `json:"token" validate:"required"`
	LastFourDigits string                 `json:"last_four_digits" validate:"omitempty,len=4,numeric"`
	ExpiryMonth    int                    `json:"expiry_month" validate:"omitempty,min=1,max=12"`
	ExpiryYear     int                    `json:"expiry_year" validate:"omitempty,min=2024"`
	BillingAddress *BillingAddressRequest `json:"billing_address" validate:"omitempty"`
	IsDefault      bool                   `json:"is_default"`
}

type BillingAddressRequest struct {
	Street     string `json:"street" validate:"required,min=5,max=255"`
	City       string `json:"city" validate:"required,min=2,max=100"`
	State      string `json:"state" validate:"required,min=2,max=100"`
	PostalCode string `json:"postal_code" validate:"required,min=3,max=20"`
	Country    string `json:"country" validate:"required,len=2"`
}

func ValidateRiderRegistration(req *RiderRegistrationRequest) ValidationErrors {
	return ValidateStruct(req)
}

func ValidateRiderUpdate(req *RiderUpdateRequest) ValidationErrors {
	errors := ValidateStruct(req)

	// Validate favorite locations don't have duplicates
	if len(req.FavoriteLocations) > 1 {
		locationTypes := make(map[string]bool)
		for i, location := range req.FavoriteLocations {
			if location.Type == "home" || location.Type == "work" {
				if locationTypes[location.Type] {
					errors = append(errors, ValidationError{
						Field:   fmt.Sprintf("favorite_locations[%d].type", i),
						Message: fmt.Sprintf("Only one %s location is allowed", location.Type),
					})
				}
				locationTypes[location.Type] = true
			}
		}
	}

	// Validate accessibility needs
	validNeeds := []string{
		"wheelchair_accessible",
		"hearing_impaired",
		"visually_impaired",
		"service_animal",
		"mobility_aid",
	}

	for i, need := range req.AccessibilityNeeds {
		if !contains(validNeeds, need) {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("accessibility_needs[%d]", i),
				Message: "Invalid accessibility need",
			})
		}
	}

	return errors
}

func ValidatePaymentMethod(req *PaymentMethodRequest) ValidationErrors {
	errors := ValidateStruct(req)

	// Validate expiry date for cards
	if req.Type == "credit_card" || req.Type == "debit_card" {
		if req.ExpiryMonth == 0 || req.ExpiryYear == 0 {
			errors = append(errors, ValidationError{
				Field:   "expiry",
				Message: "Expiry month and year are required for cards",
			})
		} else {
			// Check if card is not expired
			currentYear := time.Now().Year()
			currentMonth := int(time.Now().Month())

			if req.ExpiryYear < currentYear ||
				(req.ExpiryYear == currentYear && req.ExpiryMonth < currentMonth) {
				errors = append(errors, ValidationError{
					Field:   "expiry",
					Message: "Card is expired",
				})
			}
		}

		if req.LastFourDigits == "" {
			errors = append(errors, ValidationError{
				Field:   "last_four_digits",
				Message: "Last four digits are required for cards",
			})
		}
	}

	return errors
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
