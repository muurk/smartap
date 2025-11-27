# Custom Certificates

Using your own TLS certificates with smartap-server.

## When You Need Custom Certificates

In most cases, you don't need custom certificates. The server auto-generates certificates signed by the embedded Root CA, which matches what's injected into devices via `smartap-jtag inject-certs`.

You might want custom certificates if:

- You're using a different CA than the embedded one
- You want specific certificate attributes
- You're running multiple servers with different certificates
- You're doing advanced certificate chain testing

## Using Custom Certificates

```bash
./smartap-server server --cert /path/to/fullchain.pem --key /path/to/privkey.pem
```

**Requirements:**

- Certificate must be PEM format
- Certificate chain should be complete (server cert + intermediates)
- Key must be PEM format (unencrypted)
- Certificate must be signed by the CA that's on your device

!!! warning "CA Must Match"
    Your server certificate must be signed by the same CA that's injected into your device. If you use a custom CA, you'll need to inject that CA using:

    ```bash
    smartap-jtag inject-certs --cert-file /path/to/your-ca.der
    ```

## Generating Custom Certificates

### Create a Custom CA

```bash
# Generate CA private key
openssl genrsa -out ca-key.pem 4096

# Generate CA certificate
openssl req -new -x509 -days 3650 -key ca-key.pem -out ca-cert.pem \
    -subj "/CN=Custom Smartap CA"

# Convert to DER format for device injection
openssl x509 -in ca-cert.pem -outform DER -out ca-cert.der
```

### Create Server Certificate

```bash
# Generate server private key
openssl genrsa -out server-key.pem 2048

# Generate certificate signing request
openssl req -new -key server-key.pem -out server.csr \
    -subj "/CN=evalve.smartap-tech.com"

# Sign with CA
openssl x509 -req -in server.csr -CA ca-cert.pem -CAkey ca-key.pem \
    -CAcreateserial -out server-cert.pem -days 365 \
    -extfile <(echo "subjectAltName=DNS:evalve.smartap-tech.com,DNS:smartap-tech.com,DNS:*.smartap-tech.com")

# Create full chain
cat server-cert.pem ca-cert.pem > fullchain.pem
```

### Inject Custom CA

Before using custom certificates, inject your CA into the device:

```bash
smartap-jtag inject-certs --cert-file ca-cert.der
```

### Start Server with Custom Certs

```bash
./smartap-server server --cert fullchain.pem --key server-key.pem
```

## Certificate Requirements

For the device to accept your server's certificate:

1. **Signed by trusted CA** - The CA must be in the device's trust store (`/cert/129.der`)
2. **Valid domain** - Must include `evalve.smartap-tech.com` or `*.smartap-tech.com`
3. **Not expired** - Certificate must be within validity period
4. **RSA or ECDSA** - CC3200 supports both, RSA recommended for compatibility

## Troubleshooting

### "TLS handshake failed"

- Verify certificate chain is complete
- Check CA matches what's on device
- Confirm domain name is correct

### "Certificate file not found"

- Use absolute paths
- Verify file permissions

### Device rejects connection

- The device has a different CA than your server certificate
- Re-inject your CA to the device:
  ```bash
  smartap-jtag inject-certs --cert-file your-ca.der
  ```

---

[:material-arrow-left: Previous: Quick Start](quick-start.md){ .md-button }
[:material-arrow-right: Next: Limitations](limitations.md){ .md-button .md-button--primary }
