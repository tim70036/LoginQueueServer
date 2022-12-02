package main

import (
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Maximum message size allowed from peer.
	maxMessageSize = 8192

	// Send pings to peer with this period.
	pingPeriod = 5 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = pingPeriod * 5 / 2

	// Time to wait before force close on connection.
	closeGracePeriod = 10 * time.Second
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		conn: conn,
		send: make(chan []byte, 256),
	}
}

type TestMessage struct {
	EventId   int    `json:"eventId`
	EventData string `json:"eventData`
}

func (c *Client) readPump() {
	defer func() {
		logger.Infoln("leaving")
		c.conn.Close()
	}()

	// c.conn.SetReadLimit(maxMessageSize)

	// Heartbeat. Close connection if client does not respond to ping for too long.
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		logger.Info("pong")
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

		logger.Infoln("message: ", message)
		c.send <- []byte("received")
	}
}

func (c *Client) writePump() {
	pingTicker := time.NewTicker(pingPeriod)

	defer func() {
		logger.Infoln("leaving")
		pingTicker.Stop()
		c.conn.Close()
	}()

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
			logger.Infoln("ping")
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Errorln("Ping err:", err)
				return
			}
		}
	}
}
