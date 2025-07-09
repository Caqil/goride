package validators

import (
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type VehicleCreateRequest struct {
	DriverID             primitive.ObjectID `json:"driver_id" validate:"required,object_id"`
	VehicleTypeID        primitive.ObjectID `json:"vehicle_type_id" validate:"required,object_id"`
	Make                 string             `json:"make" validate:"required,min=2,max=50"`
	Model                string             `json:"model" validate:"required,min=1,max=50"`
	Year                 int                `json:"year" validate:"required,min=1990,max=2025"`
	Color                string             `json:"color" validate:"required,min=3,max=30"`
	LicensePlate         string             `json:"license_plate" validate:"required,license_plate"`
	VIN                  string             `json:"vin" validate:"omitempty,vin_number"`
	RegistrationNumber   string             `json:"registration_number" validate:"required,min=5,max=20"`
	RegistrationExpiry   time.Time          `json:"registration_expiry" validate:"required,future_date"`
	RegistrationDocument string             `json:"registration_document" validate:"required,url"`
	InsuranceNumber      string             `json:"insurance_number" validate:"required,min=5,max=30"`
	InsuranceExpiry      time.Time          `json:"insurance_expiry" validate:"required,future_date"`
	InsuranceDocument    string             `json:"insurance_document" validate:"required,url"`
	Capacity             int                `json:"capacity" validate:"required,min=1,max=8"`
	Photos               []string           `json:"photos" validate:"required,min=3,max=10,dive,url"`
	Features             []string           `json:"features" validate:"omitempty,max=15"`
	IsAccessible         bool               `json:"is_accessible"`
}

type VehicleUpdateRequest struct {
	Make                 string     `json:"make" validate:"omitempty,min=2,max=50"`
	Model                string     `json:"model" validate:"omitempty,min=1,max=50"`
	Year                 int        `json:"year" validate:"omitempty,min=1990,max=2025"`
	Color                string     `json:"color" validate:"omitempty,min=3,max=30"`
	LicensePlate         string     `json:"license_plate" validate:"omitempty,license_plate"`
	VIN                  string     `json:"vin" validate:"omitempty,vin_number"`
	RegistrationNumber   string     `json:"registration_number" validate:"omitempty,min=5,max=20"`
	RegistrationExpiry   *time.Time `json:"registration_expiry" validate:"omitempty,future_date"`
	RegistrationDocument string     `json:"registration_document" validate:"omitempty,url"`
	InsuranceNumber      string     `json:"insurance_number" validate:"omitempty,min=5,max=30"`
	InsuranceExpiry      *time.Time `json:"insurance_expiry" validate:"omitempty,future_date"`
	InsuranceDocument    string     `json:"insurance_document" validate:"omitempty,url"`
	Capacity             int        `json:"capacity" validate:"omitempty,min=1,max=8"`
	Photos               []string   `json:"photos" validate:"omitempty,min=1,max=10,dive,url"`
	Features             []string   `json:"features" validate:"omitempty,max=15"`
	IsAccessible         bool       `json:"is_accessible"`
	Status               string     `json:"status" validate:"omitempty,oneof=active inactive maintenance"`
}

type VehicleInspectionRequest struct {
	VehicleID      primitive.ObjectID `json:"vehicle_id" validate:"required,object_id"`
	InspectorID    primitive.ObjectID `json:"inspector_id" validate:"required,object_id"`
	InspectionType string             `json:"inspection_type" validate:"required,oneof=annual safety emissions maintenance"`
	InspectionDate time.Time          `json:"inspection_date" validate:"required,past_date"`
	ExpiryDate     time.Time          `json:"expiry_date" validate:"required,future_date"`
	Status         string             `json:"status" validate:"required,oneof=passed failed pending"`
	Certificate    string             `json:"certificate" validate:"omitempty,url"`
	Notes          string             `json:"notes" validate:"omitempty,max=1000"`
	Defects        []string           `json:"defects" validate:"omitempty,max=20"`
}

type VehicleMaintenanceRequest struct {
	VehicleID       primitive.ObjectID `json:"vehicle_id" validate:"required,object_id"`
	MaintenanceType string             `json:"maintenance_type" validate:"required,oneof=oil_change tire_rotation brake_service general_service major_repair"`
	ScheduledDate   time.Time          `json:"scheduled_date" validate:"required"`
	Description     string             `json:"description" validate:"required,min=10,max=500"`
	EstimatedCost   float64            `json:"estimated_cost" validate:"omitempty,min=0,max=10000"`
	Mileage         int                `json:"mileage" validate:"omitempty,min=0"`
}

func ValidateVehicleCreate(req *VehicleCreateRequest) ValidationErrors {
	errors := ValidateStruct(req)

	// Validate vehicle year
	currentYear := time.Now().Year()
	if req.Year > currentYear+1 {
		errors = append(errors, ValidationError{
			Field:   "year",
			Message: "Vehicle year cannot be more than 1 year in the future",
		})
	}

	// Validate vehicle age for ride-sharing (typically max 10-15 years)
	vehicleAge := currentYear - req.Year
	if vehicleAge > 15 {
		errors = append(errors, ValidationError{
			Field:   "year",
			Message: "Vehicle is too old for ride-sharing service (max 15 years)",
		})
	}

	// Validate registration expiry (must be at least 30 days in future)
	if req.RegistrationExpiry.Before(time.Now().AddDate(0, 0, 30)) {
		errors = append(errors, ValidationError{
			Field:   "registration_expiry",
			Message: "Registration must be valid for at least 30 days",
		})
	}

	// Validate insurance expiry (must be at least 30 days in future)
	if req.InsuranceExpiry.Before(time.Now().AddDate(0, 0, 30)) {
		errors = append(errors, ValidationError{
			Field:   "insurance_expiry",
			Message: "Insurance must be valid for at least 30 days",
		})
	}

	// Validate vehicle features
	validFeatures := []string{
		"air_conditioning",
		"bluetooth",
		"usb_charging",
		"wifi",
		"child_seat",
		"pet_friendly",
		"wheelchair_accessible",
		"luggage_space",
		"premium_audio",
		"leather_seats",
		"sunroof",
		"gps_navigation",
		"dash_cam",
		"phone_mount",
		"water_bottles",
	}

	for i, feature := range req.Features {
		if !contains(validFeatures, feature) {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("features[%d]", i),
				Message: fmt.Sprintf("Invalid vehicle feature: %s", feature),
			})
		}
	}

	// Validate VIN checksum if provided
	if req.VIN != "" && !isValidVINChecksum(req.VIN) {
		errors = append(errors, ValidationError{
			Field:   "vin",
			Message: "Invalid VIN checksum",
		})
	}

	// Validate color
	validColors := []string{
		"white", "black", "silver", "gray", "grey", "red", "blue", "green",
		"brown", "yellow", "orange", "purple", "pink", "gold", "beige", "tan",
	}

	colorFound := false
	for _, validColor := range validColors {
		if strings.Contains(strings.ToLower(req.Color), validColor) {
			colorFound = true
			break
		}
	}

	if !colorFound {
		errors = append(errors, ValidationError{
			Field:   "color",
			Message: "Please specify a valid vehicle color",
		})
	}

	return errors
}

