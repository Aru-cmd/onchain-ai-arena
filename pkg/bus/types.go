package bus

// Bus types for trading arena - channel-agnostic inbound/outbound

type InboundContext struct {
	Channel string `json:"channel"` // telegram, discord
	Account string `json:"account,omitempty"`

	ChatID   string `json:"chat_id"`
	ChatType string `json:"chat_type,omitempty"` // direct, group, channel
	TopicID  string `json:"topic_id,omitempty"`

	SpaceID   string `json:"space_id,omitempty"`
	SpaceType string `json:"space_type,omitempty"`

	SenderID  string `json:"sender_id"`
	MessageID string `json:"message_id,omitempty"`

	Mentioned bool `json:"mentioned,omitempty"`

	ReplyToMessageID string `json:"reply_to_message_id,omitempty"`
	ReplyToSenderID  string `json:"reply_to_sender_id,omitempty"`
}

type SenderInfo struct {
	Platform    string `json:"platform,omitempty"`
	PlatformID  string `json:"platform_id,omitempty"`
	CanonicalID string `json:"canonical_id,omitempty"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

type InboundMessage struct {
	Context    InboundContext `json:"context"`
	Sender     SenderInfo     `json:"sender"`
	Content    string         `json:"content"`
	Media      []string       `json:"media,omitempty"`
	SessionKey string         `json:"session_key"`

	Channel   string `json:"channel"`
	SenderID  string `json:"sender_id"`
	ChatID    string `json:"chat_id"`
	MessageID string `json:"message_id,omitempty"`
}

type OutboundMessage struct {
	Channel          string         `json:"channel"`
	ChatID           string         `json:"chat_id"`
	Context          InboundContext `json:"context"`
	AgentID          string         `json:"agent_id,omitempty"`
	Content          string         `json:"content"`
	ReplyToMessageID string         `json:"reply_to_message_id,omitempty"`
}
