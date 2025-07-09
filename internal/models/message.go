package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type MessageType string
type MessageStatus string

const (
	MessageTypeText     MessageType = "text"
	MessageTypeImage    MessageType = "image"
	MessageTypeAudio    MessageType = "audio"
	MessageTypeLocation MessageType = "location"
	MessageTypeFile     MessageType = "file"

	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
	MessageStatusFailed    MessageStatus = "failed"
)

type Message struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	ChatID     primitive.ObjectID `json:"chat_id" bson:"chat_id" validate:"required"`
	SenderID   primitive.ObjectID `json:"sender_id" bson:"sender_id" validate:"required"`
	Type       MessageType        `json:"type" bson:"type" default:"text"`
	Status     MessageStatus      `json:"status" bson:"status" default:"sent"`
	Content    string             `json:"content" bson:"content"`
	MediaURL   string             `json:"media_url" bson:"media_url"`
	FileSize   int64              `json:"file_size" bson:"file_size"`
	Duration   int                `json:"duration" bson:"duration"` // for audio messages
	Location   *Location          `json:"location" bson:"location"`
	IsEncrypted bool              `json:"is_encrypted" bson:"is_encrypted" default:"false"`
	ReadBy     []ReadReceipt      `json:"read_by" bson:"read_by"`
	CreatedAt  time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at" bson:"updated_at"`
	DeletedAt  *time.Time         `json:"deleted_at" bson:"deleted_at"`
}

type ReadReceipt struct {
	UserID primitive.ObjectID `json:"user_id" bson:"user_id"`
	ReadAt time.Time          `json:"read_at" bson:"read_at"`
}
