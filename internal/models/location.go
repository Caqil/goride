package models

import (
	"time"
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
