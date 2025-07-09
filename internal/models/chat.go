package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type ChatStatus string

const (
	ChatStatusActive ChatStatus = "active"
	ChatStatusClosed ChatStatus = "closed"
)

type Chat struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	RideID       primitive.ObjectID `json:"ride_id" bson:"ride_id" validate:"required"`
	Participants []primitive.ObjectID `json:"participants" bson:"participants" validate:"required"`
	Status       ChatStatus         `json:"status" bson:"status" default:"active"`
	LastMessage  *Message           `json:"last_message" bson:"last_message"`
	CreatedAt    time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at" bson:"updated_at"`
	ClosedAt     *time.Time         `json:"closed_at" bson:"closed_at"`
}
