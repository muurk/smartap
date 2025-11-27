// Package server implements a WebSocket server for Smartap IoT devices.
//
// This package provides a custom WebSocket server designed specifically for TI CC3200-based
// Smartap devices. The server handles TLS configuration, HTTP 101 upgrade responses, and
// WebSocket frame processing with exact compatibility for the device's requirements.
//
// # CC3200 Compatibility
//
// The server addresses specific requirements of the TI CC3200 chipset:
//   - TLS 1.2 with legacy RSA-based cipher suites
//   - Exact HTTP 101 response format (device validates using strstr())
//   - No extra HTTP headers that might confuse the device
//   - Binary WebSocket frames for custom protocol messages
//
// # HTTP 101 Response Format
//
// The server sends this EXACT response (critical for device compatibility):
//
//	HTTP/1.1 101 Switching Protocols\r\n
//	Upgrade: websocket\r\n
//	Connection: Upgrade\r\n
//	\r\n
//
// Note: No Sec-WebSocket-Accept, Server, or Date headers are included.
//
// # TLS Configuration
//
// Supported cipher suites (CC3200 compatible):
//   - TLS_RSA_WITH_AES_128_CBC_SHA256 (0x003C)
//   - TLS_RSA_WITH_AES_256_CBC_SHA256 (0x003D)
//   - TLS_RSA_WITH_AES_128_CBC_SHA (0x002F)
//   - TLS_RSA_WITH_AES_256_CBC_SHA (0x0035)
//   - TLS_RSA_WITH_3DES_EDE_CBC_SHA (0x000A)
//
// # Usage Example
//
//	// Create server configuration
//	config := &server.Config{
//	    Host:     "",     // Listen on all interfaces
//	    Port:     443,    // Standard HTTPS port
//	    CertPath: "/path/to/fullchain.pem",
//	    KeyPath:  "/path/to/privkey.pem",
//	    LogLevel: "info",
//	}
//
//	// Create and start server
//	srv, err := server.New(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Start blocks until shutdown signal or error
//	if err := srv.Start(); err != nil {
//	    log.Fatal(err)
//	}
//
// # Logging
//
// The server provides structured logging with different levels:
//   - debug: Detailed protocol info, hex dumps, ping/pong messages
//   - info: Connection events, messages, state changes
//   - warn: Non-fatal errors, connection issues
//   - error: Fatal errors, unexpected failures
//
// # Message Handling
//
// Incoming WebSocket messages are:
//  1. Parsed as WebSocket frames (FIN, opcode, mask, payload)
//  2. Unmasked if masked by client
//  3. Logged with hex dump for protocol analysis
//  4. Saved to analysis files for debugging
//  5. Passed to protocol handlers for processing
//
// # Graceful Shutdown
//
// The server handles SIGINT and SIGTERM signals for graceful shutdown:
//  1. Stop accepting new connections
//  2. Close existing WebSocket connections
//  3. Wait for in-flight messages to complete
//  4. Clean up resources
//
// # Thread Safety
//
// The server is fully concurrent and handles multiple device connections
// simultaneously. Each connection runs in its own goroutine.
package server
