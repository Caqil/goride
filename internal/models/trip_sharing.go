package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type TripSharingStatus string

const (
	TripSharingStatusActive   TripSharingStatus = "active"
	TripSharingStatusExpired  TripSharingStatus = "expired"
	TripSharingStatusCancelled TripSharingStatus = "cancelled"
)

type TripSharing struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	RideID          primitive.ObjectID `json:"ride_id" bson:"ride_id" validate:"required"`
	SharedByID      primitive.ObjectID `json:"shared_by_id" bson:"shared_by_id" validate:"required"`
	ShareToken      string             `json:"share_token" bson:"share_token" validate:"required"`
	ShareURL        string             `json:"share_url" bson:"share_url" validate:"required"`
	Status          TripSharingStatus  `json:"status" bson:"status" default:"active"`
	SharedContacts  []SharedContact    `json:"shared_contacts" bson:"shared_contacts"`
	ViewCount       int                `json:"view_count" bson:"view_count" default:"0"`
	LastViewedAt    *time.Time         `json:"last_viewed_at" bson:"last_viewed_at"`
	ExpiresAt       time.Time          `json:"expires_at" bson:"expires_at"`
	CreatedAt       time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at" bson:"updated_at"`
}

type SharedContact struct {
	Name        string    `json:"name" bson:"name"`
	Phone       string    `json:"phone" bson:"phone"`
	Email       string    `json:"email" bson:"email"`
	ShareMethod string    `json:"share_method" bson:"share_method"` // sms, email, link
	SentAt      time.Time `json:"sent_at" bson:"sent_at"`
	ViewedAt    *time.Time `json:"viewed_at" bson:"viewed_at"`
}