package main

import (
	"context"
	"encoding/hex"
	"github.com/disgoorg/disgo/httpserver"
	"github.com/labstack/echo/v4"
	"log/slog"
)

type Server struct {
	*echo.Echo
	EventHandlerFunc httpserver.EventHandlerFunc
	publicKey        string
}

var _ httpserver.Server = (*Server)(nil)

func NewServer(publicKey string, eventHandlerFunc httpserver.EventHandlerFunc) Server {
	return Server{
		Echo:             echo.New(),
		EventHandlerFunc: eventHandlerFunc,
		publicKey:        publicKey,
	}
}

func (s *Server) Start() {
	hexKey, err := hex.DecodeString(s.publicKey)
	if err != nil {
		slog.Error("failed to decode public key",
			"error", err)
		panic(err)
	}
	handlerFunc := httpserver.HandleInteraction(
		hexKey,
		slog.Default(),
		s.EventHandlerFunc,
	)
	s.POST("/interactions", func(ctx echo.Context) error {
		handlerFunc(ctx.Response().Writer, ctx.Request())
		return nil
	})

	s.Logger.Fatal(s.Echo.Start(":8080"))
}

func (s *Server) Close(ctx context.Context) {
	if err := s.Echo.Shutdown(ctx); err != nil {
		slog.Error("failed to shutdown http server",
			"error", err)
	}
}

func startServer() {
	r := echo.New()
	r.GET("/", func(c echo.Context) error {
		return c.String(200, "Hello, World!")
	})

	r.Logger.Fatal(r.Start(":8080"))
}
