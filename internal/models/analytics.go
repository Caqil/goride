package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type AnalyticsEvent struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID     *primitive.ObjectID `json:"user_id" bson:"user_id"`
	SessionID  string             `json:"session_id" bson:"session_id"`
	EventType  string             `json:"event_type" bson:"event_type" validate:"required"`
	EventName  string             `json:"event_name" bson:"event_name" validate:"required"`
	Properties map[string]interface{} `json:"properties" bson:"properties"`
	UserAgent  string             `json:"user_agent" bson:"user_agent"`
	IPAddress  string             `json:"ip_address" bson:"ip_address"`
	Location   *Location          `json:"location" bson:"location"`
	Platform   string             `json:"platform" bson:"platform"`
	AppVersion string             `json:"app_version" bson:"app_version"`
	CreatedAt  time.Time          `json:"created_at" bson:"created_at"`
}

type RideAnalytics struct {
	ID                 primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Date               time.Time          `json:"date" bson:"date"`
	City               string             `json:"city" bson:"city"`
	TotalRides         int64              `json:"total_rides" bson:"total_rides"`
	CompletedRides     int64              `json:"completed_rides" bson:"completed_rides"`
	CancelledRides     int64              `json:"cancelled_rides" bson:"cancelled_rides"`
	TotalRevenue       float64            `json:"total_revenue" bson:"total_revenue"`
	AverageRideValue   float64            `json:"average_ride_value" bson:"average_ride_value"`
	AverageRideTime    float64            `json:"average_ride_time" bson:"average_ride_time"`
	AverageRideDistance float64           `json:"average_ride_distance" bson:"average_ride_distance"`
	PeakHours          []string           `json:"peak_hours" bson:"peak_hours"`
	PopularRoutes      []RouteStats       `json:"popular_routes" bson:"popular_routes"`
	CreatedAt          time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at" bson:"updated_at"`
}

type RouteStats struct {
	PickupArea   string  `json:"pickup_area" bson:"pickup_area"`
	DropoffArea  string  `json:"dropoff_area" bson:"dropoff_area"`
	RideCount    int64   `json:"ride_count" bson:"ride_count"`
	AverageFare  float64 `json:"average_fare" bson:"average_fare"`
}