package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type Route struct {
	ID                primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	RideID            primitive.ObjectID `json:"ride_id" bson:"ride_id" validate:"required"`
	StartLocation     Location           `json:"start_location" bson:"start_location"`
	EndLocation       Location           `json:"end_location" bson:"end_location"`
	Waypoints         []Location         `json:"waypoints" bson:"waypoints"`
	EncodedPolyline   string             `json:"encoded_polyline" bson:"encoded_polyline"`
	Distance          float64            `json:"distance" bson:"distance"` // kilometers
	Duration          int                `json:"duration" bson:"duration"` // seconds
	TrafficDuration   int                `json:"traffic_duration" bson:"traffic_duration"`
	Steps             []RouteStep        `json:"steps" bson:"steps"`
	Bounds            *RouteBounds       `json:"bounds" bson:"bounds"`
	CreatedAt         time.Time          `json:"created_at" bson:"created_at"`
}

type RouteStep struct {
	Instruction      string    `json:"instruction" bson:"instruction"`
	Distance         float64   `json:"distance" bson:"distance"`
	Duration         int       `json:"duration" bson:"duration"`
	StartLocation    Location  `json:"start_location" bson:"start_location"`
	EndLocation      Location  `json:"end_location" bson:"end_location"`
	EncodedPolyline  string    `json:"encoded_polyline" bson:"encoded_polyline"`
	Maneuver         string    `json:"maneuver" bson:"maneuver"`
}

type RouteBounds struct {
	Northeast Location `json:"northeast" bson:"northeast"`
	Southwest Location `json:"southwest" bson:"southwest"`
}
