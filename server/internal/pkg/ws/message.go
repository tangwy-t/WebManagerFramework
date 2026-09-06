// Package ws provides WebSocket connection management and real-time push.
package ws

import "encoding/json"

// Message type constants for client->server communication.
const (
	MsgTypeAuth = "auth"
	MsgTypePing = "ping"
)

// Message type constants for server->client communication.
const (
	MsgTypeAuthOK    = "auth_ok"
	MsgTypeAuthErr   = "auth_err"
	MsgTypeNewNotice = "new_notice"
	MsgTypePong      = "pong"
	MsgTypeKicked    = "kicked"
)

// ClientMessage is the JSON message sent from client to server.
type ClientMessage struct {
	Type  string `json:"type"`
	Token string `json:"token,omitempty"`
}

// ServerMessage is the JSON message sent from server to client.
type ServerMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

// NewServerMessage creates a ServerMessage and marshals it to JSON bytes.
func NewServerMessage(msgType string, data interface{}) ([]byte, error) {
	msg := ServerMessage{Type: msgType, Data: data}
	return json.Marshal(msg)
}

// PushEvent is the message published to Redis Pub/Sub for cross-instance broadcast.
// When IsAll is true, the receiving Hub pushes to all online users.
// When IsAll is false, the receiving Hub pushes only to the specified UserIDs.
type PushEvent struct {
	Type       string          `json:"type"`
	NoticeData json.RawMessage `json:"notice_data"`
	UserIDs    []uint64        `json:"user_ids,omitempty"`
	IsAll      bool            `json:"is_all"`
}

// KickEvent is published to Redis channel "user:kick" for cross-instance single sign-on.
type KickEvent struct {
	UserID    uint64 `json:"user_id"`
	Reason    string `json:"reason"`
	KickToken string `json:"kick_token,omitempty"`
}
