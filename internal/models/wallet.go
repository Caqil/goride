package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type Wallet struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id" validate:"required"`
	Balance   float64            `json:"balance" bson:"balance" default:"0"`
	Currency  string             `json:"currency" bson:"currency" default:"USD"`
	IsActive  bool               `json:"is_active" bson:"is_active" default:"true"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}