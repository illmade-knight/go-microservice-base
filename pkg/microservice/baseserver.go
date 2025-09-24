// github.com/illmade-knight/go-iot-dataflows/builder/service.go
package microservice

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

// BaseConfig holds common configuration fields for all services.
type BaseConfig struct {
	LogLevel        string `yaml:"log_level"`
	HTTPPort        string `yaml:"http_port"` // e.g., "8080". The PORT env var will override this.
	ProjectID       string `yaml:"project_id"`
	CredentialsFile string `yaml:"credentials_file"`

	ServiceName        string `yaml:"service_name"`
	DataflowName       string `yaml:"dataflow_name"`
	ServiceDirectorURL string `yaml:"service_director_url"`
}

// Service defines the common interface for all microservices.
type Service interface {
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
	Mux() *http.ServeMux
	GetHTTPPort() string
}

// BaseServer provides common functionalities for microservice servers.
type BaseServer struct {
	Logger     zerolog.Logger
	HTTPPort   string // The listen address, e.g., ":8080"
	httpServer *http.Server
	mux        *http.ServeMux
	actualAddr string
	mu         sync.RWMutex
	readyChan  chan struct{}
	// ADDED: Atomically controlled readiness state.
	isReady *atomic.Value
}

// NewBaseServer creates and initializes a new BaseServer.
func NewBaseServer(logger zerolog.Logger, httpPort string) *BaseServer {
	mux := http.NewServeMux()

	listenAddr := httpPort
	if listenAddr == "" {
		listenAddr = "8080"
	}
	if !strings.HasPrefix(listenAddr, ":") {
		listenAddr = ":" + listenAddr
	}

	isReady := &atomic.Value{}
	isReady.Store(false) // Start in a not-ready state.

	s := &BaseServer{
		Logger:   logger,
		HTTPPort: listenAddr,
		mux:      mux,
		isReady:  isReady,
	}
	s.httpServer = &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	// Register all default handlers
	s.registerDefaultHandlers()
	return s
}

// registerDefaultHandlers sets up the built-in observability endpoints.
func (s *BaseServer) registerDefaultHandlers() {
	s.mux.HandleFunc("/healthz", s.healthzHandler)
	s.mux.HandleFunc("/readyz", s.readyzHandler)
	s.mux.Handle("/metrics", promhttp.Handler()) // Expose Prometheus metrics
}

func (s *BaseServer) SetReadyChannel(ch chan struct{}) {
	s.readyChan = ch
}

// SetReady allows the consuming service to signal that it is ready to serve traffic.
// This is thread-safe.
func (s *BaseServer) SetReady(ready bool) {
	s.isReady.Store(ready)
	if ready {
		s.Logger.Info().Msg("Service has been marked as READY.")
	} else {
		s.Logger.Warn().Msg("Service has been marked as NOT READY.")
	}
}

// Start method is a blocking call.
// It starts the HTTP server and only returns when the server is closed.
func (s *BaseServer) Start() error {
	listener, err := net.Listen("tcp", s.HTTPPort)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", s.HTTPPort, err)
	}

	s.mu.Lock()
	s.actualAddr = listener.Addr().String()
	s.mu.Unlock()

	s.Logger.Info().Str("address", s.actualAddr).Msg("HTTP server starting to listen")

	if s.readyChan != nil {
		close(s.readyChan)
	}

	if err := s.httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.Logger.Error().Err(err).Msg("HTTP server failed")
		return err
	}

	s.Logger.Info().Msg("HTTP server has stopped listening.")
	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *BaseServer) Shutdown(ctx context.Context) error {
	s.Logger.Info().Msg("Shutting down HTTP server...")
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.Logger.Error().Err(err).Msg("Error during HTTP server shutdown.")
		return err
	}
	s.Logger.Info().Msg("HTTP server stopped.")
	return nil
}

// GetHTTPPort returns the actual network port the server is listening on.
func (s *BaseServer) GetHTTPPort() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, port, err := net.SplitHostPort(s.actualAddr)
	if err != nil {
		return s.HTTPPort
	}
	return ":" + port
}

// Mux returns the underlying ServeMux for registering additional handlers.
func (s *BaseServer) Mux() *http.ServeMux {
	return s.mux
}

// healthzHandler is the liveness probe. It always returns 200 OK.
func (s *BaseServer) healthzHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// readyzHandler is the readiness probe. It returns 200 if the service is ready,
// and 503 Service Unavailable otherwise.
func (s *BaseServer) readyzHandler(w http.ResponseWriter, _ *http.Request) {
	if s.isReady.Load().(bool) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("READY"))
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte("NOT READY"))
}
