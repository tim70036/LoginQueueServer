package main

import (
	"encoding/json"
	. "game-soul-technology/joker/joker-login-queue-server/pkg/infra"
	"game-soul-technology/joker/joker-login-queue-server/pkg/msg"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"go.uber.org/zap/zapcore"
)

var (
	wsUpgrader = websocket.Upgrader{}
)

func main() {
	defer Logger.Sync()

	// TODO: remove this
	LoggerLevel.SetLevel(zapcore.DebugLevel)

	// TODO: DI
	go cfg.Run()
	go hub.Run()
	go queue.Run()

	e := echo.New()
	e.Use(middleware.RequestID())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogLatency:   true,
		LogMethod:    true,
		LogURI:       true,
		LogRequestID: true,
		LogStatus:    true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			Logger.Infof("%v %v id[%v] status[%v] latency[%vms]\n", v.Method, v.URI, v.RequestID, v.Status, v.Latency.Milliseconds())
			return nil
		},
	}))

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!\n")
	})

	e.PUT("/debug", func(c echo.Context) error {
		LoggerLevel.SetLevel(zapcore.DebugLevel)
		Logger.Info("debug logging enabled")
		return c.NoContent(http.StatusOK)
	})

	e.DELETE("/debug", func(c echo.Context) error {
		LoggerLevel.SetLevel(zapcore.InfoLevel)
		Logger.Info("debug logging disabled")
		return c.NoContent(http.StatusOK)
	})

	e.GET("/ws", handleWs)

	server := http.Server{
		Addr:    ":8080",
		Handler: e,
		//ReadTimeout: 30 * time.Second, // customize http.Server timeouts
	}
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func handleWs(c echo.Context) error {
	ticketId := c.Request().Header.Get("ticketId")
	conn, err := wsUpgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	// TODO: Extract jwt and ask main server if need to place client in queue.
	// Close connection right away if main server doesn't need to be in queue.
	// 1. queue is disabled
	// 2. client jwt's last heartbeat < 5 min or in game
	// 3. main server under maintenance
	if !cfg.IsQueueEnabled {
		rawEvent, err := json.Marshal(nil)
		if err != nil {
			Logger.Errorf("cannot marshal NoQueueServerEvent %v", err)
		}
		wsMessage := &msg.WsMessage{
			EventCode: msg.NoQueueCode,
			EventData: rawEvent,
		}
		if err := conn.WriteJSON(wsMessage); err != nil {
			Logger.Errorf("cannot write json to ws conn %v", err)
		}

		if err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "No need queue")); err != nil {
			Logger.Errorf("cannot write close message to ws conn %v", err)
		}
		conn.Close()
		return nil
	}

	client := NewClient(TicketId(ticketId), conn)
	go client.Run()
	return nil
}
