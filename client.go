package main

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Maximum message size allowed from peer.
	maxMessageSize = 8192

	// Send pings to peer with this period.
	pingInterval = 30 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = pingInterval * 5 / 2

	// Time to wait before force close on connection.
	closeGracePeriod = 10 * time.Second
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	ticketId string

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte // TODO: string?

	cleanupOnce sync.Once
}

func NewClient(ticketId string, conn *websocket.Conn) *Client {
	return &Client{
		ticketId: ticketId,
		conn:     conn,
		send:     make(chan []byte, 64),
	}
}

// WebSocket connections support one concurrent reader and one
// concurrent writer. The application ensures that these concurrency
// requirements are met by executing all reads from the readPump
// goroutine and all writes from the writePump goroutine.
func (c *Client) Run() {
	go c.readPump()
	go c.writePump()
	hub.register <- c
}

// Try cleanup before the client is closed. Note that cleanup action
// will only be performed once to avoid undesired side effect.
func (c *Client) tryCleanup() {
	c.cleanupOnce.Do(func() {
		logger.Debugf("cleanup id[%v]", c.ticketId)
		c.conn.Close()
		hub.unregister <- c
	})
}

type TestMessage struct {
	EventId   int    `json:"eventId`
	EventData string `json:"eventData`
}

func (c *Client) readPump() {
	defer c.tryCleanup()

	// c.conn.SetReadLimit(maxMessageSize)

	// Heartbeat. Set read timeout if client does not respond to ping
	// for too long. This will in turn make conn.ReadMessage get an io
	// timeout error and thus closing the connection.
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		logger.Debugf("receive pong id[%v]", c.ticketId)
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage() // TODO: Read json
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Errorln("error: ", err)
			} else {
				logger.Infoln("read closing: ", err)
			}
			break
		}

		logger.Infof("id[%v] message[%v]", c.ticketId, message)
		c.send <- []byte("received")
	}
}

func (c *Client) writePump() {
	pingTicker := time.NewTicker(pingInterval)

	defer pingTicker.Stop()
	defer c.tryCleanup()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			shit := &TestMessage{
				EventId:   1,
				EventData: string(message),
			}
			if err := c.conn.WriteJSON(shit); err != nil {
				logger.Errorln("WriteJSON err:", err)
				return
			}
		case <-pingTicker.C:
			logger.Debugf("send ping id[%v]", c.ticketId)
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Errorln("Ping err:", err)
				return
			}
		}
	}
}
