package main

import (
	"fmt"
	"game-soul-technology/joker/joker-login-queue-server/pkg/infra"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap/zapcore"
)

type Server struct {
	application *Application
	server      *http.Server
}

func ProvideServer(application *Application) *Server {
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
		infra.LoggerLevel.SetLevel(zapcore.DebugLevel)
		logger.Info("debug logging enabled")
		return c.NoContent(http.StatusOK)
	})

	e.DELETE("/debug", func(c echo.Context) error {
		infra.LoggerLevel.SetLevel(zapcore.InfoLevel)
		logger.Info("debug logging disabled")
		return c.NoContent(http.StatusOK)
	})

	e.GET("/ws", application.HandleWs)

	return &Server{
		application: application,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%v", os.Getenv("SERVER_PORT")),
			Handler: e,
			//ReadTimeout: 30 * time.Second, // customize http.Server timeouts
		},
	}
}

func (s *Server) Run() {
	logger.Infof("server running application")
	go s.application.Run()

	logger.Infof("server starts listening on port[%v]", os.Getenv("SERVER_PORT"))
	if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
		logger.Error(err)
	}
}
