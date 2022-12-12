package main

import (
	"encoding/json"
	"game-soul-technology/joker/joker-login-queue-server/pkg/client"
	"game-soul-technology/joker/joker-login-queue-server/pkg/config"
	"game-soul-technology/joker/joker-login-queue-server/pkg/msg"
	"game-soul-technology/joker/joker-login-queue-server/pkg/queue"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type Application struct {
	config        *config.Config
	clientFactory *client.ClientFactory
	hub           *client.Hub
	queue         *queue.Queue
	wsUpgrader    *websocket.Upgrader
}

func ProvideApplication(config *config.Config, clientFactory *client.ClientFactory, hub *client.Hub, queue *queue.Queue) *Application {
	return &Application{
		config:        config,
		clientFactory: clientFactory,
		hub:           hub,
		queue:         queue,
		wsUpgrader:    &websocket.Upgrader{},
	}
}

func (a *Application) Run() {
	go a.config.Run()
	go a.hub.Run()
	go a.queue.Run()
}

func (a *Application) HandleWs(c echo.Context) error {
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
		rawEvent, _ := json.Marshal(nil)
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

	client, err := a.clientFactory.Create(c, conn, a.hub)
	if err != nil {
		logger.Errorf("cannot create client %v", err)
		if err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseUnsupportedData, "")); err != nil {
			logger.Errorf("cannot write close message to ws conn %v", err)
		}
		conn.Close()
		return nil
	}

	go client.Run()

	return nil
}
