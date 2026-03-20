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

	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	chatservice "github.com/BetterAndBetterII/openase/internal/chat"
	"github.com/BetterAndBetterII/openase/internal/config"
	"github.com/BetterAndBetterII/openase/internal/infra/sse"
	notificationservice "github.com/BetterAndBetterII/openase/internal/notification"
	"github.com/BetterAndBetterII/openase/internal/provider"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/BetterAndBetterII/openase/internal/webui"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	cfg                 config.ServerConfig
	github              config.GitHubConfig
	logger              *slog.Logger
	events              provider.EventProvider
	echo                *echo.Echo
	sseHub              *sse.Hub
	inboundWebhooks     *inboundWebhookReceiver
	ticketService       *ticketservice.Service
	ticketStatusService *ticketstatus.Service
	agentPlatform       *agentplatform.Service
	catalog             catalogservice.Service
	workflowService     *workflowservice.Service
	notificationService *notificationservice.Service
	chatService         *chatservice.Service
}

type ServerOption func(*Server)

func WithNotificationService(service *notificationservice.Service) ServerOption {
	return func(server *Server) {
		server.notificationService = service
	}
}

func WithChatService(service *chatservice.Service) ServerOption {
	return func(server *Server) {
		server.chatService = service
	}
}

func NewServer(
	cfg config.ServerConfig,
	github config.GitHubConfig,
	logger *slog.Logger,
	events provider.EventProvider,
	ticketService *ticketservice.Service,
	ticketStatusService *ticketstatus.Service,
	agentPlatform *agentplatform.Service,
	catalog catalogservice.Service,
	workflowService *workflowservice.Service,
	opts ...ServerOption,
) *Server {
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
		cfg:                 cfg,
		github:              github,
		logger:              logger.With("component", "http-server"),
		events:              events,
		echo:                e,
		sseHub:              sse.NewHub(events, logger),
		ticketService:       ticketService,
		ticketStatusService: ticketStatusService,
		agentPlatform:       agentPlatform,
		catalog:             catalog,
		workflowService:     workflowService,
	}
	server.inboundWebhooks = newInboundWebhookReceiver(server.logger, newGitHubRepoScopeWebhookEndpoint(server))
	for _, opt := range opts {
		if opt != nil {
			opt(server)
		}
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

	api := s.echo.Group("/api/v1")
	api.GET("/healthz", healthHandler)
	api.GET("/openapi.json", s.handleOpenAPI)
	api.GET("/events/stream", s.handleEventStream)
	api.POST("/webhooks/github", s.handleLegacyGitHubWebhook)
	api.POST("/webhooks/:connector/:provider", s.handleInboundWebhook)
	api.GET("/projects/:projectId/tickets/stream", s.handleTicketStream)
	api.GET("/projects/:projectId/agents/stream", s.handleAgentStream)
	api.GET("/projects/:projectId/hooks/stream", s.handleHookStream)
	api.GET("/projects/:projectId/activity/stream", s.handleActivityStream)
	if s.agentPlatform != nil {
		s.registerAgentPlatformRoutes(api.Group("/platform", s.authenticateAgentToken))
	}
	if s.catalog != nil {
		s.registerCatalogRoutes(api)
	}
	s.registerTicketRoutes(api)
	s.registerChatRoutes(api)
	s.registerWorkflowRoutes(api)
	s.registerNotificationRoutes(api)
	s.registerSkillRoutes(api)
	s.registerRoleLibraryRoutes(api)
	s.registerHRAdvisorRoutes(api)
	s.registerTicketStatusRoutes()

	uiHandler := echo.WrapHandler(webui.Handler())
	s.echo.GET("/", uiHandler)
	s.echo.GET("/*", uiHandler)
}
