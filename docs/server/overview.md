# Running the Server

Running your own Smartap WebSocket server.

!!! danger "Experimental Software"
    The smartap-server is experimental and largely built on initial guesswork from memory dump analysis. The device communication protocol is not fully understood. The server can establish connections and log messages, but meaningful bidirectional control is still under development.

    **Your contributions to protocol documentation are what will make this functional.**

## What the Server Does

The Smartap server acts as a replacement for the original cloud infrastructure:

- **Accepts TLS connections** from Smartap devices
- **Maintains WebSocket sessions** for bidirectional communication
- **Logs all messages** for protocol analysis and debugging
- **Auto-generates certificates** signed by the embedded Root CA

## What the Server Doesn't Do (Yet)

These features are not implemented or incomplete:

- :material-close: **Full remote control** - Protocol not fully understood
- :material-close: **Temperature monitoring** - Message format unknown
- :material-close: **Scheduling** - Requires protocol implementation
- :material-close: **Mobile app** - No replacement app exists
- :material-close: **Voice control** - Requires protocol implementation

## Why Run a Server?

There are two main reasons to run the server:

### 1. Protocol Research

The primary reason right now is to help document the communication protocol. When your device connects:

- All messages are logged
- You can analyze the message format
- Your findings help everyone

### 2. Future Remote Control

Once the protocol is documented, the server will enable:

- Remote shower pre-heating
- Outlet activation
- Temperature monitoring
- Home automation integration

## Prerequisites

Before setting up the server, you must have:

1. **Jailbroken device** - Certificate injection completed via [smartap-jtag](../jailbreak/overview.md)
2. **Network access** - Server must be reachable from the device
3. **DNS configuration** - `evalve.smartap-tech.com` must resolve to your server

!!! warning "Jailbreak Required"
    The server is useless without first injecting the CA certificate into your device. The device will reject connections to any server not signed by its trusted CA.

    See [Device Jailbreak](../jailbreak/overview.md) to get started.

## Available Tools

| Tool | Purpose |
|------|---------|
| `smartap-server` | WebSocket server for device connections |
| `smartap-cfg` | Device configuration (WiFi, outlets, DNS) |
| `smartap-jtag` | JTAG operations (certificate injection) |

## Next Steps

If you've completed the jailbreak process:

[:material-arrow-right: Quick Start Guide](quick-start.md){ .md-button .md-button--primary }

If you haven't jailbroken your device yet:

[:material-arrow-right: Device Jailbreak Guide](../jailbreak/overview.md){ .md-button }

---

## Is This Worth It?

Be honest with yourself:

**Run the server if you:**

- Want to help document the protocol
- Enjoy reverse engineering projects
- Are comfortable with experimental software
- Can contribute findings back to the project

**Don't run the server if you:**

- Just want basic outlet control (use [smartap-cfg](../getting-started/wifi-setup.md) instead)
- Expect production-ready remote control
- Don't have time to troubleshoot
- Aren't interested in protocol research

---

[:material-arrow-right: Next: Quick Start](quick-start.md){ .md-button .md-button--primary }
