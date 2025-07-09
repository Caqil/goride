package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type SurgePricing struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Area         string             `json:"area" bson:"area" validate:"required"`
	GeofenceID   primitive.ObjectID `json:"geofence_id" bson:"geofence_id"`
	RideTypes    []RideType         `json:"ride_types" bson:"ride_types"`
	Multiplier   float64            `json:"multiplier" bson:"multiplier" validate:"required,min=1"`
	Demand       int                `json:"demand" bson:"demand"`
	Supply       int                `json:"supply" bson:"supply"`
	IsActive     bool               `json:"is_active" bson:"is_active" default:"true"`
	StartTime    time.Time          `json:"start_time" bson:"start_time"`
	EndTime      *time.Time         `json:"end_time" bson:"end_time"`
	Reason       string             `json:"reason" bson:"reason"`
	CreatedAt    time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at" bson:"updated_at"`
}