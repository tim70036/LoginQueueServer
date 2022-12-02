package main

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

var (
	upgrader = websocket.Upgrader{}
	logger   = NewLogger().Sugar()
)

func main() {
	defer logger.Sync()

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

	e.GET("/ws", func(c echo.Context) error {
		// id := c.Response().Header().Get(echo.HeaderXRequestID)
		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		client := NewClient(conn)

		// Allow collection of memory referenced by the caller by doing all work in
		// new goroutines.
		go client.writePump()
		go client.readPump()

		return nil
	})

	s := http.Server{
		Addr:    ":8080",
		Handler: e,
		//ReadTimeout: 30 * time.Second, // customize http.Server timeouts
	}
	if err := s.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
