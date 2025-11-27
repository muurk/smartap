# System Architecture

Technical overview of the system.

## Component Overview

The system has three main components:

1. **Smartap Device** - The CC3200-based shower controller running proprietary firmware
2. **smartap-server** - A replacement WebSocket server that devices connect to
3. **Configuration Tools** - `smartap-cfg` for device configuration, `smartap-jtag` for certificate injection

These components work together: after injecting a custom CA certificate via JTAG, the device trusts your server's TLS certificate and can establish a connection.

## The Smartap Device

### Hardware

**Main Controller:** Texas Instruments CC3200

- ARM Cortex-M4 processor
- Integrated WiFi (2.4GHz only)
- 256KB RAM
- 1MB Flash storage
- Built-in TLS/SSL support

**Interfaces:**

- JTAG (unpopulated header)
- HTTP server (WiFi configuration)
- TLS client (for cloud communication)

### Firmware

The device runs proprietary firmware based on TI's CC3200 SDK:

- SimpleLink WiFi stack
- SSL/TLS client implementation
- Certificate-based authentication
- WebSocket protocol for device communication

**Certificate Store:**

The device's flash filesystem contains:

- `/cert/129.der` - CA certificate (replaced by custom CA)
- `/cert/130.der` - Client public key (mutual TLS)
- `/cert/131.der` - Client private key (mutual TLS)

**Firmware Support:**

The tools need specific memory addresses for each firmware version. Firmware definitions are stored in `internal/gdb/firmwares/firmwares.yaml`. If your firmware version isn't recognized, see [How It Works: Adding New Firmware](../how-it-works/adding-firmware.md) or the [Firmware Analysis Guide](../contributing/firmware-analysis.md).

### Communication Protocol

**Connection Sequence:**

1. Device boots and connects to WiFi
2. DNS lookup for `evalve.smartap-tech.com`
3. TLS connection established (validates server cert against CA)
4. WebSocket upgrade
5. Device authenticates with server
6. Bi-directional message exchange begins

**Message Format:**

Messages use an unknown binary frame structure over WebSocket. The protocol is NOT JSON-based. Message format analysis is ongoing - see [Protocol Documentation](protocol.md) for current understanding.

## The Revival Server

### Purpose

The server replaces the original cloud infrastructure:

- Accepts TLS connections from devices
- Handles WebSocket communication
- Stores device state
- Provides API for control/monitoring
- (Future) integrates with home automation

### Implementation

**Language:** Go (Golang)

**Project Structure:**

```
smartap/
├── cmd/
│   ├── smartap-cfg/      # Device configuration tool (TUI)
│   ├── smartap-jtag/     # JTAG operations (cert injection, memory dump)
│   └── smartap-server/   # WebSocket server
├── internal/
│   ├── config/           # Configuration parsing
│   ├── deviceconfig/     # Device configuration client
│   ├── discovery/        # mDNS device discovery
│   ├── gdb/              # GDB scripting for JTAG
│   ├── logging/          # Logging utilities
│   ├── protocol/         # Protocol implementation
│   ├── server/           # Server implementation
│   ├── ui/               # UI components
│   ├── urls/             # URL handling
│   ├── version/          # Version info
│   └── wizard/           # TUI wizard
├── pkg/
│   └── smartap/          # Public API (future)
├── docs/                 # Documentation source (MkDocs)
└── hardware/             # OpenOCD configs, GDB scripts
```

**Key Features:**

- TLS termination with custom certificates
- WebSocket server
- Device registry and state management
- Protocol message parsing/routing
- Configuration tool (TUI)

### Network Architecture

**DNS Redirection:**

The device attempts to connect to `evalve.smartap-tech.com`. Since the original servers are offline, you must configure your local DNS (via router, Pi-hole, or hosts file) to resolve this domain to your server's IP address.

This allows the device to find your server without modifying the device's hardcoded server address (though `smartap-cfg set-server` can also change the server address directly on the device).

**TLS Trust Chain:**

The device validates the server's TLS certificate against its stored CA certificate. After certificate injection, the device trusts certificates signed by your custom CA instead of the original Comodo CA.

## Certificate Architecture

### Trust Chain Replacement

