package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type VehicleStatus string

const (
	VehicleStatusActive      VehicleStatus = "active"
	VehicleStatusInactive    VehicleStatus = "inactive"
	VehicleStatusMaintenance VehicleStatus = "maintenance"
	VehicleStatusSuspended   VehicleStatus = "suspended"
)

type Vehicle struct {
	ID                   primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	DriverID             primitive.ObjectID `json:"driver_id" bson:"driver_id" validate:"required"`
	VehicleTypeID        primitive.ObjectID `json:"vehicle_type_id" bson:"vehicle_type_id" validate:"required"`
	Make                 string             `json:"make" bson:"make" validate:"required"`
	Model                string             `json:"model" bson:"model" validate:"required"`
	Year                 int                `json:"year" bson:"year" validate:"required"`
	Color                string             `json:"color" bson:"color" validate:"required"`
	LicensePlate         string             `json:"license_plate" bson:"license_plate" validate:"required"`
	VIN                  string             `json:"vin" bson:"vin"`
	RegistrationNumber   string             `json:"registration_number" bson:"registration_number"`
	RegistrationExpiry   time.Time          `json:"registration_expiry" bson:"registration_expiry"`
	RegistrationDocument string             `json:"registration_document" bson:"registration_document"`
	RegistrationStatus   DocumentStatus     `json:"registration_status" bson:"registration_status" default:"pending"`
	InsuranceNumber      string             `json:"insurance_number" bson:"insurance_number"`
	InsuranceExpiry      time.Time          `json:"insurance_expiry" bson:"insurance_expiry"`
	InsuranceDocument    string             `json:"insurance_document" bson:"insurance_document"`
	InsuranceStatus      DocumentStatus     `json:"insurance_status" bson:"insurance_status" default:"pending"`
	Status               VehicleStatus      `json:"status" bson:"status" default:"inactive"`
	Capacity             int                `json:"capacity" bson:"capacity" validate:"required"`
	Photos               []string           `json:"photos" bson:"photos"`
	Features             []string           `json:"features" bson:"features"`
	IsAccessible         bool               `json:"is_accessible" bson:"is_accessible" default:"false"`
	TotalRides           int64              `json:"total_rides" bson:"total_rides" default:"0"`
	TotalDistance        float64            `json:"total_distance" bson:"total_distance" default:"0"`
	LastMaintenanceAt    *time.Time         `json:"last_maintenance_at" bson:"last_maintenance_at"`
	NextMaintenanceAt    *time.Time         `json:"next_maintenance_at" bson:"next_maintenance_at"`
	CreatedAt            time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at" bson:"updated_at"`
	ApprovedAt           *time.Time         `json:"approved_at" bson:"approved_at"`
}