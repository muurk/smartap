package logging

import (
	"encoding/hex"
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

// LogLevelEnvVar is the environment variable that controls logging verbosity.
// When unset or empty, logging is silent (no zap output).
// Valid values: "debug", "info", "warn", "error"
const LogLevelEnvVar = "SMARTAP_LOG_LEVEL"

// Initialize creates a new logger with the specified level.
// If level is empty, it checks SMARTAP_LOG_LEVEL environment variable.
// If neither is set, logging is disabled (silent mode).
func Initialize(level string) error {
	// If no level provided, check environment variable
	if level == "" {
		level = os.Getenv(LogLevelEnvVar)
	}

	// If still no level, use silent mode (nop logger)
	if level == "" {
		logger = zap.NewNop()
		return nil
	}

	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		// Unknown level - use info as default when explicitly set to something
		zapLevel = zapcore.InfoLevel
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapLevel),
		Development:      false,
		Encoding:         "console",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	// Customize encoder for better readability
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	var err error
	logger, err = config.Build()
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	return nil
}

// InitializeFromEnv initializes the logger from the SMARTAP_LOG_LEVEL
// environment variable. This is the recommended way to initialize logging
// for CLI commands that want silent mode by default.
func InitializeFromEnv() error {
	return Initialize("")
}

// GetLogger returns the global logger instance
func GetLogger() *zap.Logger {
	if logger == nil {
		// Fallback to silent logger if not initialized
		// This ensures no unexpected log output in CLI commands
		logger = zap.NewNop()
	}
	return logger
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...zap.Field) {
	GetLogger().Fatal(msg, fields...)
}

// LogConnection logs a connection event
func LogConnection(remoteAddr string, event string) {
	Info("Connection event",
		zap.String("remote_addr", remoteAddr),
		zap.String("event", event),
	)
}

// LogTLSHandshake logs TLS handshake details
func LogTLSHandshake(remoteAddr string, version uint16, cipherSuite uint16, serverName string) {
	Info("TLS handshake completed",
		zap.String("remote_addr", remoteAddr),
		zap.Uint16("tls_version", version),
		zap.String("tls_version_name", tlsVersionName(version)),
		zap.Uint16("cipher_suite", cipherSuite),
		zap.String("cipher_suite_name", cipherSuiteName(cipherSuite)),
		zap.String("server_name", serverName),
	)
}

// LogHTTPRequest logs an HTTP request
func LogHTTPRequest(remoteAddr string, method string, path string, headers map[string]string) {
	Info("HTTP request received",
		zap.String("remote_addr", remoteAddr),
		zap.String("method", method),
		zap.String("path", path),
		zap.Any("headers", headers),
	)
}

// LogHTTPResponse logs an HTTP response
func LogHTTPResponse(remoteAddr string, statusCode int, headers map[string]string) {
	Info("HTTP response sent",
		zap.String("remote_addr", remoteAddr),
		zap.Int("status_code", statusCode),
		zap.Any("headers", headers),
	)
}

// LogWebSocketMessage logs a WebSocket message
func LogWebSocketMessage(remoteAddr string, direction string, messageType int, data []byte) {
	fields := []zap.Field{
		zap.String("remote_addr", remoteAddr),
		zap.String("direction", direction),
		zap.String("message_type", wsMessageTypeName(messageType)),
		zap.Int("length", len(data)),
	}

	// For binary messages or debug mode, add hex dump
	if messageType == 2 || GetLogger().Core().Enabled(zapcore.DebugLevel) {
		fields = append(fields, zap.String("hex_dump", hexDump(data)))
	}

	// For text messages, include the content
	if messageType == 1 {
		fields = append(fields, zap.String("content", string(data)))
	}

	Info("WebSocket message", fields...)
}

// LogRawBytes logs raw bytes (useful for debugging protocol issues)
func LogRawBytes(label string, data []byte) {
	Debug(label,
		zap.Int("length", len(data)),
		zap.String("hex", hexDump(data)),
		zap.String("ascii", asciiDump(data)),
	)
}

// Helper functions

func tlsVersionName(version uint16) string {
	switch version {
	case 0x0301:
		return "TLS 1.0"
	case 0x0302:
		return "TLS 1.1"
	case 0x0303:
		return "TLS 1.2"
	case 0x0304:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown (0x%04x)", version)
	}
}

func cipherSuiteName(suite uint16) string {
	names := map[uint16]string{
		0x003C: "TLS_RSA_WITH_AES_128_CBC_SHA256",
		0x003D: "TLS_RSA_WITH_AES_256_CBC_SHA256",
		0x002F: "TLS_RSA_WITH_AES_128_CBC_SHA",
		0x0035: "TLS_RSA_WITH_AES_256_CBC_SHA",
		0x000A: "TLS_RSA_WITH_3DES_EDE_CBC_SHA",
	}
	if name, ok := names[suite]; ok {
		return name
	}
	return fmt.Sprintf("Unknown (0x%04x)", suite)
}

func wsMessageTypeName(msgType int) string {
	switch msgType {
	case 1:
		return "text"
	case 2:
		return "binary"
	case 8:
		return "close"
	case 9:
		return "ping"
	case 10:
		return "pong"
	default:
		return fmt.Sprintf("unknown(%d)", msgType)
	}
}

func hexDump(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	// Limit to first 256 bytes for logging
	if len(data) > 256 {
		return hex.EncodeToString(data[:256]) + "..."
	}
	return hex.EncodeToString(data)
}

func asciiDump(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	// Limit to first 256 bytes
	if len(data) > 256 {
		data = data[:256]
	}

	result := make([]byte, len(data))
	for i, b := range data {
		if b >= 32 && b <= 126 {
			result[i] = b
		} else {
			result[i] = '.'
		}
	}
	return string(result)
}

// Sync flushes any buffered log entries
func Sync() {
	if logger != nil {
		_ = logger.Sync()
	}
}
