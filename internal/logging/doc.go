// Package logging provides structured logging for the Smartap server.
//
// This package wraps zap logger with convenience functions for common logging
// patterns used throughout the server. It provides both general logging functions
// and specialized functions for protocol-specific logging needs.
//
// # Log Levels
//
// The package supports standard log levels:
//   - Debug: Detailed debugging info (hex dumps, frame parsing, ping/pong)
//   - Info: Normal operations (connections, messages, state changes)
//   - Warn: Non-fatal issues (connection drops, retries)
//   - Error: Fatal issues (startup failures, critical errors)
//
// # Structured Logging
//
// All log functions use structured fields for queryability:
//
//	logging.Info("Device connected",
//	    zap.String("remote_addr", "192.168.1.100"),
//	    zap.String("device_id", "ABC123"),
//	    zap.String("firmware", "1.2.3"),
//	)
//
// # Specialized Logging
//
// The package provides domain-specific logging functions:
//
// Connection Logging:
//
//	logging.LogConnection(remoteAddr, "connection_accepted")
//	logging.LogConnection(remoteAddr, "tls_handshake_complete")
//	logging.LogConnection(remoteAddr, "websocket_upgraded")
//	logging.LogConnection(remoteAddr, "websocket_closed")
//
// WebSocket Message Logging:
//
//	logging.LogWebSocketMessage(remoteAddr, "received", msgType, payload)
//	logging.LogWebSocketMessage(remoteAddr, "sent", msgType, payload)
//
// HTTP Request Logging:
//
//	logging.LogHTTPRequest(r, statusCode, responseSize)
//
// # Configuration
//
// Initialize logging at server startup:
//
//	if err := logging.InitLogger("debug"); err != nil {
//	    log.Fatal(err)
//	}
//	defer logging.Sync()
//
// # Output Format
//
// Logs are written to stdout in console format (human-readable) for development
// and can be configured for JSON format in production:
//
//	2025-11-25T10:30:45.123-0800  INFO  Connection event
//	  remote_addr=192.168.1.100
//	  event=connection_accepted
//
// # Thread Safety
//
// All logging functions are safe for concurrent use. The underlying zap logger
// handles synchronization automatically.
package logging
