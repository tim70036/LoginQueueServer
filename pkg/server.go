package main

import (
	"fmt"
	"game-soul-technology/joker/joker-login-queue-server/pkg/infra"
	"net/http"
	"os"

	"github.com/imroc/req/v3"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Server struct {
	application *Application
	server      *http.Server
	logger      *zap.SugaredLogger
}

func ProvideServer(application *Application, httpClient *req.Client, loggerFactory *infra.LoggerFactory) *Server {
	logger := loggerFactory.Create("Server").Sugar()

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
		httpClient.EnableDumpAll()
		logger.Info("debug logging enabled")
		return c.NoContent(http.StatusOK)
	})

	e.DELETE("/debug", func(c echo.Context) error {
		infra.LoggerLevel.SetLevel(zapcore.InfoLevel)
		httpClient.DisableDumpAll()
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
		logger: logger,
	}
}

func (s *Server) Run() {
	s.logger.Infof("server running application")
	go s.application.Run()

	s.logger.Infof("server starts listening on port[%v]", os.Getenv("SERVER_PORT"))
	if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
		s.logger.Error(err)
	}
}
