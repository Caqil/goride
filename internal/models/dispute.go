package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type DisputeType string
type DisputeStatus string

const (
	DisputeTypeFare        DisputeType = "fare"
	DisputeTypeService     DisputeType = "service"
	DisputeTypeDriver      DisputeType = "driver"
	DisputeTypeRider       DisputeType = "rider"
	DisputeTypePayment     DisputeType = "payment"
	DisputeTypeCancellation DisputeType = "cancellation"
	DisputeTypeRoute       DisputeType = "route"
	DisputeTypeOther       DisputeType = "other"

	DisputeStatusOpen      DisputeStatus = "open"
	DisputeStatusInReview  DisputeStatus = "in_review"
	DisputeStatusResolved  DisputeStatus = "resolved"
	DisputeStatusClosed    DisputeStatus = "closed"
	DisputeStatusEscalated DisputeStatus = "escalated"
)

type Dispute struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	DisputeNumber   string             `json:"dispute_number" bson:"dispute_number" validate:"required"`
	RideID          primitive.ObjectID `json:"ride_id" bson:"ride_id" validate:"required"`
	RaisedByID      primitive.ObjectID `json:"raised_by_id" bson:"raised_by_id" validate:"required"`
	RaisedAgainstID primitive.ObjectID `json:"raised_against_id" bson:"raised_against_id"`
	Type            DisputeType        `json:"type" bson:"type" validate:"required"`
	Status          DisputeStatus      `json:"status" bson:"status" default:"open"`
	Subject         string             `json:"subject" bson:"subject" validate:"required"`
	Description     string             `json:"description" bson:"description" validate:"required"`
	Evidence        []DisputeEvidence  `json:"evidence" bson:"evidence"`
	AssignedTo      *primitive.ObjectID `json:"assigned_to" bson:"assigned_to"`
	Resolution      string             `json:"resolution" bson:"resolution"`
	RefundAmount    float64            `json:"refund_amount" bson:"refund_amount" default:"0"`
	Priority        int                `json:"priority" bson:"priority" default:"1"`
	Comments        []DisputeComment   `json:"comments" bson:"comments"`
	CreatedAt       time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at" bson:"updated_at"`
	ResolvedAt      *time.Time         `json:"resolved_at" bson:"resolved_at"`
}

type DisputeEvidence struct {
	Type        string    `json:"type" bson:"type"` // image, video, audio, document
	URL         string    `json:"url" bson:"url"`
	Description string    `json:"description" bson:"description"`
	UploadedAt  time.Time `json:"uploaded_at" bson:"uploaded_at"`
}

type DisputeComment struct {
	AuthorID  primitive.ObjectID `json:"author_id" bson:"author_id"`
	Comment   string             `json:"comment" bson:"comment"`
	IsInternal bool              `json:"is_internal" bson:"is_internal" default:"false"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
}
