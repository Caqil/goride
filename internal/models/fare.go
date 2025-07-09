package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type FareStructure struct {
	ID               primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	City             string             `json:"city" bson:"city" validate:"required"`
	RideType         RideType           `json:"ride_type" bson:"ride_type" validate:"required"`
	BaseFare         float64            `json:"base_fare" bson:"base_fare" validate:"required"`
	PricePerKM       float64            `json:"price_per_km" bson:"price_per_km" validate:"required"`
	PricePerMinute   float64            `json:"price_per_minute" bson:"price_per_minute" validate:"required"`
	MinimumFare      float64            `json:"minimum_fare" bson:"minimum_fare" validate:"required"`
	MaximumFare      float64            `json:"maximum_fare" bson:"maximum_fare"`
	BookingFee       float64            `json:"booking_fee" bson:"booking_fee" default:"0"`
	CancellationFee  float64            `json:"cancellation_fee" bson:"cancellation_fee" default:"0"`
	WaitingTimeRate  float64            `json:"waiting_time_rate" bson:"waiting_time_rate" default:"0"`
	FreeWaitingTime  int                `json:"free_waiting_time" bson:"free_waiting_time" default:"5"` // minutes
	Currency         string             `json:"currency" bson:"currency" default:"USD"`
	IsActive         bool               `json:"is_active" bson:"is_active" default:"true"`
	EffectiveFrom    time.Time          `json:"effective_from" bson:"effective_from"`
	EffectiveUntil   *time.Time         `json:"effective_until" bson:"effective_until"`
	CreatedAt        time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at" bson:"updated_at"`
}