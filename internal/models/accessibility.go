package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type AccessibilityOption struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name" validate:"required"`
	Code        string             `json:"code" bson:"code" validate:"required"`
	Description string             `json:"description" bson:"description"`
	Icon        string             `json:"icon" bson:"icon"`
	IsActive    bool               `json:"is_active" bson:"is_active" default:"true"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}