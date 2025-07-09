package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type RideStatus string
type RideType string

const (
	RideStatusRequested   RideStatus = "requested"
	RideStatusAccepted    RideStatus = "accepted"
	RideStatusDriverArrived RideStatus = "driver_arrived"
	RideStatusInProgress  RideStatus = "in_progress"
	RideStatusCompleted   RideStatus = "completed"
	RideStatusCancelled   RideStatus = "cancelled"
	RideStatusNoShow      RideStatus = "no_show"

	RideTypeStandard      RideType = "standard"
	RideTypePremium       RideType = "premium"
	RideTypeLuxury        RideType = "luxury"
	RideTypeShared        RideType = "shared"
	RideTypeXL            RideType = "xl"
	RideTypeAccessible    RideType = "accessible"
)

type Ride struct {
	ID                  primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	RideNumber          string             `json:"ride_number" bson:"ride_number" validate:"required"`
	RiderID             primitive.ObjectID `json:"rider_id" bson:"rider_id" validate:"required"`
	DriverID            *primitive.ObjectID `json:"driver_id" bson:"driver_id"`
	VehicleID           *primitive.ObjectID `json:"vehicle_id" bson:"vehicle_id"`
	RideType            RideType           `json:"ride_type" bson:"ride_type" validate:"required"`
	Status              RideStatus         `json:"status" bson:"status" default:"requested"`
	PickupLocation      Location           `json:"pickup_location" bson:"pickup_location" validate:"required"`
	DropoffLocation     Location           `json:"dropoff_location" bson:"dropoff_location" validate:"required"`
	Waypoints           []Location         `json:"waypoints" bson:"waypoints"`
	ScheduledTime       *time.Time         `json:"scheduled_time" bson:"scheduled_time"`
	RequestedAt         time.Time          `json:"requested_at" bson:"requested_at"`
	AcceptedAt          *time.Time         `json:"accepted_at" bson:"accepted_at"`
	DriverArrivedAt     *time.Time         `json:"driver_arrived_at" bson:"driver_arrived_at"`
	StartedAt           *time.Time         `json:"started_at" bson:"started_at"`
	CompletedAt         *time.Time         `json:"completed_at" bson:"completed_at"`
	CancelledAt         *time.Time         `json:"cancelled_at" bson:"cancelled_at"`
	CancellationReason  string             `json:"cancellation_reason" bson:"cancellation_reason"`
	CancelledBy         string             `json:"cancelled_by" bson:"cancelled_by"`
	EstimatedDuration   int                `json:"estimated_duration" bson:"estimated_duration"` // minutes
	EstimatedDistance   float64            `json:"estimated_distance" bson:"estimated_distance"` // kilometers
	ActualDuration      int                `json:"actual_duration" bson:"actual_duration"`
	ActualDistance      float64            `json:"actual_distance" bson:"actual_distance"`
	EstimatedFare       float64            `json:"estimated_fare" bson:"estimated_fare"`
	ActualFare          float64            `json:"actual_fare" bson:"actual_fare"`
	SurgeMultiplier     float64            `json:"surge_multiplier" bson:"surge_multiplier" default:"1.0"`
	Currency            string             `json:"currency" bson:"currency" default:"USD"`
	Route               *Route             `json:"route" bson:"route"`
	PaymentID           *primitive.ObjectID `json:"payment_id" bson:"payment_id"`
	RiderRating         *float64           `json:"rider_rating" bson:"rider_rating"`
	DriverRating        *float64           `json:"driver_rating" bson:"driver_rating"`
	SpecialRequests     []string           `json:"special_requests" bson:"special_requests"`
	PromoCode           string             `json:"promo_code" bson:"promo_code"`
	TipAmount           float64            `json:"tip_amount" bson:"tip_amount" default:"0"`
	IsShared            bool               `json:"is_shared" bson:"is_shared" default:"false"`
	SharedWith          []primitive.ObjectID `json:"shared_with" bson:"shared_with"`
	OTP                 string             `json:"otp" bson:"otp"`
	CreatedAt           time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at" bson:"updated_at"`
}