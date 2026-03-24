package shared

import "time"

// MessageType distinguishes between text and file messages
type MessageType string

const (
	TextMessage      MessageType = "text"
	FileMessageType  MessageType = "file"
	AdminCommandType MessageType = "admin_command"
	EditMessageType  MessageType = "edit"
	DeleteMessage    MessageType = "delete"
	TypingMessage    MessageType = "typing"
	ReactionMessage  MessageType = "reaction"
	DirectMessage    MessageType = "dm"
	SearchMessage    MessageType = "search"
	PinMessage       MessageType = "pin"
	ReadReceiptType  MessageType = "read_receipt"
	JoinChannelType  MessageType = "join_channel"
	LeaveChannelType MessageType = "leave_channel"
	ListChannelsType MessageType = "list_channels"
)

type Message struct {
	Sender    string      `json:"sender"`
	Content   string      `json:"content"`
	CreatedAt time.Time   `json:"created_at"`
	Type      MessageType `json:"type,omitempty"`
	Encrypted bool        `json:"encrypted,omitempty"`

	// MessageID uniquely identifies a message for edits, deletes, reactions, pins
	MessageID int64 `json:"message_id,omitempty"`

	// Recipient for direct messages (empty = broadcast to all)
	Recipient string `json:"recipient,omitempty"`

	// Edited indicates the message has been modified
	Edited bool `json:"edited,omitempty"`

	Channel string `json:"channel,omitempty"`

	// Reaction metadata
	Reaction *ReactionMeta `json:"reaction,omitempty"`

	// For file messages, Content is empty and File is set
	File *FileMeta `json:"file,omitempty"`
}

type FileMeta struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Data     []byte `json:"data"` // raw bytes (base64-encoded in JSON)
}

// ReactionMeta contains reaction data for a message
type ReactionMeta struct {
	Emoji     string `json:"emoji"`
	TargetID  int64  `json:"target_id"`
	IsRemoval bool   `json:"is_removal,omitempty"`
}

// Handshake is sent by the client on WebSocket connect for authentication
type Handshake struct {
	Username string `json:"username"`
	Admin    bool   `json:"admin"`
	AdminKey string `json:"admin_key,omitempty"`
}
