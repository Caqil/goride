package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type GeofenceType string

const (
	GeofenceTypeCircle  GeofenceType = "circle"
	GeofenceTypePolygon GeofenceType = "polygon"
)

type Geofence struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name" validate:"required"`
	Type        GeofenceType       `json:"type" bson:"type" validate:"required"`
	Coordinates [][]float64        `json:"coordinates" bson:"coordinates" validate:"required"`
	Radius      float64            `json:"radius" bson:"radius"` // for circle type
	City        string             `json:"city" bson:"city"`
	Country     string             `json:"country" bson:"country"`
	IsActive    bool               `json:"is_active" bson:"is_active" default:"true"`
	Properties  map[string]interface{} `json:"properties" bson:"properties"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}