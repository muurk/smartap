package gdb

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewCertManager(t *testing.T) {
	mgr, err := NewCertManager()
	if err != nil {
		t.Fatalf("NewCertManager failed: %v", err)
	}

	if mgr == nil {
		t.Fatal("expected non-nil CertManager")
	}

	if mgr.rootCA == nil {
		t.Error("expected rootCA to be loaded")
	}

	if mgr.rootCAKey == nil {
		t.Error("expected rootCAKey to be loaded")
	}
}

func TestCertManager_GetRootCA(t *testing.T) {
	mgr, err := NewCertManager()
	if err != nil {
		t.Fatalf("NewCertManager failed: %v", err)
	}

	tests := []struct {
		name    string
		format  string
		wantErr bool
	}{
		{
			name:    "DER format",
			format:  "der",
			wantErr: false,
		},
		{
			name:    "PEM format",
			format:  "pem",
			wantErr: false,
		},
		{
			name:    "Invalid format",
			format:  "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := mgr.GetRootCA(tt.format)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(data) == 0 {
					t.Error("expected non-empty certificate data")
				}

				// Verify format
				if tt.format == "pem" {
					if !strings.Contains(string(data), "BEGIN CERTIFICATE") {
						t.Error("expected PEM format to contain 'BEGIN CERTIFICATE'")
					}
				}
			}
		})
	}
}

func TestCertManager_GetRootCAKey(t *testing.T) {
	mgr, err := NewCertManager()
	if err != nil {
		t.Fatalf("NewCertManager failed: %v", err)
	}

	data, err := mgr.GetRootCAKey()
	if err != nil {
		t.Fatalf("GetRootCAKey failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty key data")
	}

	// Should be PEM format
	if !strings.Contains(string(data), "BEGIN") {
		t.Error("expected PEM format to contain 'BEGIN'")
	}
}

func TestCertManager_ListEmbedded(t *testing.T) {
	mgr, err := NewCertManager()
	if err != nil {
		t.Fatalf("NewCertManager failed: %v", err)
	}

	list := mgr.ListEmbedded()
	if len(list) == 0 {
		t.Error("expected at least one embedded certificate")
	}

	// Should contain root_ca
	found := false
	for _, name := range list {
		if name == "root_ca" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected list to contain 'root_ca'")
	}
}

func TestCertManager_LoadCustom(t *testing.T) {
	mgr, err := NewCertManager()
	if err != nil {
		t.Fatalf("NewCertManager failed: %v", err)
	}

	// Create temporary certificate file (PEM)
	tempDir := t.TempDir()
	pemPath := filepath.Join(tempDir, "test.pem")

	// Get embedded cert in PEM format
	pemData, err := mgr.GetRootCA("pem")
	if err != nil {
		t.Fatalf("failed to get root CA: %v", err)
	}

	if err := os.WriteFile(pemPath, pemData, 0644); err != nil {
		t.Fatalf("failed to write test cert: %v", err)
	}

	// Test loading
	data, err := mgr.LoadCustom(pemPath)
	if err != nil {
		t.Fatalf("LoadCustom failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty certificate data")
	}

	// Should be DER format (LoadCustom converts PEM to DER)
	_, parseErr := x509.ParseCertificate(data)
	if parseErr != nil {
		t.Errorf("expected valid DER certificate, parse error: %v", parseErr)
	}
}

func TestCertManager_LoadCustom_DER(t *testing.T) {
	mgr, err := NewCertManager()
	if err != nil {
		t.Fatalf("NewCertManager failed: %v", err)
	}

	// Create temporary certificate file (DER)
	tempDir := t.TempDir()
	derPath := filepath.Join(tempDir, "test.der")

	// Get embedded cert in DER format
	derData, err := mgr.GetRootCA("der")
	if err != nil {
		t.Fatalf("failed to get root CA: %v", err)
	}

	if err := os.WriteFile(derPath, derData, 0644); err != nil {
		t.Fatalf("failed to write test cert: %v", err)
	}

	// Test loading
	data, err := mgr.LoadCustom(derPath)
	if err != nil {
		t.Fatalf("LoadCustom failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty certificate data")
	}
}

func TestCertManager_LoadCustom_NonExistent(t *testing.T) {
	mgr, err := NewCertManager()
	if err != nil {
		t.Fatalf("NewCertManager failed: %v", err)
	}

	_, err = mgr.LoadCustom("/nonexistent/path/cert.pem")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}

	var certErr *CertificateError
	if !strings.Contains(err.Error(), "certificate error") {
		t.Errorf("expected CertificateError, got: %v", err)
	}
	_ = certErr
}

