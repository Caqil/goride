package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type Review struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	RatingID  primitive.ObjectID `json:"rating_id" bson:"rating_id" validate:"required"`
	RideID    primitive.ObjectID `json:"ride_id" bson:"ride_id" validate:"required"`
	ReviewerID primitive.ObjectID `json:"reviewer_id" bson:"reviewer_id" validate:"required"`
	RevieweeID primitive.ObjectID `json:"reviewee_id" bson:"reviewee_id" validate:"required"`
	ReviewerType UserType         `json:"reviewer_type" bson:"reviewer_type" validate:"required"`
	Rating    float64            `json:"rating" bson:"rating" validate:"required,min=1,max=5"`
	Title     string             `json:"title" bson:"title"`
	Comment   string             `json:"comment" bson:"comment"`
	Pros      []string           `json:"pros" bson:"pros"`
	Cons      []string           `json:"cons" bson:"cons"`
	Tags      []string           `json:"tags" bson:"tags"`
	IsPublic  bool               `json:"is_public" bson:"is_public" default:"true"`
	IsVerified bool              `json:"is_verified" bson:"is_verified" default:"true"`
	HelpfulCount int             `json:"helpful_count" bson:"helpful_count" default:"0"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}
