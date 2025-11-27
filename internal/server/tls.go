package server

import (
	"crypto/tls"
	"fmt"

	"github.com/muurk/smartap/internal/logging"
	"go.uber.org/zap"
)

// NewTLSConfig creates a TLS configuration compatible with TI CC3200 devices
// The CC3200 only supports TLS 1.2 with specific cipher suites
func NewTLSConfig(certPath, keyPath string) (*tls.Config, error) {
	// Load certificate and private key
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
	}

	logging.Info("TLS configuration created from files",
		zap.String("cert", certPath),
		zap.String("key", keyPath),
		zap.String("tls_version", "1.2 only"),
	)

	return buildCC3200TLSConfig(cert), nil
}

// NewTLSConfigFromMemory creates a TLS configuration from in-memory certificate and key (PEM format)
// This is used when auto-generating certificates signed by the embedded Root CA
func NewTLSConfigFromMemory(certPEM, keyPEM []byte) (*tls.Config, error) {
	// Load certificate from PEM-encoded data
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS certificate from memory: %w", err)
	}

	logging.Info("TLS configuration created from in-memory certificate",
		zap.String("source", "auto-generated"),
		zap.String("tls_version", "1.2 only"),
	)

	return buildCC3200TLSConfig(cert), nil
}

// buildCC3200TLSConfig creates a TLS config with CC3200-compatible settings
func buildCC3200TLSConfig(cert tls.Certificate) *tls.Config {
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},

		// Force TLS 1.2 only (CC3200 doesn't support TLS 1.3)
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS12,

		// CC3200 compatible cipher suites
		// These are RSA-based cipher suites that the CC3200 supports
		// Using hex values directly because some constants don't exist in Go's TLS package
		CipherSuites: []uint16{
			0x003C, // TLS_RSA_WITH_AES_128_CBC_SHA256
			0x003D, // TLS_RSA_WITH_AES_256_CBC_SHA256
			0x002F, // TLS_RSA_WITH_AES_128_CBC_SHA
			0x0035, // TLS_RSA_WITH_AES_256_CBC_SHA
			0x000A, // TLS_RSA_WITH_3DES_EDE_CBC_SHA
		},

		// Prefer server cipher suites to ensure we use a compatible one
		PreferServerCipherSuites: true,

		// Enable session tickets for performance
		SessionTicketsDisabled: false,

		// Callback to log TLS handshake details
		VerifyConnection: func(cs tls.ConnectionState) error {
			logging.LogTLSHandshake(
				cs.ServerName,
				cs.Version,
				cs.CipherSuite,
				cs.ServerName,
			)
			return nil
		},
	}

	return config
}

// GetTLSInfo returns human-readable TLS configuration information
func GetTLSInfo(config *tls.Config) map[string]interface{} {
	cipherNames := []string{
		"TLS_RSA_WITH_AES_128_CBC_SHA256",
		"TLS_RSA_WITH_AES_256_CBC_SHA256",
		"TLS_RSA_WITH_AES_128_CBC_SHA",
		"TLS_RSA_WITH_AES_256_CBC_SHA",
		"TLS_RSA_WITH_3DES_EDE_CBC_SHA",
	}

	return map[string]interface{}{
		"min_version":     "TLS 1.2",
		"max_version":     "TLS 1.2",
		"cipher_suites":   cipherNames,
		"num_certs":       len(config.Certificates),
		"session_tickets": !config.SessionTicketsDisabled,
	}
}