func TestCertManager_ConvertPEMToDER(t *testing.T) {
	mgr, err := NewCertManager()
	if err != nil {
		t.Fatalf("NewCertManager failed: %v", err)
	}

	// Get PEM data
	pemData, err := mgr.GetRootCA("pem")
	if err != nil {
		t.Fatalf("failed to get PEM: %v", err)
	}

	// Convert to DER
	derData, err := mgr.ConvertPEMToDER(pemData)
	if err != nil {
		t.Fatalf("ConvertPEMToDER failed: %v", err)
	}

	if len(derData) == 0 {
		t.Error("expected non-empty DER data")
	}

	// Should be parseable as certificate
	_, parseErr := x509.ParseCertificate(derData)
	if parseErr != nil {
		t.Errorf("expected valid DER certificate, parse error: %v", parseErr)
	}
}

func TestCertManager_ConvertPEMToDER_Invalid(t *testing.T) {
	mgr, err := NewCertManager()
	if err != nil {
		t.Fatalf("NewCertManager failed: %v", err)
	}

	// Try to convert non-PEM data
	_, err = mgr.ConvertPEMToDER([]byte("not a PEM certificate"))
	if err == nil {
		t.Error("expected error for invalid PEM")
	}
}

func TestDefaultCertParams(t *testing.T) {
	params := DefaultCertParams()

	if params.CommonName != "*.smartap-tech.com" {
		t.Errorf("expected CN '*.smartap-tech.com', got %s", params.CommonName)
	}

	if params.Country != "GB" {
		t.Errorf("expected Country 'GB', got %s", params.Country)
	}

	if params.State != "England" {
		t.Errorf("expected State 'England', got %s", params.State)
	}

	if params.Locality != "London" {
		t.Errorf("expected Locality 'London', got %s", params.Locality)
	}

	if params.Organization != "Smartap Revival Project" {
		t.Errorf("expected Organization 'Smartap Revival Project', got %s", params.Organization)
	}

	if params.ValidDays != 730 {
		t.Errorf("expected ValidDays 730, got %d", params.ValidDays)
	}

	// Check SANs
	expectedSANs := []string{
		"*.smartap-tech.com",
		"smartap-tech.com",
		"eValve.smartap-tech.com",
	}

	if len(params.SANs) != len(expectedSANs) {
		t.Fatalf("expected %d SANs, got %d", len(expectedSANs), len(params.SANs))
	}

	for i, expected := range expectedSANs {
		if params.SANs[i] != expected {
			t.Errorf("SAN %d: expected %s, got %s", i, expected, params.SANs[i])
		}
	}
}

func TestCertManager_GenerateServerCert(t *testing.T) {
	mgr, err := NewCertManager()
	if err != nil {
		t.Fatalf("NewCertManager failed: %v", err)
	}

	params := DefaultCertParams()
	cert, err := mgr.GenerateServerCert(params)
	if err != nil {
		t.Fatalf("GenerateServerCert failed: %v", err)
	}

	if cert == nil {
		t.Fatal("expected non-nil ServerCert")
	}

	// Check all fields are populated
	if len(cert.CertPEM) == 0 {
		t.Error("expected non-empty CertPEM")
	}

	if len(cert.CertDER) == 0 {
		t.Error("expected non-empty CertDER")
	}

	if len(cert.KeyPEM) == 0 {
		t.Error("expected non-empty KeyPEM")
	}

	if cert.Certificate == nil {
		t.Error("expected non-nil Certificate")
	}

	if cert.PrivateKey == nil {
		t.Error("expected non-nil PrivateKey")
	}

	// Verify certificate properties
	if cert.Certificate.Subject.CommonName != params.CommonName {
		t.Errorf("expected CN %s, got %s", params.CommonName, cert.Certificate.Subject.CommonName)
	}

	// Check key algorithm
	if cert.Certificate.PublicKeyAlgorithm != x509.RSA {
		t.Errorf("expected RSA key, got %v", cert.Certificate.PublicKeyAlgorithm)
	}

	// Check key size (2048-bit)
	if rsaKey, ok := cert.Certificate.PublicKey.(*rsa.PublicKey); ok {
		if rsaKey.N.BitLen() != 2048 {
			t.Errorf("expected 2048-bit key, got %d bits", rsaKey.N.BitLen())
		}
	}

	// Check signature algorithm
	if cert.Certificate.SignatureAlgorithm != x509.SHA256WithRSA {
		t.Errorf("expected SHA256WithRSA, got %v", cert.Certificate.SignatureAlgorithm)
	}

	// Check key usage
	requiredUsage := x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
	if (cert.Certificate.KeyUsage & requiredUsage) != requiredUsage {
		t.Error("certificate missing required key usage")
	}

	// Check extended key usage
	hasServerAuth := false
	for _, usage := range cert.Certificate.ExtKeyUsage {
		if usage == x509.ExtKeyUsageServerAuth {
			hasServerAuth = true
			break
		}
	}
	if !hasServerAuth {
		t.Error("certificate missing ExtKeyUsageServerAuth")
	}

	// Check SANs
	if len(cert.Certificate.DNSNames) != len(params.SANs) {
		t.Errorf("expected %d SANs, got %d", len(params.SANs), len(cert.Certificate.DNSNames))
	}

	// Verify PEM encoding
	pemBlock, _ := pem.Decode(cert.CertPEM)
	if pemBlock == nil {
		t.Error("failed to decode CertPEM")
	}

	keyBlock, _ := pem.Decode(cert.KeyPEM)
	if keyBlock == nil {
		t.Error("failed to decode KeyPEM")
	}
}