func ValidateVehicleUpdate(req *VehicleUpdateRequest) ValidationErrors {
	errors := ValidateStruct(req)

	// Validate year if provided
	if req.Year != 0 {
		currentYear := time.Now().Year()
		if req.Year > currentYear+1 {
			errors = append(errors, ValidationError{
				Field:   "year",
				Message: "Vehicle year cannot be more than 1 year in the future",
			})
		}

		vehicleAge := currentYear - req.Year
		if vehicleAge > 15 {
			errors = append(errors, ValidationError{
				Field:   "year",
				Message: "Vehicle is too old for ride-sharing service (max 15 years)",
			})
		}
	}

	// Validate expiry dates if provided
	if req.RegistrationExpiry != nil && req.RegistrationExpiry.Before(time.Now().AddDate(0, 0, 7)) {
		errors = append(errors, ValidationError{
			Field:   "registration_expiry",
			Message: "Registration must be valid for at least 7 days",
		})
	}

	if req.InsuranceExpiry != nil && req.InsuranceExpiry.Before(time.Now().AddDate(0, 0, 7)) {
		errors = append(errors, ValidationError{
			Field:   "insurance_expiry",
			Message: "Insurance must be valid for at least 7 days",
		})
	}

	return errors
}

func ValidateVehicleInspection(req *VehicleInspectionRequest) ValidationErrors {
	errors := ValidateStruct(req)

	// Validate inspection date is not too far in the past (max 30 days)
	if req.InspectionDate.Before(time.Now().AddDate(0, 0, -30)) {
		errors = append(errors, ValidationError{
			Field:   "inspection_date",
			Message: "Inspection date cannot be more than 30 days in the past",
		})
	}

	// Validate expiry date is reasonable based on inspection type
	minValidityDays := map[string]int{
		"annual":      365,
		"safety":      180,
		"emissions":   365,
		"maintenance": 90,
	}

	if days, exists := minValidityDays[req.InspectionType]; exists {
		if req.ExpiryDate.Before(req.InspectionDate.AddDate(0, 0, days)) {
			errors = append(errors, ValidationError{
				Field:   "expiry_date",
				Message: fmt.Sprintf("%s inspection must be valid for at least %d days", req.InspectionType, days),
			})
		}
	}

	// Validate defects for failed inspections
	if req.Status == "failed" && len(req.Defects) == 0 {
		errors = append(errors, ValidationError{
			Field:   "defects",
			Message: "Defects must be specified for failed inspections",
		})
	}

	return errors
}

