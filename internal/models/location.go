package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Location struct {
	Type        string    `json:"type" bson:"type" default:"Point"`
	Coordinates []float64 `json:"coordinates" bson:"coordinates" validate:"required,len=2"`
	Address     string    `json:"address" bson:"address"`
	City        string    `json:"city" bson:"city"`
	State       string    `json:"state" bson:"state"`
	Country     string    `json:"country" bson:"country"`
	PostalCode  string    `json:"postal_code" bson:"postal_code"`
	PlaceID     string    `json:"place_id" bson:"place_id"`
	Timestamp   time.Time `json:"timestamp" bson:"timestamp"`
}

func (l Location) Latitude() float64 {
	if len(l.Coordinates) >= 2 {
		return l.Coordinates[1]
	}
	return 0
}

func (l Location) Longitude() float64 {
	if len(l.Coordinates) >= 1 {
		return l.Coordinates[0]
	}
	return 0
}

type LocationHistory struct {
	ID        primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	UserID    *primitive.ObjectID `json:"user_id" bson:"user_id"`
	RideID    *primitive.ObjectID `json:"ride_id" bson:"ride_id"`
	UserType  UserType            `json:"user_type" bson:"user_type"`
	Location  Location            `json:"location" bson:"location" validate:"required"`
	Accuracy  float64             `json:"accuracy" bson:"accuracy"`
	Speed     float64             `json:"speed" bson:"speed"`
	Bearing   float64             `json:"bearing" bson:"bearing"`
	Activity  string              `json:"activity" bson:"activity"` // driving, walking, stationary
	Source    string              `json:"source" bson:"source"`     // gps, network, passive
	IsActive  bool                `json:"is_active" bson:"is_active" default:"true"`
	CreatedAt time.Time           `json:"created_at" bson:"created_at"`
}