**Original (Now Defunct):**

```
COMODO Root CA
    └── COMODO RSA DV Secure Server CA
            └── *.smartap-tech.com (expired)
```

**Revival System:**

```
Custom Root CA (on device)
    └── Server Certificate (on server)
        CN: *.smartap-tech.com or evalve.smartap-tech.com
```

### Certificate Generation

Certificates generated using OpenSSL:

1. **Root CA** - 4096-bit RSA, 10-year validity
2. **Server Cert** - 2048-bit RSA, 2-year validity
3. Wildcard or specific SAN for smartap-tech.com domain

See [Certificate Details](certificate-details.md) for specifics.

### Certificate Deployment

**Server Side:** Standard TLS configuration

**Device Side:** JTAG-based flashing

- OpenOCD for JTAG communication
- GDB to call filesystem functions
- Direct memory manipulation to replace `/cert/129.der`

## Configuration Tool

### Purpose

User-friendly TUI for device configuration:

- Network discovery
- WiFi setup
- Outlet configuration
- Server address configuration

### Implementation

**Language:** Go with Bubble Tea TUI framework

**Features:**

- Auto-discovery of devices
- Interactive forms for configuration
- Real-time validation
- Works over both direct connection and network

## Development Tools

### JTAG Setup

**Hardware:** Raspberry Pi connected to device JTAG pins

**Software Stack:**

```
┌──────────────────────────────┐
│ GDB (arm-none-eabi-gdb)      │  ← Certificate flashing scripts
├──────────────────────────────┤
│ OpenOCD                      │  ← JTAG protocol handler
├──────────────────────────────┤
│ Linux GPIO (sysfsgpio)       │  ← GPIO driver
├──────────────────────────────┤
│ Raspberry Pi Hardware        │  ← Physical JTAG interface
└──────────────────────────────┘
```

### Analysis Tools

**Static Analysis:**

- Ghidra - Decompilation and memory analysis
- Binary analysis of memory dumps
- Function identification and naming

**Dynamic Analysis:**

- GDB breakpoints and watchpoints
- Memory dumps during operation
- Protocol capture (smartap-server)

## Security Considerations

### Device Security

**Preserved:**

- TLS encryption for all communication
- Certificate validation (now validating your CA)
- No backdoors or weakened security

**Changed:**

- Trust anchor replaced (custom CA instead of COMODO)
- All traffic stays on local network
- No cloud exposure

### Network Security

**Recommendations:**

- Use strong WiFi password
- Segment IoT devices on separate VLAN (optional)
- Firewall rules to block internet access from device
- Regular server software updates

### Certificate Security

**Best Practices:**

- Protect CA private key (offline storage recommended)
- Server certificates with reasonable expiry
- No certificate reuse across installations
- Document certificate locations

## Future Architecture

### Planned Components

**Home Assistant Integration:**

```
Home Assistant
    └── Smartap Integration (HACS)
            └── smartap-server API
                    └── Devices
```

**MQTT Support:**

```
smartap-server ←→ MQTT Broker ←→ Various Clients
```

**Web UI:**

```
Browser → Web UI (React/Vue) → REST API → smartap-server
```

## Performance Characteristics

### Resource Usage

**Server:**

- RAM: <100MB typical
- CPU: Minimal (<5% on Raspberry Pi 4)
- Network: Low bandwidth (kilobytes/sec per device)
- Storage: <50MB for application + logs

**Device:**

- Response time: <500ms typical for commands
- Connection stability: Hours/days between reconnects
- WiFi signal: Dependent on installation location

## Scalability

**Current:**

- Tested with 1-2 devices
- Designed for single-home use (1-10 devices)
- Single server instance

**Future:**

- Multi-device coordination
- Distributed server deployments
- Cloud-optional architecture

---

For implementation details, see:

- [How It Works](../how-it-works/overview.md) - Understanding the mechanics
- [Research Background](research-background.md) - How we got here
- [Protocol Documentation](protocol.md) - Message formats
- [Hardware Access](hardware-access.md) - JTAG details
- [Certificate Details](certificate-details.md) - Certificate specifics

---

[:material-arrow-right: Next: Research Background](research-background.md){ .md-button .md-button--primary }
