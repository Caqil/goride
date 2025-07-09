package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type SOSStatus string

const (
	SOSStatusActive    SOSStatus = "active"
	SOSStatusResolved  SOSStatus = "resolved"
	SOSStatusFalseAlarm SOSStatus = "false_alarm"
)

type SOS struct {
	ID                 primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID             primitive.ObjectID `json:"user_id" bson:"user_id" validate:"required"`
	RideID             *primitive.ObjectID `json:"ride_id" bson:"ride_id"`
	EmergencyID        primitive.ObjectID `json:"emergency_id" bson:"emergency_id" validate:"required"`
	Status             SOSStatus          `json:"status" bson:"status" default:"active"`
	Location           Location           `json:"location" bson:"location" validate:"required"`
	Message            string             `json:"message" bson:"message"`
	AudioRecording     string             `json:"audio_recording" bson:"audio_recording"`
	VideoRecording     string             `json:"video_recording" bson:"video_recording"`
	Photos             []string           `json:"photos" bson:"photos"`
	EmergencyContacts  []EmergencyContact `json:"emergency_contacts" bson:"emergency_contacts"`
	AuthoritiesContacted bool             `json:"authorities_contacted" bson:"authorities_contacted" default:"false"`
	ResponseTeamID     *primitive.ObjectID `json:"response_team_id" bson:"response_team_id"`
	IncidentNumber     string             `json:"incident_number" bson:"incident_number"`
	CreatedAt          time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at" bson:"updated_at"`
	ResolvedAt         *time.Time         `json:"resolved_at" bson:"resolved_at"`
}