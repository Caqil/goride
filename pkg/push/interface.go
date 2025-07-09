package push

import "context"

type PushProvider interface {
	SendNotification(ctx context.Context, request *NotificationRequest) (*NotificationResponse, error)
	SendBulkNotifications(ctx context.Context, requests []*NotificationRequest) ([]*NotificationResponse, error)
	SubscribeToTopic(ctx context.Context, tokens []string, topic string) error
	UnsubscribeFromTopic(ctx context.Context, tokens []string, topic string) error
	ValidateToken(ctx context.Context, token string) (bool, error)
}

type NotificationRequest struct {
	Token       string              `json:"token"`
	Tokens      []string            `json:"tokens,omitempty"`
	Topic       string              `json:"topic,omitempty"`
	Title       string              `json:"title"`
	Body        string              `json:"body"`
	Data        map[string]string   `json:"data,omitempty"`
	ImageURL    string              `json:"image_url,omitempty"`
	Sound       string              `json:"sound,omitempty"`
	Badge       int                 `json:"badge,omitempty"`
	Priority    string              `json:"priority,omitempty"`
	TTL         int                 `json:"ttl,omitempty"`
	CollapseKey string              `json:"collapse_key,omitempty"`
	Action      *NotificationAction `json:"action,omitempty"`
	iOS         *IOSConfig          `json:"ios,omitempty"`
	Android     *AndroidConfig      `json:"android,omitempty"`
}

type NotificationResponse struct {
	MessageID   string `json:"message_id"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
	Token       string `json:"token,omitempty"`
	CanonicalID string `json:"canonical_id,omitempty"`
}

type NotificationAction struct {
	ClickAction string `json:"click_action"`
	URL         string `json:"url"`
}

type IOSConfig struct {
	Sound            string            `json:"sound,omitempty"`
	Badge            int               `json:"badge,omitempty"`
	ContentAvailable bool              `json:"content_available,omitempty"`
	MutableContent   bool              `json:"mutable_content,omitempty"`
	Category         string            `json:"category,omitempty"`
	CustomData       map[string]string `json:"custom_data,omitempty"`
}

type AndroidConfig struct {
	Priority     string            `json:"priority,omitempty"`
	Sound        string            `json:"sound,omitempty"`
	Color        string            `json:"color,omitempty"`
	Tag          string            `json:"tag,omitempty"`
	ClickAction  string            `json:"click_action,omitempty"`
	BodyLocKey   string            `json:"body_loc_key,omitempty"`
	BodyLocArgs  []string          `json:"body_loc_args,omitempty"`
	TitleLocKey  string            `json:"title_loc_key,omitempty"`
	TitleLocArgs []string          `json:"title_loc_args,omitempty"`
	ChannelID    string            `json:"channel_id,omitempty"`
	CustomData   map[string]string `json:"custom_data,omitempty"`
}
