package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	chatservice "github.com/BetterAndBetterII/openase/internal/chat"
	"github.com/BetterAndBetterII/openase/internal/config"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	humanauthdomain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	machinetransport "github.com/BetterAndBetterII/openase/internal/infra/machinetransport"
	"github.com/BetterAndBetterII/openase/internal/infra/sse"
	machinechannelservice "github.com/BetterAndBetterII/openase/internal/machinechannel"
	notificationservice "github.com/BetterAndBetterII/openase/internal/notification"
	projectupdateservice "github.com/BetterAndBetterII/openase/internal/projectupdate"
	"github.com/BetterAndBetterII/openase/internal/provider"
	runtimeobservability "github.com/BetterAndBetterII/openase/internal/runtime/observability"
	scheduledjobservice "github.com/BetterAndBetterII/openase/internal/scheduledjob"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	githubreposervice "github.com/BetterAndBetterII/openase/internal/service/githubrepo"
	humanauthservice "github.com/BetterAndBetterII/openase/internal/service/humanauth"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	cfg                        config.ServerConfig
	auth                       config.AuthConfig
	github                     config.GitHubConfig
	logger                     *slog.Logger
	events                     provider.EventProvider
	trace                      provider.TraceProvider
	metrics                    provider.MetricsProvider
	metricsHandler             http.Handler
	echo                       *echo.Echo
	sseHub                     *sse.Hub
	activityEmitter            *activitysvc.Emitter
	ticketService              *ticketservice.Service
	ticketStatusService        *ticketstatus.Service
	agentPlatform              *agentplatform.Service
	catalog                    catalogservice.Services
	workflowService            *workflowservice.Service
	scheduledJobService        *scheduledjobservice.Service
	notificationService        *notificationservice.Service
	projectUpdateService       *projectupdateservice.Service
	chatService                *chatservice.Service
	projectConversationService *chatservice.ProjectConversationService
	githubAuthService          githubauthservice.SecurityManager
	githubRepoService          githubreposervice.Service
	humanAuthService           *humanauthservice.Service
	humanAuthorizer            *humanauthservice.Authorizer
	memoryCollector            runtimeobservability.ProcessMemoryCollector
	ticketWorkspaceResetter    ticketWorkspaceResetter
	machineChannel             *machinechannelservice.Service
	machineSessions            *machinechannelservice.SessionRegistry
	reverseRuntimeRelay        *machinetransport.ReverseRuntimeRelayRegistry
	shutdownCtx                context.Context
	shutdownCancel             context.CancelFunc
	shutdownOnce               sync.Once
	connMu                     sync.Mutex
	activeConns                map[net.Conn]struct{}
}

type ticketWorkspaceResetter interface {
	ResetTicketWorkspace(ctx context.Context, ticketID uuid.UUID) error
}

type ServerOption func(*Server)

func WithNotificationService(service *notificationservice.Service) ServerOption {
	return func(server *Server) {
		server.notificationService = service
	}
}

func WithProjectUpdateService(service *projectupdateservice.Service) ServerOption {
	return func(server *Server) {
		server.projectUpdateService = service
	}
}

func WithChatService(service *chatservice.Service) ServerOption {
	return func(server *Server) {
		server.chatService = service
	}
}

func WithProjectConversationService(service *chatservice.ProjectConversationService) ServerOption {
	return func(server *Server) {
		server.projectConversationService = service
	}
}

func WithGitHubAuthService(service githubauthservice.SecurityManager) ServerOption {
	return func(server *Server) {
		server.githubAuthService = service
	}
}

func WithGitHubRepoService(service githubreposervice.Service) ServerOption {
	return func(server *Server) {
		server.githubRepoService = service
	}
}

func WithHumanAuthConfig(cfg config.AuthConfig) ServerOption {
	return func(server *Server) {
		server.auth = cfg
	}
}

func WithHumanAuthService(service *humanauthservice.Service, authorizer *humanauthservice.Authorizer) ServerOption {
	return func(server *Server) {
		server.humanAuthService = service
		server.humanAuthorizer = authorizer
	}
}

func WithTicketWorkspaceResetter(service ticketWorkspaceResetter) ServerOption {
	return func(server *Server) {
		server.ticketWorkspaceResetter = service
	}
}

func WithTraceProvider(trace provider.TraceProvider) ServerOption {
	return func(server *Server) {
		server.trace = trace
	}
}

func WithScheduledJobService(service *scheduledjobservice.Service) ServerOption {
	return func(server *Server) {
		server.scheduledJobService = service
	}
}

