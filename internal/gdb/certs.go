package gdb

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	_ "embed"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"
)

// Embedded certificate files
//
//go:embed certs/ca-root-cert.der
var rootCADER []byte

//go:embed certs/ca-root-cert.pem
var rootCAPEM []byte

//go:embed certs/ca-root-key.pem
var rootCAKeyPEM []byte

// CertManager manages embedded and custom certificates.
type CertManager struct {
	// rootCA is the parsed root CA certificate
	rootCA *x509.Certificate
	// rootCAKey is the parsed root CA private key
	rootCAKey *rsa.PrivateKey
}

// NewCertManager creates a new certificate manager with embedded certs.
func NewCertManager() (*CertManager, error) {
	mgr := &CertManager{}

	// Parse root CA certificate
	block, _ := pem.Decode(rootCAPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode root CA PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse root CA certificate: %w", err)
	}
	mgr.rootCA = cert

	// Parse root CA private key
	keyBlock, _ := pem.Decode(rootCAKeyPEM)
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode root CA key PEM")
	}

	key, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		// Try PKCS1 format
		key, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse root CA private key: %w", err)
		}
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("root CA key is not RSA")
	}
	mgr.rootCAKey = rsaKey

	return mgr, nil
}

// GetRootCA returns the embedded root CA certificate in the specified format.
// format can be "der" or "pem".
func (cm *CertManager) GetRootCA(format string) ([]byte, error) {
	switch format {
	case "der":
		return rootCADER, nil
	case "pem":
		return rootCAPEM, nil
	default:
		return nil, fmt.Errorf("unsupported format: %s (use 'der' or 'pem')", format)
	}
}

// GetRootCAKey returns the embedded root CA private key in PEM format.
func (cm *CertManager) GetRootCAKey() ([]byte, error) {
	return rootCAKeyPEM, nil
}

// ListEmbedded returns the names of available embedded certificates.
func (cm *CertManager) ListEmbedded() []string {
	return []string{
		"root_ca",
	}
}

// LoadCustom loads a certificate from the file system.
// The file can be in PEM or DER format.
func (cm *CertManager) LoadCustom(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &CertificateError{
			Operation: "load",
			Path:      path,
			Err:       err,
		}
	}

	// Check if it's PEM format
	if block, _ := pem.Decode(data); block != nil {
		// It's PEM, return the decoded bytes (DER)
		return block.Bytes, nil
	}

	// Assume it's already DER
	return data, nil
}

// ConvertPEMToDER converts a PEM-encoded certificate to DER format.
func (cm *CertManager) ConvertPEMToDER(pemData []byte) ([]byte, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, &CertificateError{
			Operation: "convert",
			Err:       fmt.Errorf("not a valid PEM certificate"),
		}
	}
	return block.Bytes, nil
}

// CertParams holds parameters for generating a server certificate.
type CertParams struct {
	// CommonName is the CN field (default: *.smartap-tech.com)
	CommonName string
	// Country is the C field (default: GB)
	Country string
	// State is the ST field (default: England)
	State string
	// Locality is the L field (default: London)
	Locality string
	// Organization is the O field (default: Smartap Revival Project)
	Organization string
	// SANs are the Subject Alternative Names
	SANs []string
	// ValidDays is certificate validity in days (default: 730 = 2 years)
	ValidDays int
}

// DefaultCertParams returns CertParams with the exact values required by the device.
// These values MUST match the format in regenerate-wildcard-cert.sh.
func DefaultCertParams() CertParams {
	return CertParams{
		CommonName:   "*.smartap-tech.com",
		Country:      "GB",
		State:        "England",
		Locality:     "London",
		Organization: "Smartap Revival Project",
		SANs: []string{
			"*.smartap-tech.com",
			"smartap-tech.com",
			"eValve.smartap-tech.com",
		},
		ValidDays: 730,
	}
}