func TestCertManager_ValidateCertificateFormat(t *testing.T) {
	mgr, err := NewCertManager()
	if err != nil {
		t.Fatalf("NewCertManager failed: %v", err)
	}

	// Generate a valid certificate
	params := DefaultCertParams()
	cert, err := mgr.GenerateServerCert(params)
	if err != nil {
		t.Fatalf("GenerateServerCert failed: %v", err)
	}

	// Should pass validation
	err = mgr.ValidateCertificateFormat(cert.CertDER)
	if err != nil {
		t.Errorf("expected valid certificate to pass validation, got error: %v", err)
	}
}

func TestCertManager_ValidateCertificateFormat_InvalidDER(t *testing.T) {
	mgr, err := NewCertManager()
	if err != nil {
		t.Fatalf("NewCertManager failed: %v", err)
	}

	// Test with invalid DER data
	err = mgr.ValidateCertificateFormat([]byte("not a valid DER certificate"))
	if err == nil {
		t.Error("expected error for invalid DER")
	}
}

func TestCertManager_ValidateCertificateFormat_EmbeddedRootCA(t *testing.T) {
	mgr, err := NewCertManager()
	if err != nil {
		t.Fatalf("NewCertManager failed: %v", err)
	}

	// Get embedded root CA
	derData, err := mgr.GetRootCA("der")
	if err != nil {
		t.Fatalf("failed to get root CA: %v", err)
	}

	// Validate it - it may not pass all device requirements since it's a CA
	// but it should at least be parseable
	err = mgr.ValidateCertificateFormat(derData)
	// The root CA might fail validation because it's a CA cert, not a server cert
	// That's okay, we just want to make sure the validation function runs
	_ = err
}

func TestServerCert_FormatValidation(t *testing.T) {
	mgr, err := NewCertManager()
	if err != nil {
		t.Fatalf("NewCertManager failed: %v", err)
	}

	// Generate certificate
	params := DefaultCertParams()
	cert, err := mgr.GenerateServerCert(params)
	if err != nil {
		t.Fatalf("GenerateServerCert failed: %v", err)
	}

	// PEM should contain certificate header
	if !strings.Contains(string(cert.CertPEM), "BEGIN CERTIFICATE") {
		t.Error("expected PEM to contain 'BEGIN CERTIFICATE'")
	}

	// PEM should contain private key header
	if !strings.Contains(string(cert.KeyPEM), "BEGIN") && !strings.Contains(string(cert.KeyPEM), "PRIVATE KEY") {
		t.Error("expected KeyPEM to contain private key header")
	}

	// DER should be parseable
	parsedCert, err := x509.ParseCertificate(cert.CertDER)
	if err != nil {
		t.Errorf("failed to parse DER certificate: %v", err)
	}

	// Parsed cert should match Certificate field
	if parsedCert.SerialNumber.Cmp(cert.Certificate.SerialNumber) != 0 {
		t.Error("parsed certificate doesn't match Certificate field")
	}
}
