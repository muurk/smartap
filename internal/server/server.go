package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/muurk/smartap/internal/gdb"
	"github.com/muurk/smartap/internal/logging"
	"go.uber.org/zap"
)

// Config holds the server configuration
type Config struct {
	Host         string
	Port         int
	CertPath     string // Path to certificate file (optional if GenerateCert is true)
	KeyPath      string // Path to private key file (optional if GenerateCert is true)
	GenerateCert bool   // If true, auto-generate certificate signed by embedded Root CA
	LogLevel     string
	AnalysisDir  string // Directory to write message analysis logs (empty = disabled)
}

// Server represents the Smartap WebSocket server
type Server struct {
	config      *Config
	listener    net.Listener
	tlsConfig   *tls.Config
	wg          sync.WaitGroup
	mu          sync.Mutex
	activeConns map[string]net.Conn
	analysisDir string // Directory for message analysis logs (empty = disabled)
}

// New creates a new Server instance
func New(config *Config) (*Server, error) {
	// Initialize logging
	if err := logging.Initialize(config.LogLevel); err != nil {
		return nil, fmt.Errorf("failed to initialize logging: %w", err)
	}

	var tlsConfig *tls.Config
	var err error

	if config.GenerateCert {
		// Generate certificate in memory
		logging.Info("Generating server certificate signed by embedded Root CA")
		tlsConfig, err = generateAndLoadCert()
		if err != nil {
			return nil, fmt.Errorf("failed to generate certificate: %w", err)
		}
		logging.Info("Using auto-generated certificate (in-memory)")
	} else {
		// Load from files
		tlsConfig, err = NewTLSConfig(config.CertPath, config.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS config: %w", err)
		}
	}

	return &Server{
		config:      config,
		tlsConfig:   tlsConfig,
		activeConns: make(map[string]net.Conn),
		analysisDir: config.AnalysisDir,
	}, nil
}

// Start starts the server and blocks until shutdown
func (s *Server) Start() error {
	// Create TLS listener
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	if s.config.GenerateCert {
		logging.Info("Starting Smartap WebSocket Server",
			zap.String("addr", addr),
			zap.String("cert", "auto-generated (in-memory)"),
			zap.String("log_level", s.config.LogLevel),
		)
	} else {
		logging.Info("Starting Smartap WebSocket Server",
			zap.String("addr", addr),
			zap.String("cert", s.config.CertPath),
			zap.String("key", s.config.KeyPath),
			zap.String("log_level", s.config.LogLevel),
		)
	}

	// Log TLS configuration details
	logging.Info("TLS Configuration",
		zap.Any("tls_info", GetTLSInfo(s.tlsConfig)),
	)

	listener, err := tls.Listen("tcp", addr, s.tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to create TLS listener: %w", err)
	}
	s.listener = listener

	logging.Info("Server listening for connections",
		zap.String("addr", addr),
	)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start accepting connections in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- s.acceptConnections()
	}()

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		logging.Info("Shutdown signal received, stopping server...")
		return s.Shutdown(context.Background())
	case err := <-errChan:
		return err
	}
}

// acceptConnections accepts and handles incoming connections
func (s *Server) acceptConnections() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			// Check if listener was closed (during shutdown)
			if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
				return nil
			}
			logging.Error("Failed to accept connection", zap.Error(err))
			continue
		}

		// Handle connection in goroutine
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handleConnection(conn)
		}()
	}
}

