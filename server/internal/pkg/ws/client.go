package ws

import (
	"context"
	"encoding/json"
	"github.com/tangwy-t/webmanager-server/internal/pkg/contextkeys"
	"github.com/tangwy-t/webmanager-server/internal/pkg/logger"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
	sendBufferSize = 256
	kickTimeout    = 5 * time.Second
)

// ws 客户端私有状态键(连接级,非身份):类型化避免与字符串键冲突。
// 身份(userID)不在此列:统一走 contextkeys.UserID,与 HTTP Auth 中间件
// 注入的键同源,服务层用 contextkeys.UserIDFromCtx 即可读取。
type wsStateKey string

const (
	ctxKeyAuthed    wsStateKey = "ws:authed"
	ctxKeyKicked    wsStateKey = "ws:kicked"
	ctxKeyKickToken wsStateKey = "ws:kickToken"
)

// Client represents a single WebSocket connection.
// The embedded context.Context carries userId, authed, and kicked state.
type Client struct {
	hub       HubInterface
	logger    logger.LoggerInterface
	conn      *websocket.Conn
	send      chan []byte
	done      chan struct{} // closed exactly once by Close; gates all sends
	mu        sync.RWMutex
	doneOnce  sync.Once // guards closing the done channel
	closeOnce sync.Once // guards closing the TCP connection
	context.Context
}

func NewClient(hub HubInterface, conn *websocket.Conn, logger logger.LoggerInterface) *Client {
	return &Client{
		hub:     hub,
		conn:    conn,
		send:    make(chan []byte, sendBufferSize),
		done:    make(chan struct{}),
		Context: context.Background(),
		logger:  logger,
	}
}

func (c *Client) UserID() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	id, _ := contextkeys.UserIDFromCtx(c.Context)
	return id
}

func (c *Client) SetUserID(userID uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Context = context.WithValue(contextkeys.WithUserID(c.Context, userID), ctxKeyAuthed, true)
}

func (c *Client) IsAuthed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, _ := c.Value(ctxKeyAuthed).(bool)
	return v
}

func (c *Client) IsKicked() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, _ := c.Value(ctxKeyKicked).(bool)
	return v
}

func (c *Client) KickToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	t, _ := c.Value(ctxKeyKickToken).(string)
	return t
}

func (c *Client) SetKickToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Context = context.WithValue(c.Context, ctxKeyKickToken, token)
}

// Close signals client shutdown. It closes the done channel exactly once.
// The send channel is deliberately NEVER closed: Send selects on done before
// writing, so a ping received by a still-running ReadPump after Hub.Stop can
// never panic on a send to a closed channel.
func (c *Client) Close() { c.doneOnce.Do(func() { close(c.done) }) }

func (c *Client) closeConn() { c.closeOnce.Do(func() { c.conn.Close() }) }

func (c *Client) Kick(reason string) {
	c.mu.Lock()
	c.Context = context.WithValue(c.Context, ctxKeyKicked, true)
	c.mu.Unlock()
	msg, _ := MarshalKicked(reason)
	c.Send(msg)
	time.AfterFunc(kickTimeout, func() {
		if c.IsKicked() {
			c.closeConn()
		}
	})
}

func (c *Client) Send(data []byte) {
	select {
	case <-c.done:
		return
	default:
	}
	select {
	case c.send <- data:
	case <-c.done:
	default:
		// buffer full — drop the message (backpressure)
		c.logger.Warn("ws client send buffer full, dropping message",
			zap.Uint64("userID", c.UserID()))
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.closeConn()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.logger.Warn("ws client read error", zap.Error(err))
			}
			break
		}
		if c.IsKicked() {
			continue
		}
		var msg ClientMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			c.logger.Warn("ws client invalid message", zap.Error(err))
			continue
		}
		switch msg.Type {
		case MsgTypeAuth:
			if c.IsAuthed() {
				continue
			}
			c.hub.handleAuth(c, msg.Token)
		case MsgTypePing:
			pong, _ := MarshalPong()
			c.Send(pong)
		default:
			c.logger.Warn("ws client unknown message type", zap.String("type", msg.Type))
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() { ticker.Stop(); c.closeConn() }()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.logger.Warn("ws client write error", zap.Error(err))
				return
			}
		case <-c.done:
			// Shutdown signalled by Close(): send a close frame and exit.
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
