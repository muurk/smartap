package server

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/muurk/smartap/internal/logging"
	"go.uber.org/zap"
)

// WriteExactHTTP101Response writes the exact HTTP 101 response that the CC3200 device expects
// This is CRITICAL - the device validates the response using strstr() and expects:
//
//	HTTP/1.1 101 Switching Protocols\r\n
//	Upgrade: websocket\r\n
//	Connection: Upgrade\r\n
//	\r\n
//
// Any extra headers or different order/capitalization will cause validation failure
func WriteExactHTTP101Response(conn net.Conn, remoteAddr string) error {
	// Build the exact response the device expects
	// Note: Exact capitalization matters!
	// - "Upgrade: websocket" (lowercase 'w')
	// - "Connection: Upgrade" (uppercase 'U')
	response := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"\r\n"

	// Log the exact bytes we're sending (for debugging with GDB)
	logging.LogRawBytes("HTTP 101 Response", []byte(response))

	// Write to connection
	n, err := conn.Write([]byte(response))
	if err != nil {
		return fmt.Errorf("failed to write HTTP 101 response: %w", err)
	}

	logging.Info("Sent HTTP 101 Switching Protocols response",
		zap.String("remote_addr", remoteAddr),
		zap.Int("bytes_written", n),
	)

	// Log structured response details
	headers := map[string]string{
		"Upgrade":    "websocket",
		"Connection": "Upgrade",
	}
	logging.LogHTTPResponse(remoteAddr, 101, headers)

	return nil
}

// ValidateWebSocketUpgradeRequest checks if the incoming HTTP request is a valid WebSocket upgrade
func ValidateWebSocketUpgradeRequest(req *http.Request) error {
	// Check method
	if req.Method != "GET" {
		return fmt.Errorf("invalid method: %s (expected GET)", req.Method)
	}

	// Check Upgrade header
	upgrade := strings.ToLower(req.Header.Get("Upgrade"))
	if upgrade != "websocket" {
		return fmt.Errorf("invalid Upgrade header: %s (expected websocket)", upgrade)
	}

	// Check Connection header
	connection := strings.ToLower(req.Header.Get("Connection"))
	if !strings.Contains(connection, "upgrade") {
		return fmt.Errorf("invalid Connection header: %s (expected upgrade)", connection)
	}

	// Check Sec-WebSocket-Version
	version := req.Header.Get("Sec-WebSocket-Version")
	if version != "13" {
		return fmt.Errorf("invalid Sec-WebSocket-Version: %s (expected 13)", version)
	}

	// Sec-WebSocket-Key is required (but we don't validate the value)
	if req.Header.Get("Sec-WebSocket-Key") == "" {
		return fmt.Errorf("missing Sec-WebSocket-Key header")
	}

	return nil
}

// ReadHTTPRequest reads an HTTP request from a raw connection
// This is used before we've upgraded to WebSocket
func ReadHTTPRequest(conn net.Conn) (*http.Request, error) {
	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read HTTP request: %w", err)
	}
	return req, nil
}

// LogHTTPRequestDetails logs all details of an HTTP request
func LogHTTPRequestDetails(req *http.Request, remoteAddr string) {
	headers := make(map[string]string)
	for key, values := range req.Header {
		headers[key] = strings.Join(values, ", ")
	}

	logging.LogHTTPRequest(remoteAddr, req.Method, req.URL.Path, headers)

	// Log specific WebSocket headers at debug level
	logging.Debug("WebSocket upgrade request details",
		zap.String("remote_addr", remoteAddr),
		zap.String("host", req.Host),
		zap.String("origin", req.Header.Get("Origin")),
		zap.String("sec_websocket_key", req.Header.Get("Sec-WebSocket-Key")),
		zap.String("sec_websocket_version", req.Header.Get("Sec-WebSocket-Version")),
		zap.String("sec_websocket_protocol", req.Header.Get("Sec-WebSocket-Protocol")),
		zap.String("user_agent", req.Header.Get("User-Agent")),
	)
}
