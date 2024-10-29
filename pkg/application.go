package main

import (
	"encoding/json"
	"game-soul-technology/joker/joker-login-queue-server/pkg/client"
	"game-soul-technology/joker/joker-login-queue-server/pkg/config"
	"game-soul-technology/joker/joker-login-queue-server/pkg/infra"
	"game-soul-technology/joker/joker-login-queue-server/pkg/msg"
	"game-soul-technology/joker/joker-login-queue-server/pkg/queue"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/imroc/req/v3"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type Application struct {
	config        *config.Config
	queueConfig   *config.QueueConfig
	clientFactory *client.ClientFactory
	hub           *client.Hub
	queue         *queue.Queue
	wsUpgrader    *websocket.Upgrader
	httpClient    *req.Client
	logger        *zap.SugaredLogger
}

func ProvideApplication(config *config.Config, queueConfig *config.QueueConfig, clientFactory *client.ClientFactory, hub *client.Hub, queue *queue.Queue, httpClient *req.Client, loggerFactory *infra.LoggerFactory) *Application {
	return &Application{
		config:        config,
		queueConfig:   queueConfig,
		clientFactory: clientFactory,
		hub:           hub,
		queue:         queue,
		wsUpgrader:    &websocket.Upgrader{},
		httpClient:    httpClient,
		logger:        loggerFactory.Create("Application").Sugar(),
	}
}

func (a *Application) Run() {
	go a.queueConfig.Run()
	go a.hub.Run()
	go a.queue.Run()
}

func (a *Application) HandleWs(c echo.Context) error {
	conn, err := a.wsUpgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	// Close connection right away if this client doesn't need to be
	// in queue.
	// 1. queue is disabled
	// 2. queue is enabled, but current online users has not reach threshold.
	// 2. client jwt's last heartbeat < 5 min or is in game
	// 3. main server under maintenance
	if !a.queueConfig.ShouldQueue() {
		a.rejectWs(conn, websocket.CloseNormalClosure, "No need queue", true)
		return nil
	}

	jwt := c.Request().Header.Get("jwt")
	if needQueue := a.sessionNeedQueue(jwt); !needQueue {
		a.rejectWs(conn, websocket.CloseNormalClosure, "No need queue", true)
		return nil
	}

	client, err := a.clientFactory.Create(c, conn)
	if err != nil {
		a.logger.Errorf("cannot create client %v", err)
		a.rejectWs(conn, websocket.CloseUnsupportedData, err.Error(), false)
		return nil
	}

	go client.Run()

	return nil
}

func (a *Application) rejectWs(conn *websocket.Conn, closeCode int, closeReason string, shouldSendEvent bool) {
	if shouldSendEvent {
		rawEvent, err := json.Marshal(&msg.ShouldQueueEvent{
			ShouldQueue: false,
		})
		if err != nil {
			a.logger.Errorf("cannot marshal ShouldQueueEvent %v", err)
			return
		}

		wsMessage := &msg.WsMessage{
			EventCode: msg.ShouldQueueCode,
			EventData: rawEvent,
		}
		if err := conn.WriteJSON(wsMessage); err != nil {
			a.logger.Errorf("cannot write json to ws conn %v", err)
		}
	}

	time.Sleep(client.CloseGracePeriod) // Ensure that other message is sent.
	if err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(closeCode, closeReason)); err != nil {
		a.logger.Debugf("cannot write close message to ws conn %v", err)
	}

	time.Sleep(client.CloseGracePeriod)
	conn.Close() // Ensure that close message is sent.
}

func (a *Application) sessionNeedQueue(jwt string) bool {

	roomSessionResult := &struct {
		Data struct {
			IsInRoom bool   `json:"isInRoom"`
			RoomId   string `json:"roomId"`
		} `json:"data"`
	}{}

	resp, err := a.httpClient.R().
		SetHeader("jwt", jwt).
		SetResult(roomSessionResult).
		Get(os.Getenv("MAIN_SERVER_HOST") + "/api/room/session")

	if err != nil {
		a.logger.Errorf("request failed %v", err)
		return false
	}

	if resp.StatusCode == 503 {
		a.logger.Debugf("no need que main server under maintenance")
		return false
	}

	if resp.IsSuccess() && roomSessionResult.Data.IsInRoom {
		a.logger.Debugf("no need que since client has roomSessionResult[%+v]", roomSessionResult)
		return false
	}

	userSessionResult := &struct {
		Data struct {
			Uid             string `json:"uid"`
			Jwt             string `json:"jwt"`
			CreateTime      string `json:"createTime"`
			LastHeartbeatIP string `json:"lastHeartbeatIP"`
			LastHeartbeat   string `json:"lastHeartbeat"`
		} `json:"data"`
	}{}

	resp, err = a.httpClient.R().
		SetHeader("jwt", jwt).
		SetResult(userSessionResult).
		Get(os.Getenv("MAIN_SERVER_HOST") + "/api/user/session")

	if err != nil {
		a.logger.Errorf("request failed %v", err)
		return false
	}

	if resp.StatusCode == 503 {
		a.logger.Debugf("no need que main server under maintenance")
		return false
	}

	if resp.IsSuccess() {
		lastHeartbeatTime, err := time.Parse(time.RFC3339, userSessionResult.Data.LastHeartbeat)
		if err != nil {
			a.logger.Errorf("cannot parse lastHeartbeatTime from userSessionResult[%v] %v", userSessionResult, err)
			return true
		}

		// A client will receive main server session after he finishes login
		// queue. He then can use this session to do anything he wants on
		// main server. However, he has to stay online. If he goes offline
		// over a period of time, he has to go into login queue again.
		// This constant controls the time period.
		if time.Since(lastHeartbeatTime) < time.Duration(*a.config.SessionStaleSeconds)*time.Second {
			a.logger.Debugf("no need que since client has userSessionResult[%+v]", userSessionResult)
			return false
		}
	}

	return true
}
