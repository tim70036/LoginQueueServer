package main

import (
	"encoding/json"
	"game-soul-technology/joker/joker-login-queue-server/pkg/msg"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type Application struct {
	config     *Config
	hub        *Hub
	queue      *Queue
	wsUpgrader *websocket.Upgrader
}

func ProvideApplication(config *Config, hub *Hub, queue *Queue) *Application {
	return &Application{
		config:     config,
		hub:        hub,
		queue:      queue,
		wsUpgrader: &websocket.Upgrader{},
	}
}

func (a *Application) Run() {
	go a.config.Run()
	go a.hub.Run()
	go a.queue.Run()
}

func (a *Application) HandleWs(c echo.Context) error {
	ticketId := c.Request().Header.Get("ticketId")
	conn, err := a.wsUpgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	// TODO: Extract jwt and ask main server if need to place client in queue.
	// Close connection right away if main server doesn't need to be in queue.
	// 1. queue is disabled
	// 2. client jwt's last heartbeat < 5 min or in game
	// 3. main server under maintenance
	if !a.config.IsQueueEnabled {
		rawEvent, err := json.Marshal(nil)
		if err != nil {
			logger.Errorf("cannot marshal NoQueueServerEvent %v", err)
		}
		wsMessage := &msg.WsMessage{
			EventCode: msg.NoQueueCode,
			EventData: rawEvent,
		}
		if err := conn.WriteJSON(wsMessage); err != nil {
			logger.Errorf("cannot write json to ws conn %v", err)
		}

		if err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "No need queue")); err != nil {
			logger.Errorf("cannot write close message to ws conn %v", err)
		}
		conn.Close()
		return nil
	}

	client := &Client{
		ticketId:      TicketId(ticketId),
		conn:          conn,
		sendWsMessage: make(chan *msg.WsMessage, 64),
		close:         make(chan []byte, 1),
		hub:           a.hub,
	}
	go client.Run()

	return nil
}