// handleConnection handles a single TLS connection
func (s *Server) handleConnection(conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()

	// Track active connection
	s.mu.Lock()
	s.activeConns[remoteAddr] = conn
	s.mu.Unlock()

	defer func() {
		_ = conn.Close()
		s.mu.Lock()
		delete(s.activeConns, remoteAddr)
		s.mu.Unlock()
		logging.LogConnection(remoteAddr, "connection_closed")
	}()

	logging.LogConnection(remoteAddr, "connection_accepted")

	// Log TLS connection state
	if tlsConn, ok := conn.(*tls.Conn); ok {
		// Force TLS handshake
		if err := tlsConn.Handshake(); err != nil {
			logging.Error("TLS handshake failed",
				zap.String("remote_addr", remoteAddr),
				zap.Error(err),
			)
			return
		}

		state := tlsConn.ConnectionState()
		logging.LogTLSHandshake(
			remoteAddr,
			state.Version,
			state.CipherSuite,
			state.ServerName,
		)
	}

	// Read HTTP upgrade request
	req, err := ReadHTTPRequest(conn)
	if err != nil {
		logging.Error("Failed to read HTTP request",
			zap.String("remote_addr", remoteAddr),
			zap.Error(err),
		)
		return
	}

	// Log request details
	LogHTTPRequestDetails(req, remoteAddr)

	// Validate WebSocket upgrade request
	if err := ValidateWebSocketUpgradeRequest(req); err != nil {
		logging.Error("Invalid WebSocket upgrade request",
			zap.String("remote_addr", remoteAddr),
			zap.Error(err),
		)
		return
	}

	// Send our custom HTTP 101 response
	// This is CRITICAL - must be exact format for CC3200 device
	if err := WriteExactHTTP101Response(conn, remoteAddr); err != nil {
		logging.Error("Failed to send HTTP 101 response",
			zap.String("remote_addr", remoteAddr),
			zap.Error(err),
		)
		return
	}

	// Now handle the WebSocket connection
	if err := HandleWebSocketConnection(conn, remoteAddr, s.analysisDir); err != nil {
		logging.Error("WebSocket connection error",
			zap.String("remote_addr", remoteAddr),
			zap.Error(err),
		)
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	logging.Info("Shutting down server...")

	// Close listener to stop accepting new connections
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			logging.Error("Error closing listener", zap.Error(err))
		}
	}

	// Close all active connections
	s.mu.Lock()
	for addr, conn := range s.activeConns {
		logging.Info("Closing active connection", zap.String("remote_addr", addr))
		_ = conn.Close()
	}
	s.mu.Unlock()

	// Wait for all goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logging.Info("All connections closed gracefully")
	case <-ctx.Done():
		logging.Warn("Shutdown timeout, forcing close")
	case <-time.After(10 * time.Second):
		logging.Warn("Shutdown timeout after 10 seconds, forcing close")
	}

	// Sync logger
	logging.Sync()

	return nil
}

// GetActiveConnections returns the number of active connections
func (s *Server) GetActiveConnections() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.activeConns)
}

// generateAndLoadCert generates a server certificate signed by the embedded Root CA
// and returns a TLS configuration using that certificate.
// The certificate is kept in memory only and never written to disk.
func generateAndLoadCert() (*tls.Config, error) {
	// Create certificate manager
	certMgr, err := gdb.NewCertManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create cert manager: %w", err)
	}

	// Use default parameters (matches regenerate-wildcard-cert.sh exactly)
	params := gdb.DefaultCertParams()

	logging.Info("Generating certificate with parameters",
		zap.String("CN", params.CommonName),
		zap.String("Organization", params.Organization),
		zap.Strings("SANs", params.SANs),
		zap.Int("valid_days", params.ValidDays),
	)

	// Generate certificate
	serverCert, err := certMgr.GenerateServerCert(params)
	if err != nil {
		return nil, fmt.Errorf("failed to generate server certificate: %w", err)
	}

	logging.Info("Certificate generated successfully",
		zap.String("CN", serverCert.Certificate.Subject.CommonName),
		zap.String("issuer", serverCert.Certificate.Issuer.CommonName),
		zap.Time("not_before", serverCert.Certificate.NotBefore),
		zap.Time("not_after", serverCert.Certificate.NotAfter),
	)

	// Create TLS config from in-memory certificate
	return NewTLSConfigFromMemory(serverCert.CertPEM, serverCert.KeyPEM)
}
