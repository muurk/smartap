# Certificate Details

Deep dive into the certificate replacement process.

## Original Certificate Chain

### Production Setup

The original Smartap cloud service used:

```
AddTrust External CA Root
    └── COMODO RSA Certification Authority
            └── COMODO RSA Domain Validation Secure Server CA
                    └── *.smartap-tech.com (Server Certificate)
```

**Server Certificate Details** ([crt.sh](https://crt.sh/?id=6715786779)):

- **CN:** `*.smartap-tech.com`
- **SAN:** `*.smartap-tech.com`, `smartap-tech.com`
- **Issuer:** COMODO RSA Domain Validation Secure Server CA
- **Valid:** May 2022 - June 2023 (Expired)
- **Key:** RSA 2048-bit
- **Signature:** SHA-256

### Device Certificate Store

The device stores certificates in flash filesystem:

| File | Purpose | Replaceable |
|------|---------|-------------|
| `/cert/129.der` | CA Certificate (trust anchor) | ✅ Yes - via JTAG |
| `/cert/130.der` | Client Public Key | ⚠️ Possible but not needed |
| `/cert/131.der` | Client Private Key | ⚠️ Possible but not needed |

**Key Discovery:** Certificate `/cert/129.der` is the trust anchor. Replacing this allows the device to trust any certificate signed by our custom CA.

## Custom Certificate Authority

### Generation Process

The provided script generates:

1. **Root CA** (ca-root-cert.pem / ca-root-cert.der)
2. **Server Certificate** (server-cert.pem)
3. **Server Private Key** (server-key.pem)
4. **Full Chain** (server-fullchain.pem)

### Root CA Specifications

```bash
# 4096-bit RSA key
openssl genrsa -out ca-root-key.pem 4096

# Self-signed certificate, 10-year validity
openssl req -new -x509 -days 3650 -key ca-root-key.pem \
  -out ca-root-cert.pem \
  -subj "/C=GB/ST=England/L=London/O=Smartap/CN=Smartap Root CA" \
  -addext "basicConstraints=critical,CA:TRUE" \
  -addext "keyUsage=critical,keyCertSign,cRLSign"
```

**Important Attributes:**

- `basicConstraints=critical,CA:TRUE` - Marks as CA certificate
- `keyUsage=critical,keyCertSign,cRLSign` - Allowed to sign certificates
- Long validity - Reduces need to reflash device

### Server Certificate Specifications

**Wildcard Version:**

```bash
# 2048-bit key (matches original)
openssl genrsa -out server-key.pem 2048

# CSR with wildcard CN
openssl req -new -key server-key.pem -out server.csr \
  -subj "/C=GB/ST=England/L=London/O=Smartap/CN=*.smartap-tech.com"

# Sign with CA
openssl x509 -req -in server.csr -CA ca-root-cert.pem -CAkey ca-root-key.pem \
  -out server-cert.pem -days 730 -sha256 \
  -extfile extensions.cnf
```

**extensions.cnf:**
```
basicConstraints = CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = *.smartap-tech.com
DNS.2 = smartap-tech.com
DNS.3 = evalve.smartap-tech.com
```

## Device Certificate Flashing

### Filesystem Access via JTAG

The CC3200 uses TI's SimpleLink filesystem. Key functions:

```c
// Function addresses (from memory analysis)
sl_FsOpen  = 0x20015c64
sl_FsWrite = 0x20014bf8
sl_FsClose = 0x2001555c
sl_FsDel   = 0x20016ea8
```

### File Mode Calculation

The filesystem requires a complex mode parameter:

```c
mode = (Access << 12) | (SizeGran << 8) | (Size << 0) | (Flags << 16)
```

**Parameters:**

- **Access:** `3` (_FS_MODE_OPEN_WRITE_CREATE_IF_NOT_EXIST)
- **SizeGran:** Granularity index (0 = 256 bytes)
- **Size:** Number of blocks (cert_size / granularity)
- **Flags:** `0x5` (COMMIT | NO_SIGNATURE_TEST)

### GDB Script Process

1. **Halt device** - Stop CPU execution
2. **Load certificate** - Copy DER file to device RAM
3. **Delete old certificate** - Call `sl_FsDel` on `/cert/129.der`
4. **Create new file** - Call `sl_FsOpen` with calculated mode
5. **Write data** - Call `sl_FsWrite` with certificate data
6. **Close file** - Call `sl_FsClose` to commit
7. **Resume device** - Continue execution

### Memory Layout

```
0x20004000    Code/Data region
0x20030000    Work buffer (certificate data loaded here)
0x20031000    File handle storage
0x20031004    Filename string
0x20031d00    Stack pointer for function calls
```

## SSL Validation in Device

### Three Security Checks

1. **Domain Name Verification**
   - Checks for "smartap.com" in certificate CN
   - Simple string match

2. **Certificate Chain Validation**
   - Validates full chain to trusted CA
   - Uses SimpleLink SSL functions
   - Checks against `/cert/129.der`

3. **Socket-Level Security**
   - Method verification (`SL_SO_SECMETHOD`)
   - Private key validation
   - Domain name check (`SO_SECURE_DOMAIN_NAME_VERIFICATION`)

### Why Certificate Replacement Works

By replacing `/cert/129.der`:

- Device trusts our custom CA
- Any certificate signed by our CA passes validation
- Domain check still passes (*.smartap-tech.com matches)
- No code modification required!

## Security Implications

### Preserved Security

- TLS encryption still enforced
- Certificate validation still performed
- Man-in-the-middle attacks still prevented

### Changed Trust Model

**Before:** Device trusted COMODO (public CA)
**After:** Device trusts your custom CA (private)

**Implications:**

- Only certificates signed by YOUR CA are trusted
- No internet CAs can intercept communication
- You control the trust chain completely

### Best Practices

1. **Protect CA private key**
   - Store offline when not in use
   - Never share or expose
   - Back up securely

2. **Server certificate management**
   - Reasonable expiry (1-2 years)
   - Regenerate periodically
   - Use strong key sizes

3. **Physical security**
   - JTAG access requires physical presence
   - Certificates can't be changed remotely
   - Device must be physically accessed to compromise

## Certificate Expiry

### Root CA

- Generated with 10-year validity
- After expiry, would need to reflash device
- Plan ahead!

### Server Certificates

- Generated with 2-year validity
- Can be regenerated anytime without device access
- Just restart server with new certificate

## Troubleshooting

### Device Won't Connect After Flashing

**Check:**

1. Certificate format correct (DER, not PEM)
2. Certificate written successfully (check GDB output)
3. Server using matching certificate signed by same CA
4. Domain name in server cert matches device expectations

### TLS Handshake Fails

**Verify:**

1. Server certificate chain includes CA cert
2. Server certificate has correct domain (SAN)
3. Time sync (certificates have validity periods)
4. TLS version compatibility

### Certificate Flashing Failed

**Recovery:**

1. Reflash with original certificate (if backed up)
2. Regenerate custom certificate and try again
3. Check JTAG connection stability
4. Verify filesystem functions at correct addresses

## References

- [TI CC3200 Security Features](https://www.ti.com/lit/ug/swru369b/swru369b.pdf)
- [TI CC3200 SDK SSL Example](https://software-dl.ti.com/ecs/CC3200SDK/1_5_0/exports/cc3200-sdk/example/ssl/README.html)
- [OpenSSL Certificate Documentation](https://www.openssl.org/docs/)

---

[:material-arrow-left: Previous: Protocol](protocol.md){ .md-button }
[:material-arrow-right: Next: Development](development.md){ .md-button .md-button--primary }
