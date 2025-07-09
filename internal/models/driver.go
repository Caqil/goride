package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type DriverStatus string
type DocumentStatus string

const (
	DriverStatusOnline    DriverStatus = "online"
	DriverStatusOffline   DriverStatus = "offline"
	DriverStatusBusy      DriverStatus = "busy"
	DriverStatusBreak     DriverStatus = "break"
	DriverStatusSuspended DriverStatus = "suspended"

	DocumentStatusPending  DocumentStatus = "pending"
	DocumentStatusApproved DocumentStatus = "approved"
	DocumentStatusRejected DocumentStatus = "rejected"
	DocumentStatusExpired  DocumentStatus = "expired"
)

type Driver struct {
	ID                    primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	UserID                primitive.ObjectID   `json:"user_id" bson:"user_id" validate:"required"`
	LicenseNumber         string               `json:"license_number" bson:"license_number" validate:"required"`
	LicenseExpiry         time.Time            `json:"license_expiry" bson:"license_expiry" validate:"required"`
	LicenseDocument       string               `json:"license_document" bson:"license_document"`
	LicenseStatus         DocumentStatus       `json:"license_status" bson:"license_status" default:"pending"`
	InsuranceNumber       string               `json:"insurance_number" bson:"insurance_number"`
	InsuranceExpiry       time.Time            `json:"insurance_expiry" bson:"insurance_expiry"`
	InsuranceDocument     string               `json:"insurance_document" bson:"insurance_document"`
	InsuranceStatus       DocumentStatus       `json:"insurance_status" bson:"insurance_status" default:"pending"`
	BackgroundCheckStatus DocumentStatus       `json:"background_check_status" bson:"background_check_status" default:"pending"`
	BackgroundCheckDate   *time.Time           `json:"background_check_date" bson:"background_check_date"`
	Status                DriverStatus         `json:"status" bson:"status" default:"offline"`
	Rating                float64              `json:"rating" bson:"rating" default:"0"`
	TotalRatings          int64                `json:"total_ratings" bson:"total_ratings" default:"0"`
	TotalRides            int64                `json:"total_rides" bson:"total_rides" default:"0"`
	TotalEarnings         float64              `json:"total_earnings" bson:"total_earnings" default:"0"`
	CurrentLocation       *Location            `json:"current_location" bson:"current_location"`
	LastLocationUpdate    *time.Time           `json:"last_location_update" bson:"last_location_update"`
	BankAccount           *BankAccount         `json:"bank_account" bson:"bank_account"`
	TaxInformation        *TaxInformation      `json:"tax_information" bson:"tax_information"`
	OnlineHours           float64              `json:"online_hours" bson:"online_hours" default:"0"`
	AcceptanceRate        float64              `json:"acceptance_rate" bson:"acceptance_rate" default:"0"`
	CancellationRate      float64              `json:"cancellation_rate" bson:"cancellation_rate" default:"0"`
	CompletionRate        float64              `json:"completion_rate" bson:"completion_rate" default:"0"`
	IsAvailable           bool                 `json:"is_available" bson:"is_available" default:"false"`
	VehicleIDs            []primitive.ObjectID `json:"vehicle_ids" bson:"vehicle_ids"`
	PreferredAreas        []string             `json:"preferred_areas" bson:"preferred_areas"`
	WorkingHours          *WorkingHours        `json:"working_hours" bson:"working_hours"`
	EmergencyContacts     []EmergencyContact   `json:"emergency_contacts" bson:"emergency_contacts"`
	CreatedAt             time.Time            `json:"created_at" bson:"created_at"`
	UpdatedAt             time.Time            `json:"updated_at" bson:"updated_at"`
	ApprovedAt            *time.Time           `json:"approved_at" bson:"approved_at"`
}

type BankAccount struct {
	AccountNumber string `json:"account_number" bson:"account_number"`
	RoutingNumber string `json:"routing_number" bson:"routing_number"`
	AccountName   string `json:"account_name" bson:"account_name"`
	BankName      string `json:"bank_name" bson:"bank_name"`
	AccountType   string `json:"account_type" bson:"account_type"`
	IsVerified    bool   `json:"is_verified" bson:"is_verified" default:"false"`
}

type TaxInformation struct {
	TaxID       string `json:"tax_id" bson:"tax_id"`
	TaxIDType   string `json:"tax_id_type" bson:"tax_id_type"`
	TaxDocument string `json:"tax_document" bson:"tax_document"`
	IsVerified  bool   `json:"is_verified" bson:"is_verified" default:"false"`
}

type WorkingHours struct {
	Monday    []TimeSlot `json:"monday" bson:"monday"`
	Tuesday   []TimeSlot `json:"tuesday" bson:"tuesday"`
	Wednesday []TimeSlot `json:"wednesday" bson:"wednesday"`
	Thursday  []TimeSlot `json:"thursday" bson:"thursday"`
	Friday    []TimeSlot `json:"friday" bson:"friday"`
	Saturday  []TimeSlot `json:"saturday" bson:"saturday"`
	Sunday    []TimeSlot `json:"sunday" bson:"sunday"`
}

type TimeSlot struct {
	StartTime string `json:"start_time" bson:"start_time"`
	EndTime   string `json:"end_time" bson:"end_time"`
}

type EmergencyContact struct {
	Name         string `json:"name" bson:"name"`
	Phone        string `json:"phone" bson:"phone"`
	Relationship string `json:"relationship" bson:"relationship"`
}