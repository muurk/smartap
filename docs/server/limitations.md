# Server Limitations

Understanding what the server can and cannot do.

!!! info "Help Improve This"
    The limitations documented here exist because the device protocol isn't fully understood. Your contributions to protocol documentation directly reduce these limitations.

## Protocol Limitations

### Incomplete Protocol Documentation

The device communication protocol was reverse-engineered from memory dumps. Much is still unknown:

| Aspect | Status |
|--------|--------|
| TLS connection | :material-check: Working |
| WebSocket handshake | :material-check: Working |
| Message framing | :material-help-circle: Partially understood |
| Command format | :material-close: Not documented |
| Response parsing | :material-close: Not documented |
| State synchronization | :material-close: Unknown |

### What This Means

- **Server can connect** to devices
- **Server can log messages** for analysis
- **Server cannot control** devices (yet)
- **Server cannot query** device status (yet)

## Feature Limitations

### Not Implemented

These features don't exist in the server:

- :material-close: Remote outlet activation
- :material-close: Temperature monitoring
- :material-close: Scheduling / timers
- :material-close: Usage tracking
- :material-close: Mobile app
- :material-close: Voice assistant integration
- :material-close: Multi-device management
- :material-close: User authentication
- :material-close: Web interface

### Why Not?

Implementing these features requires understanding the device protocol. Once we know how to:

1. Send commands the device understands
2. Parse device responses
3. Track device state

...then these features become possible.

## Hardware Constraints

The CC3200 chip has limitations:

| Constraint | Impact |
|------------|--------|
| 2.4GHz WiFi only | Cannot use 5GHz networks |
| Limited TLS versions | May not support newest ciphers |
| Fixed firmware | Cannot update device software |
| Single connection | Device maintains one server connection |

## Why These Limitations Exist

Understanding the technical constraints helps explain what we're working with.

### The CC3200 TLS Stack

The device uses Texas Instruments' SimpleLink SDK, which has a limited TLS implementation:

| Feature | CC3200 Support | Modern Standard |
|---------|----------------|-----------------|
| TLS Version | 1.2 only | 1.3 preferred |
| Key Exchange | RSA only | ECDHE preferred |
| Cipher Mode | CBC | GCM preferred |
| Certificate | RSA 2048 | RSA/ECDSA 2048+ |

The server is configured to accommodate these constraints:

```go
// From internal/server/tls.go
CipherSuites: []uint16{
    0x003C, // TLS_RSA_WITH_AES_128_CBC_SHA256
    0x003D, // TLS_RSA_WITH_AES_256_CBC_SHA256
    0x002F, // TLS_RSA_WITH_AES_128_CBC_SHA
    0x0035, // TLS_RSA_WITH_AES_256_CBC_SHA
    0x000A, // TLS_RSA_WITH_3DES_EDE_CBC_SHA
},
MinVersion: tls.VersionTLS12,
MaxVersion: tls.VersionTLS12,  // No TLS 1.3
```

This isn't a flaw in our implementation—it's a hardware constraint. The CC3200 was released in 2014 and reflects the TLS landscape of that era.

### The WebSocket Validation Bug

When the device receives an HTTP 101 Switching Protocols response, it validates using something like:

```c
if (strstr(response, "HTTP/1.1 101") == NULL) {
    close_connection();
}
```

Standard Go HTTP libraries add headers that cause this validation to fail:

```
// Standard library sends:
HTTP/1.1 101 Switching Protocols
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=
Server: Go-http-server
Date: Thu, 01 Jan 1970 00:00:00 GMT

// Device expects EXACTLY:
HTTP/1.1 101 Switching Protocols
Upgrade: websocket
Connection: Upgrade

```

The server writes raw bytes to avoid any library interference:

```go
// From internal/server/http.go
response := "HTTP/1.1 101 Switching Protocols\r\n" +
    "Upgrade: websocket\r\n" +
    "Connection: Upgrade\r\n" +
    "\r\n"
conn.Write([]byte(response))
```

This was discovered through trial-and-error. The device simply closed connections until this exact format was used.

### The Binary Protocol

The device doesn't speak JSON or any documented protocol. Communication uses a custom binary format:

