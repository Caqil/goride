package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type VehicleType struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name" validate:"required"`
	DisplayName string             `json:"display_name" bson:"display_name" validate:"required"`
	Description string             `json:"description" bson:"description"`
	Icon        string             `json:"icon" bson:"icon"`
	Image       string             `json:"image" bson:"image"`
	MinCapacity int                `json:"min_capacity" bson:"min_capacity" validate:"required"`
	MaxCapacity int                `json:"max_capacity" bson:"max_capacity" validate:"required"`
	Features    []string           `json:"features" bson:"features"`
	IsActive    bool               `json:"is_active" bson:"is_active" default:"true"`
	SortOrder   int                `json:"sort_order" bson:"sort_order" default:"0"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}