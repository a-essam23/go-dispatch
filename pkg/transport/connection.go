package transport

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

// callback executed when a message is received.
type MessageHandler func(ctx context.Context, connId uuid.UUID, msg []byte)

type OnCloseHandler func(connId uuid.UUID, err error)
type ConnectionConfig struct {
	ReadTimeout time.Duration
}

// Connection represents a single, thread-safe WebSocket connection.
type Connection struct {
	id     uuid.UUID
	conn   *websocket.Conn
	config ConnectionConfig
	send   chan []byte

	onMessage MessageHandler
	onClose   OnCloseHandler

	done      chan struct{}
	wg        *sync.WaitGroup
	ctx       context.Context
	closeOnce sync.Once
	cancel    context.CancelFunc

	logger *slog.Logger
}

func NewConnection(parentCtx context.Context, wg *sync.WaitGroup, conn *websocket.Conn, config ConnectionConfig, onMessage MessageHandler, onClose OnCloseHandler, logger *slog.Logger) *Connection {
	id := uuid.New()
	connCtx, cancel := context.WithCancel(parentCtx)
	connLogger := logger.With(slog.String("connID", id.String()))

	return &Connection{
		id:        id,
		conn:      conn,
		logger:    connLogger,
		config:    config,
		onMessage: onMessage,
		send:      make(chan []byte, 256), // Buffered channel
		done:      make(chan struct{}),
		ctx:       connCtx,
		cancel:    cancel,
		onClose:   onClose,
		wg:        wg,
	}
}

func (c *Connection) Run() {
	c.wg.Add(1)
	go c.readPump()
	go c.writePump()

	c.logger.Info("connection established")
}

// readPump pumps messages from the WebSocket connection to the message handler.
func (c *Connection) readPump() {
	var readErr error
	defer func() {
		c.Close(readErr)
	}()

	for {
		readCtx, cancelRead := context.WithTimeout(c.ctx, c.config.ReadTimeout)
		defer cancelRead()
		typ, r, err := c.conn.Reader(readCtx)
		if err != nil {
			readErr = err
			cancelRead()
			return
		}
		// Ensure we are only handling text or binary messages.
		if typ != websocket.MessageText && typ != websocket.MessageBinary {
			continue
		}
		// Pass a connection-scoped context to the handler.
		// Read the full message. Use io.ReadAll for safety.
		message, err := io.ReadAll(r)
		if err != nil {
			c.logger.Error("Connection readpump failed for some reason")
			readErr = err

			return
		}
		c.onMessage(c.ctx, c.id, message)
	}
}

// writePump pumps messages from the send channel to the WebSocket connection.
func (c *Connection) writePump() {
	var writeErr error

	defer func() {
		c.Close(writeErr)
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// The send channel was closed, signal clean closure.
				c.conn.Close(websocket.StatusNormalClosure, "")
				return
			}
			if err := c.conn.Write(c.ctx, websocket.MessageText, message); err != nil {
				writeErr = err
				return
			}
		case <-c.ctx.Done():
			c.conn.Close(websocket.StatusNormalClosure, "request cancelled")
			return
		}
	}
}

// sends a message to the client. It is safe for concurrent use.
func (c *Connection) Send(message []byte) {
	select {
	case c.send <- message:
	case <-c.ctx.Done():
		c.logger.Warn("Attempted to send on a closed connection")
	}
}

// gracefully shuts down the connection and its resources.
func (c *Connection) Close(err error) {
	c.closeOnce.Do(func() {
		status := websocket.CloseStatus(err)
		c.logger.Info("Transport connection closing", slog.Any("reason", err), slog.String("status", status.String()))

		c.cancel() // Signal goroutines to stop.
		close(c.send)
		c.conn.Close(websocket.StatusNormalClosure, "")
		c.logger.Info("Connection closed")
		if c.onClose != nil {
			c.onClose(c.id, err)
		}
		c.wg.Done()
		close(c.done)
	})
}

// returns a channel that is closed when the connection is fully terminated.
func (c *Connection) Done() <-chan struct{} {
	return c.done
}

// ID returns the unique identifier of the connection.
func (c *Connection) ID() uuid.UUID {
	return c.id
}

func (c *Connection) SetOnMessageHandler(handler MessageHandler) {
	c.onMessage = handler
}
func (c *Connection) SetOnCloseHandler(handler OnCloseHandler) {
	c.onClose = handler
}