func ValidateVehicleMaintenance(req *VehicleMaintenanceRequest) ValidationErrors {
	errors := ValidateStruct(req)

	// Validate scheduled date (cannot be more than 90 days in future)
	if req.ScheduledDate.After(time.Now().AddDate(0, 0, 90)) {
		errors = append(errors, ValidationError{
			Field:   "scheduled_date",
			Message: "Maintenance cannot be scheduled more than 90 days in advance",
		})
	}

	// Validate cost reasonableness based on maintenance type
	maxCosts := map[string]float64{
		"oil_change":      200,
		"tire_rotation":   100,
		"brake_service":   800,
		"general_service": 500,
		"major_repair":    5000,
	}

	if maxCost, exists := maxCosts[req.MaintenanceType]; exists && req.EstimatedCost > maxCost {
		errors = append(errors, ValidationError{
			Field:   "estimated_cost",
			Message: fmt.Sprintf("Estimated cost seems too high for %s", req.MaintenanceType),
		})
	}

	return errors
}

// Helper functions
func isValidVINChecksum(vin string) bool {
	// Simplified VIN checksum validation
	if len(vin) != 17 {
		return false
	}

	// VIN validation is complex, this is a basic implementation
	// In production, use a proper VIN validation library
	weights := []int{8, 7, 6, 5, 4, 3, 2, 10, 0, 9, 8, 7, 6, 5, 4, 3, 2}
	values := map[rune]int{
		'A': 1, 'B': 2, 'C': 3, 'D': 4, 'E': 5, 'F': 6, 'G': 7, 'H': 8,
		'J': 1, 'K': 2, 'L': 3, 'M': 4, 'N': 5, 'P': 7, 'R': 9,
		'S': 2, 'T': 3, 'U': 4, 'V': 5, 'W': 6, 'X': 7, 'Y': 8, 'Z': 9,
		'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7, '8': 8, '9': 9,
	}

	sum := 0
	for i, char := range vin {
		if i == 8 { // Skip check digit position
			continue
		}
		if val, exists := values[char]; exists {
			sum += val * weights[i]
		}
	}

	remainder := sum % 11
	checkDigit := vin[8]

	if remainder == 10 {
		return checkDigit == 'X'
	}

	return rune(checkDigit) == rune('0'+remainder)
}
