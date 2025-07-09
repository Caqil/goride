package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RideTypeModel struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name            string             `json:"name" bson:"name" validate:"required"`
	DisplayName     string             `json:"display_name" bson:"display_name" validate:"required"`
	Description     string             `json:"description" bson:"description"`
	Icon            string             `json:"icon" bson:"icon"`
	Image           string             `json:"image" bson:"image"`
	BasePrice       float64            `json:"base_price" bson:"base_price" validate:"required"`
	PricePerKM      float64            `json:"price_per_km" bson:"price_per_km" validate:"required"`
	PricePerMinute  float64            `json:"price_per_minute" bson:"price_per_minute" validate:"required"`
	MinimumFare     float64            `json:"minimum_fare" bson:"minimum_fare" validate:"required"`
	MaxCapacity     int                `json:"max_capacity" bson:"max_capacity" validate:"required"`
	LuggageCapacity int                `json:"luggage_capacity" bson:"luggage_capacity"`
	Features        []string           `json:"features" bson:"features"`
	IsActive        bool               `json:"is_active" bson:"is_active" default:"true"`
	IsAccessible    bool               `json:"is_accessible" bson:"is_accessible" default:"false"`
	SortOrder       int                `json:"sort_order" bson:"sort_order" default:"0"`
	CreatedAt       time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at" bson:"updated_at"`
}