// ServerCert represents a generated server certificate.
type ServerCert struct {
	// CertPEM is the certificate in PEM format
	CertPEM []byte
	// CertDER is the certificate in DER format
	CertDER []byte
	// KeyPEM is the private key in PEM format
	KeyPEM []byte
	// Certificate is the parsed x509 certificate
	Certificate *x509.Certificate
	// PrivateKey is the RSA private key
	PrivateKey *rsa.PrivateKey
}

// GenerateServerCert generates a new server certificate signed by the embedded Root CA.
// The certificate will match the exact format required by the CC3200 device:
//   - RSA 2048-bit key
//   - SHA-256 signature
//   - Key usage: digitalSignature, keyEncipherment
//   - Extended key usage: serverAuth
//   - Subject and SANs as specified in params
func (cm *CertManager) GenerateServerCert(params CertParams) (*ServerCert, error) {
	// Generate RSA 2048-bit key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, &CertificateError{
			Operation: "generate_key",
			Err:       err,
		}
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, &CertificateError{
			Operation: "generate_serial",
			Err:       err,
		}
	}

	notBefore := time.Now()
	notAfter := notBefore.AddDate(0, 0, params.ValidDays)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Country:      []string{params.Country},
			Province:     []string{params.State},
			Locality:     []string{params.Locality},
			Organization: []string{params.Organization},
			CommonName:   params.CommonName,
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		// Key usage as required by device
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},

		// Subject Alternative Names
		DNSNames: params.SANs,

		// Not a CA
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	// Create certificate signed by Root CA
	certDER, err := x509.CreateCertificate(
		rand.Reader,
		&template,
		cm.rootCA,
		&privateKey.PublicKey,
		cm.rootCAKey,
	)
	if err != nil {
		return nil, &CertificateError{
			Operation: "create_certificate",
			Err:       err,
		}
	}

	// Parse the generated certificate
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, &CertificateError{
			Operation: "parse_certificate",
			Err:       err,
		}
	}

	// Encode to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return &ServerCert{
		CertPEM:     certPEM,
		CertDER:     certDER,
		KeyPEM:      keyPEM,
		Certificate: cert,
		PrivateKey:  privateKey,
	}, nil
}

// ValidateCertificateFormat validates that a certificate matches device requirements.
// Returns nil if valid, error describing the problem if invalid.
func (cm *CertManager) ValidateCertificateFormat(certDER []byte) error {
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return &CertificateError{
			Operation: "validate",
			Err:       fmt.Errorf("failed to parse certificate: %w", err),
		}
	}

	// Check key algorithm
	if cert.PublicKeyAlgorithm != x509.RSA {
		return &CertificateError{
			Operation: "validate",
			Err:       fmt.Errorf("certificate must use RSA keys, got %v", cert.PublicKeyAlgorithm),
		}
	}

	// Check key size (2048-bit)
	if rsaKey, ok := cert.PublicKey.(*rsa.PublicKey); ok {
		if rsaKey.N.BitLen() != 2048 {
			return &CertificateError{
				Operation: "validate",
				Err:       fmt.Errorf("certificate must use 2048-bit RSA key, got %d bits", rsaKey.N.BitLen()),
			}
		}
	}

	// Check signature algorithm (SHA-256)
	if cert.SignatureAlgorithm != x509.SHA256WithRSA {
		return &CertificateError{
			Operation: "validate",
			Err:       fmt.Errorf("certificate must use SHA256WithRSA signature, got %v", cert.SignatureAlgorithm),
		}
	}

	// Check key usage
	requiredUsage := x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
	if (cert.KeyUsage & requiredUsage) != requiredUsage {
		return &CertificateError{
			Operation: "validate",
			Err:       fmt.Errorf("certificate must have KeyUsageDigitalSignature and KeyUsageKeyEncipherment"),
		}
	}

	// Check extended key usage
	hasServerAuth := false
	for _, usage := range cert.ExtKeyUsage {
		if usage == x509.ExtKeyUsageServerAuth {
			hasServerAuth = true
			break
		}
	}
	if !hasServerAuth {
		return &CertificateError{
			Operation: "validate",
			Err:       fmt.Errorf("certificate must have ExtKeyUsageServerAuth"),
		}
	}

	return nil
}
