package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CallType string
type CallStatus string
type CallDirection string

const (
	// Call Types
	CallTypeVoice         CallType = "voice"
	CallTypeEmergency     CallType = "emergency"
	CallTypeSupport       CallType = "support"
	CallTypeDriverToRider CallType = "driver_to_rider"
	CallTypeRiderToDriver CallType = "rider_to_driver"

	// Call Status
	CallStatusInitiated CallStatus = "initiated"
	CallStatusRinging   CallStatus = "ringing"
	CallStatusAnswered  CallStatus = "answered"
	CallStatusCompleted CallStatus = "completed"
	CallStatusFailed    CallStatus = "failed"
	CallStatusCancelled CallStatus = "cancelled"
	CallStatusBusy      CallStatus = "busy"
	CallStatusNoAnswer  CallStatus = "no_answer"

	// Call Direction
	CallDirectionInbound  CallDirection = "inbound"
	CallDirectionOutbound CallDirection = "outbound"
)

type Call struct {
	ID            primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	CallSID       string              `json:"call_sid" bson:"call_sid"`
	RideID        *primitive.ObjectID `json:"ride_id" bson:"ride_id"`
	CallerID      primitive.ObjectID  `json:"caller_id" bson:"caller_id"`
	CalleeID      primitive.ObjectID  `json:"callee_id" bson:"callee_id"`
	Type          CallType            `json:"type" bson:"type"`
	Status        CallStatus          `json:"status" bson:"status"`
	Direction     CallDirection       `json:"direction" bson:"direction"`
	FromNumber    string              `json:"from_number" bson:"from_number"`
	ToNumber      string              `json:"to_number" bson:"to_number"`
	ProxyNumber   string              `json:"proxy_number" bson:"proxy_number"`
	Duration      int                 `json:"duration" bson:"duration"` // in seconds
	RecordingURL  string              `json:"recording_url" bson:"recording_url"`
	Cost          float64             `json:"cost" bson:"cost"`
	IsRecorded    bool                `json:"is_recorded" bson:"is_recorded"`
	IsEmergency   bool                `json:"is_emergency" bson:"is_emergency"`
	QualityRating int                 `json:"quality_rating" bson:"quality_rating"`
	StartTime     *time.Time          `json:"start_time" bson:"start_time"`
	EndTime       *time.Time          `json:"end_time" bson:"end_time"`
	CreatedAt     time.Time           `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at" bson:"updated_at"`
}

type CallRequest struct {
	CallerID   primitive.ObjectID  `json:"caller_id" validate:"required"`
	CalleeID   primitive.ObjectID  `json:"callee_id" validate:"required"`
	RideID     *primitive.ObjectID `json:"ride_id"`
	Type       CallType            `json:"type" validate:"required"`
	IsRecorded bool                `json:"is_recorded"`
}

type CallResponse struct {
	CallID      primitive.ObjectID `json:"call_id"`
	CallSID     string             `json:"call_sid"`
	ProxyNumber string             `json:"proxy_number"`
	Status      CallStatus         `json:"status"`
	Message     string             `json:"message"`
}

type CallAnalytics struct {
	TotalCalls      int64                `json:"total_calls"`
	SuccessfulCalls int64                `json:"successful_calls"`
	FailedCalls     int64                `json:"failed_calls"`
	AverageDuration float64              `json:"average_duration"`
	TotalDuration   int64                `json:"total_duration"`
	CallsByType     map[CallType]int64   `json:"calls_by_type"`
	CallsByStatus   map[CallStatus]int64 `json:"calls_by_status"`
	AverageQuality  float64              `json:"average_quality"`
	EmergencyCalls  int64                `json:"emergency_calls"`
}

type CallAnalyticsParams struct {
	StartDate *time.Time          `json:"start_date"`
	EndDate   *time.Time          `json:"end_date"`
	CallType  *CallType           `json:"call_type"`
	UserID    *primitive.ObjectID `json:"user_id"`
}
