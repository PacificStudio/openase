package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	"github.com/BetterAndBetterII/openase/internal/infra/sse"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/BetterAndBetterII/openase/internal/webui"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	cfg    config.ServerConfig
	logger *slog.Logger
	echo   *echo.Echo
	sseHub *sse.Hub
}

func NewServer(cfg config.ServerConfig, logger *slog.Logger, events provider.EventProvider) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus: true,
		LogURI:    true,
		LogMethod: true,
		LogError:  true,
		LogValuesFunc: func(c echo.Context, values middleware.RequestLoggerValues) error {
			log := logger.With(
				"method", values.Method,
				"uri", values.URI,
				"status", values.Status,
				"request_id", c.Response().Header().Get(echo.HeaderXRequestID),
			)
			if values.Error != nil || values.Status >= http.StatusInternalServerError {
				log.Error("http request completed", "error", values.Error)
				return nil
			}
			log.Info("http request completed")
			return nil
		},
	}))

	server := &Server{
		cfg:    cfg,
		logger: logger.With("component", "http-server"),
		echo:   e,
		sseHub: sse.NewHub(events, logger),
	}
	server.registerRoutes()

	return server
}

func (s *Server) Handler() http.Handler {
	return s.echo
}

func (s *Server) Run(ctx context.Context) error {
	defer func() {
		if err := s.sseHub.Close(); err != nil {
			s.logger.Error("close sse hub", "error", err)
		}
	}()

	errCh := make(chan error, 1)
	httpServer := &http.Server{
		Addr:         net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.Port)),
		Handler:      s.echo,
		ReadTimeout:  s.cfg.ReadTimeout,
		WriteTimeout: s.cfg.WriteTimeout,
		IdleTimeout:  s.cfg.WriteTimeout,
	}

	go func() {
		s.logger.Info("http server starting", "address", httpServer.Addr)
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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
		defer cancel()

		s.logger.Info("http server stopping")
		if err := s.echo.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}

		return <-errCh
	}
}

func (s *Server) registerRoutes() {
	healthHandler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"service": "openase",
			"status":  "ok",
			"time":    time.Now().UTC().Format(time.RFC3339),
			"port":    strconv.Itoa(s.cfg.Port),
		})
	}

	s.echo.GET("/healthz", healthHandler)
	s.echo.GET("/api/v1/healthz", healthHandler)
	s.echo.GET("/api/v1/events/stream", s.handleEventStream)

	uiHandler := echo.WrapHandler(webui.Handler())
	s.echo.GET("/", uiHandler)
	s.echo.GET("/*", uiHandler)
}
