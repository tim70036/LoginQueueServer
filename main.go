package main

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"go.uber.org/zap/zapcore"
)

func main() {
	defer logger.Sync()

	// TODO: DI
	upgrader := websocket.Upgrader{}
	go config.Run()
	go hub.Run()
	go queue.Run()

	// TODO: remove this
	loggerLevel.SetLevel(zapcore.DebugLevel)

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
			logger.Infof("%v %v id[%v] status[%v] latency[%vms]\n", v.Method, v.URI, v.RequestID, v.Status, v.Latency.Milliseconds())
			return nil
		},
	}))

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!\n")
	})

	e.PUT("/debug", func(c echo.Context) error {
		loggerLevel.SetLevel(zapcore.DebugLevel)
		logger.Info("debug logging enabled")
		return c.NoContent(http.StatusOK)
	})

	e.DELETE("/debug", func(c echo.Context) error {
		loggerLevel.SetLevel(zapcore.InfoLevel)
		logger.Info("debug logging disabled")
		return c.NoContent(http.StatusOK)
	})

	e.GET("/ws", func(c echo.Context) error {
		ticketId := c.Request().Header.Get("ticketId")
		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}

		// TODO: Extract jwt and ask main server if need to place client in queue.
		// Close connection right away if main server doesn't need to be in queue.
		// 1. queue is disabled
		// 2. main server online user number < threshold
		// 3. client jwt's last heartbeat < 5 min or in game

		client := NewClient(TicketId(ticketId), conn)
		go client.Run()

		return nil
	})

	server := http.Server{
		Addr:    ":8080",
		Handler: e,
		//ReadTimeout: 30 * time.Second, // customize http.Server timeouts
	}
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