func WithMetricsProvider(metrics provider.MetricsProvider) ServerOption {
	return func(server *Server) {
		server.metrics = metrics
	}
}

func WithMetricsHandler(handler http.Handler) ServerOption {
	return func(server *Server) {
		server.metricsHandler = handler
	}
}

func WithProcessMemoryCollector(collector runtimeobservability.ProcessMemoryCollector) ServerOption {
	return func(server *Server) {
		server.memoryCollector = collector
	}
}

func WithMachineChannel(service *machinechannelservice.Service, sessions *machinechannelservice.SessionRegistry) ServerOption {
	return func(server *Server) {
		server.machineChannel = service
		server.machineSessions = sessions
	}
}

func WithReverseRuntimeRelay(relay *machinetransport.ReverseRuntimeRelayRegistry) ServerOption {
	return func(server *Server) {
		server.reverseRuntimeRelay = relay
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
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) &&
			strings.HasPrefix(c.Request().URL.Path, "/api/") &&
			(httpErr.Code == http.StatusNotFound || httpErr.Code == http.StatusMethodNotAllowed) {
			_ = c.String(httpErr.Code, http.StatusText(httpErr.Code))
			return
		}
		e.DefaultHTTPErrorHandler(err, c)
	}
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())

	//nolint:gosec // stored on Server and invoked during beginShutdown
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())

	server := &Server{
		cfg:                 cfg,
		auth:                config.AuthConfig{Mode: config.AuthModeDisabled},
		github:              github,
		logger:              logger.With("component", "http-server"),
		events:              events,
		metrics:             provider.NewNoopMetricsProvider(),
		echo:                e,
		sseHub:              sse.NewHub(events, logger),
		ticketService:       ticketService,
		ticketStatusService: ticketStatusService,
		agentPlatform:       agentPlatform,
		catalog:             catalogservice.SplitServices(catalog),
		workflowService:     workflowService,
		memoryCollector:     runtimeobservability.RuntimeProcessMemoryCollector{},
		shutdownCtx:         shutdownCtx,
		shutdownCancel:      shutdownCancel,
		activeConns:         make(map[net.Conn]struct{}),
	}
	if ticketService != nil {
		server.activityEmitter = activitysvc.NewEmitter(activitysvc.RecordFunc(func(ctx context.Context, input activitysvc.RecordInput) (catalogdomain.ActivityEvent, error) {
			return ticketService.RecordActivityEvent(ctx, ticketservice.RecordActivityEventInput{
				ProjectID: input.ProjectID,
				TicketID:  input.TicketID,
				AgentID:   input.AgentID,
				EventType: input.EventType,
				Message:   input.Message,
				Metadata:  input.Metadata,
				CreatedAt: input.CreatedAt,
			})
		}), events)
	}
	for _, opt := range opts {
		if opt != nil {
			opt(server)
		}
	}
	e.Use(server.injectRequestLogger())
	e.Use(server.traceRequest())
	e.Use(server.metricsMiddleware())
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
				"trace_id", traceValue(c, traceIDContextKey),
				"span_id", traceValue(c, spanIDContextKey),
			)
			if values.Error != nil || values.Status >= http.StatusInternalServerError {
				log.Error("http request completed", "error", values.Error)
				return nil
			}
			log.Info("http request completed")
			return nil
		},
	}))
	registerServerRoutes(server)

	return server
}

type authzEvaluation struct {
	scope       humanauthdomain.ScopeRef
	permission  humanauthdomain.PermissionKey
	roles       []humanauthdomain.RoleKey
	permissions []humanauthdomain.PermissionKey
}

func (s *Server) Handler() http.Handler {
	return s.echo
}

func (s *Server) shutdownAwareContext(parent context.Context) (context.Context, context.CancelFunc) {
	if s == nil || s.shutdownCtx == nil {
		//nolint:gosec // returned to the caller for request lifecycle control
		ctx, cancel := context.WithCancel(parent)
		return ctx, cancel
	}

	ctx, cancel := context.WithCancel(parent)
	stop := context.AfterFunc(s.shutdownCtx, cancel)
	return ctx, func() {
		stop()
		cancel()
	}
}

