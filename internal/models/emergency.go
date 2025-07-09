package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type EmergencyType string
type EmergencyStatus string

const (
	EmergencyTypeSOS       EmergencyType = "sos"
	EmergencyTypeAccident  EmergencyType = "accident"
	EmergencyTypeMedical   EmergencyType = "medical"
	EmergencyTypeSafety    EmergencyType = "safety"
	EmergencyTypeBreakdown EmergencyType = "breakdown"

	EmergencyStatusActive   EmergencyStatus = "active"
	EmergencyStatusResolved EmergencyStatus = "resolved"
	EmergencyStatusFalse    EmergencyStatus = "false_alarm"
)

type Emergency struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID          primitive.ObjectID `json:"user_id" bson:"user_id" validate:"required"`
	RideID          *primitive.ObjectID `json:"ride_id" bson:"ride_id"`
	Type            EmergencyType      `json:"type" bson:"type" validate:"required"`
	Status          EmergencyStatus    `json:"status" bson:"status" default:"active"`
	Location        Location           `json:"location" bson:"location" validate:"required"`
	Description     string             `json:"description" bson:"description"`
	ContactedBy     []string           `json:"contacted_by" bson:"contacted_by"`
	EmergencyNumber string             `json:"emergency_number" bson:"emergency_number"`
	ResponseTime    *time.Time         `json:"response_time" bson:"response_time"`
	ResolvedBy      *primitive.ObjectID `json:"resolved_by" bson:"resolved_by"`
	Notes           string             `json:"notes" bson:"notes"`
	CreatedAt       time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at" bson:"updated_at"`
	ResolvedAt      *time.Time         `json:"resolved_at" bson:"resolved_at"`
}