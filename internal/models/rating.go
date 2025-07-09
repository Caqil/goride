package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type Rating struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	RideID     primitive.ObjectID `json:"ride_id" bson:"ride_id" validate:"required"`
	RaterID    primitive.ObjectID `json:"rater_id" bson:"rater_id" validate:"required"`
	RatedID    primitive.ObjectID `json:"rated_id" bson:"rated_id" validate:"required"`
	RaterType  UserType           `json:"rater_type" bson:"rater_type" validate:"required"`
	Rating     float64            `json:"rating" bson:"rating" validate:"required,min=1,max=5"`
	Comment    string             `json:"comment" bson:"comment"`
	Tags       []string           `json:"tags" bson:"tags"`
	IsAnonymous bool              `json:"is_anonymous" bson:"is_anonymous" default:"false"`
	CreatedAt  time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at" bson:"updated_at"`
}
