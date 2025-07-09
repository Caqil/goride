package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type RideRequest struct {
	ID                  primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	RiderID             primitive.ObjectID `json:"rider_id" bson:"rider_id" validate:"required"`
	RideType            RideType           `json:"ride_type" bson:"ride_type" validate:"required"`
	PickupLocation      Location           `json:"pickup_location" bson:"pickup_location" validate:"required"`
	DropoffLocation     Location           `json:"dropoff_location" bson:"dropoff_location" validate:"required"`
	Waypoints           []Location         `json:"waypoints" bson:"waypoints"`
	ScheduledTime       *time.Time         `json:"scheduled_time" bson:"scheduled_time"`
	EstimatedFare       float64            `json:"estimated_fare" bson:"estimated_fare"`
	EstimatedDuration   int                `json:"estimated_duration" bson:"estimated_duration"`
	EstimatedDistance   float64            `json:"estimated_distance" bson:"estimated_distance"`
	SurgeMultiplier     float64            `json:"surge_multiplier" bson:"surge_multiplier" default:"1.0"`
	SpecialRequests     []string           `json:"special_requests" bson:"special_requests"`
	PaymentMethodID     primitive.ObjectID `json:"payment_method_id" bson:"payment_method_id"`
	PromoCode           string             `json:"promo_code" bson:"promo_code"`
	IsShared            bool               `json:"is_shared" bson:"is_shared" default:"false"`
	MaxWaitTime         int                `json:"max_wait_time" bson:"max_wait_time" default:"5"` // minutes
	NearbyDrivers       []primitive.ObjectID `json:"nearby_drivers" bson:"nearby_drivers"`
	RequestedDrivers    []primitive.ObjectID `json:"requested_drivers" bson:"requested_drivers"`
	RejectedDrivers     []primitive.ObjectID `json:"rejected_drivers" bson:"rejected_drivers"`
	ExpiresAt           time.Time          `json:"expires_at" bson:"expires_at"`
	CreatedAt           time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at" bson:"updated_at"`
}