```
[0x7e] [0x03] [msg_id: 4 bytes] [length: 2 bytes] [payload: N bytes] [padding]
```

We've identified message types through static analysis of memory dumps:

- `0x01`: Telemetry broadcast (97% of traffic)
- `0x29`: Telemetry response
- `0x42`: Command message
- `0x55`: Pressure mode status

But the payload structure within these messages is still being documented. The device sends telemetry every ~1.8 seconds, and we can parse the framing, but the meaning of individual bytes is mostly unknown.

## What the Server Actually Does

Despite the limitations, the server provides real value:

### Accepts Device Connections

The server establishes TLS connections with jailbroken devices. This alone proves the certificate injection worked and the device trusts your CA.

### Logs All Messages

Every WebSocket frame is logged with:

- Timestamp
- Raw hex payload
- Parsed frame structure (if recognized)
- Message type identification

This data is invaluable for protocol research.

### Analysis Mode

Running with `--analysis-dir` writes structured logs:

```bash
./smartap-server server --analysis-dir ./captures --log-level debug
```

Output includes JSON Lines files with timestamped messages, hex dumps, and parsed frame metadata.

### Auto-Generated Certificates

The server embeds a Root CA and can generate server certificates on-demand. No external PKI required—certificates are kept in memory only. This zero-dependency design means you can run the server immediately after jailbreaking.

## Extending the Server

The server is designed for extension. Key files for development:

| File | Purpose |
|------|---------|
| `internal/server/websocket.go` | WebSocket frame handling |
| `internal/server/tls.go` | TLS configuration |
| `internal/server/http.go` | HTTP 101 response handling |
| `internal/protocol/parser.go` | Message parsing |
| `internal/protocol/frame.go` | Frame structure |
| `internal/protocol/types.go` | Message type definitions |

### Adding Protocol Support

When you identify a new message type:

1. Add a constant to `internal/protocol/types.go`
2. Add parsing logic to `internal/protocol/parser.go`
3. Add handling in `internal/server/websocket.go`
4. Document findings in `docs/technical/protocol.md`

### Testing Changes

```bash
# Build with your changes
make build

# Run with debug logging
./bin/smartap-server server --log-level debug

# Or with analysis logging
./bin/smartap-server server --analysis-dir ./captures
```

The most valuable contributions come from correlating physical device actions (pressing buttons, triggering valves) with the messages captured immediately before/after.

## Network Constraints

### Local Network Only

Current implementation is local-only:

- Device and server must be on same network
- No remote access from outside your home
- VPN required for external access

### DNS Dependency

The device hardcodes `evalve.smartap-tech.com`:

- Must configure local DNS override
- Router or Pi-hole configuration required
- Cannot change domain on device

## Stability

### Alpha Software

This is experimental software:

- **Expect bugs** - Not extensively tested
- **Breaking changes** - APIs may change
- **Limited testing** - Small user base
- **No guarantees** - Best effort support

### No Warranty

- Software provided as-is
- No guaranteed uptime
- Community-supported only
- Use at your own risk

## How to Help

The best way to reduce limitations is to help document the protocol:

### 1. Capture Traffic

Run the server with debug logging:

```bash
./smartap-server server --log-level debug
```

Share the message logs (sanitized of any personal data).

### 2. Analyze Messages

If you have reverse engineering skills:

- Examine the message format
- Identify command/response patterns
- Document findings

### 3. Submit Findings

- Open a GitHub issue with your analysis
- Create a pull request updating protocol docs
- Share in community discussions

### 4. Test Changes

When protocol code is updated:

- Test with your device
- Report success or failure
- Help verify implementations

## Future Improvements

Work is ongoing to:

- [ ] Document complete message format
- [ ] Implement basic commands
- [ ] Add device status queries
- [ ] Create simple web interface
- [ ] Add Home Assistant integration

Progress depends on community contributions to protocol understanding.

---

## Questions?

- [FAQ](../about/faq.md) - Common questions
- [Community](../about/community.md) - Get help
- [GitHub Issues](https://github.com/muurk/smartap/issues) - Report issues

---

[:material-arrow-left: Previous: Custom Certificates](custom-certificates.md){ .md-button }
[:material-home: Back to Overview](overview.md){ .md-button .md-button--primary }
