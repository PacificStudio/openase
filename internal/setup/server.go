package setup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type ServerOptions struct {
	Host    string
	Port    int
	Service *Service
}

type Server struct {
	host    string
	port    int
	service *Service
	echo    *echo.Echo
}

func NewServer(opts ServerOptions) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.Recover())

	server := &Server{
		host:    opts.Host,
		port:    opts.Port,
		service: opts.Service,
		echo:    e,
	}
	server.registerRoutes()

	return server
}

func (s *Server) Handler() http.Handler {
	return s.echo
}

func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	httpServer := &http.Server{
		Addr:         net.JoinHostPort(s.host, strconv.Itoa(s.port)),
		Handler:      s.echo,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	go func() {
		if err := s.echo.StartServer(httpServer); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.echo.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown setup server: %w", err)
		}
		return <-errCh
	}
}

func (s *Server) registerRoutes() {
	s.echo.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusTemporaryRedirect, "/setup")
	})
	s.echo.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"service": "openase-setup",
			"status":  "ok",
		})
	})
	s.echo.GET("/setup", func(c echo.Context) error {
		return c.HTML(http.StatusOK, indexHTML)
	})
	s.echo.GET("/api/v1/setup/bootstrap", s.handleBootstrap)
	s.echo.POST("/api/v1/setup/test-database", s.handleTestDatabase)
	s.echo.POST("/api/v1/setup/complete", s.handleComplete)
}

func (s *Server) handleBootstrap(c echo.Context) error {
	payload, err := s.service.Bootstrap(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, payload)
}

func (s *Server) handleTestDatabase(c echo.Context) error {
	var request RawDatabaseInput
	if err := json.NewDecoder(c.Request().Body).Decode(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "request body must be valid JSON"})
	}

	payload, err := s.service.TestDatabase(c.Request().Context(), request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, payload)
}

func (s *Server) handleComplete(c echo.Context) error {
	var request RawCompleteRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "request body must be valid JSON"})
	}

	payload, err := s.service.Complete(c.Request().Context(), request)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrSetupAlreadyCompleted) {
			status = http.StatusConflict
		}
		return c.JSON(status, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, payload)
}