// beginShutdown prefers bounded process exit over preserving long-lived streams.
// OpenASE actively cancels streaming handlers and reverse websocket sessions so
// clients reconnect after restart instead of holding shutdown open indefinitely.
func (s *Server) beginShutdown() {
	if s == nil {
		return
	}

	s.shutdownOnce.Do(func() {
		if s.shutdownCancel != nil {
			s.shutdownCancel()
		}
		if s.sseHub != nil {
			if err := s.sseHub.Close(); err != nil {
				s.logger.Error("close sse hub", "error", err)
			}
		}
		if s.machineSessions == nil {
			s.closeActiveConnections()
			return
		}

		disconnectedAt := time.Now().UTC()
		for _, session := range s.machineSessions.CloseAll("server shutting down; reconnect after restart") {
			if s.machineChannel != nil {
				_, _ = s.machineChannel.RecordDisconnectedSession(context.Background(), machinechannelservice.DisconnectedSessionRecord{
					MachineID:      session.MachineID,
					SessionID:      session.SessionID,
					DisconnectedAt: disconnectedAt,
					Reason:         "server_shutdown",
				})
			}
			s.publishMachineChannelDisconnect(context.Background(), session.MachineID, session.SessionID, "server_shutdown")
		}
		s.closeActiveConnections()
	})
}

func (s *Server) trackConnectionState(conn net.Conn, state http.ConnState) {
	if s == nil || conn == nil {
		return
	}

	s.connMu.Lock()
	defer s.connMu.Unlock()

	switch state {
	case http.StateNew, http.StateActive, http.StateHijacked:
		s.activeConns[conn] = struct{}{}
	case http.StateIdle, http.StateClosed:
		delete(s.activeConns, conn)
	}
}

func (s *Server) closeActiveConnections() {
	if s == nil {
		return
	}

	s.connMu.Lock()
	connections := make([]net.Conn, 0, len(s.activeConns))
	for conn := range s.activeConns {
		connections = append(connections, conn)
	}
	s.connMu.Unlock()

	for _, conn := range connections {
		_ = conn.Close()
	}
}

func (s *Server) Run(ctx context.Context) error {
	defer s.beginShutdown()
	if s.machineChannel != nil && s.machineSessions != nil {
		//nolint:gosec // server lifecycle goroutine is intentionally tied to process-scoped ctx
		go s.runMachineSessionExpiryLoop(ctx)
	}

	errCh := make(chan error, 1)
	httpServer := &http.Server{
		Addr:         net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.Port)),
		Handler:      s.echo,
		ReadTimeout:  s.cfg.ReadTimeout,
		WriteTimeout: s.cfg.WriteTimeout,
		IdleTimeout:  s.cfg.WriteTimeout,
		ConnState:    s.trackConnectionState,
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
		s.beginShutdown()

		// StartServer serves the custom httpServer instance below, so shutdown must
		// target that same server to close its listener and active connections.
		shutdownErrCh := make(chan error, 1)
		go func() {
			shutdownErrCh <- httpServer.Shutdown(shutdownCtx)
		}()

		select {
		case err := <-shutdownErrCh:
			if err != nil {
				return fmt.Errorf("shutdown http server: %w", err)
			}
		case <-shutdownCtx.Done():
			s.logger.Warn("http server shutdown timed out; force closing active connections")
			if err := httpServer.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("force close http server: %w", err)
			}
			if err := <-shutdownErrCh; err != nil &&
				!errors.Is(err, context.DeadlineExceeded) &&
				!errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("shutdown http server after force close: %w", err)
			}
		}

		return <-errCh
	}
}

func (s *Server) metricsMiddleware() echo.MiddlewareFunc {
	var inFlight int64

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Path() == "/api/v1/system/metrics" {
				return next(c)
			}

			currentInFlight := atomic.AddInt64(&inFlight, 1)
			s.metrics.Gauge("openase.http.server.in_flight_requests", provider.Tags{
				"server": "http",
			}).Set(float64(currentInFlight))
			start := time.Now()

			err := next(c)

			status := c.Response().Status
			if status == 0 {
				if err != nil {
					status = http.StatusInternalServerError
				} else {
					status = http.StatusOK
				}
			}

			route := c.Path()
			if route == "" {
				route = "unmatched"
			}

			tags := provider.Tags{
				"method": c.Request().Method,
				"route":  route,
				"status": strconv.Itoa(status),
			}
			s.metrics.Counter("openase.http.server.requests_total", tags).Add(1)
			s.metrics.Histogram("openase.http.server.duration_seconds", tags).Record(time.Since(start).Seconds())

			remainingInFlight := atomic.AddInt64(&inFlight, -1)
			s.metrics.Gauge("openase.http.server.in_flight_requests", provider.Tags{
				"server": "http",
			}).Set(float64(remainingInFlight))

			return err
		}
	}
}

func (s *Server) handleMetrics(c echo.Context) error {
	if s.metricsHandler == nil {
		return echo.NewHTTPError(http.StatusNotFound, "metrics export is disabled")
	}

	s.metricsHandler.ServeHTTP(c.Response(), c.Request())
	return nil
}
