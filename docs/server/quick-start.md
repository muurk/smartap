# Server Quick Start

Get the smartap-server running in minutes.

## Download

Download the appropriate binary for your server:

| Platform | Download |
|----------|----------|
| Linux x86_64 | [smartap-server-linux-amd64](https://github.com/muurk/smartap/releases/latest/download/smartap-server-linux-amd64) |
| Linux ARM64 (Pi 4/5) | [smartap-server-linux-arm64](https://github.com/muurk/smartap/releases/latest/download/smartap-server-linux-arm64) |
| Linux ARMv7 (Pi 32-bit) | [smartap-server-linux-armv7](https://github.com/muurk/smartap/releases/latest/download/smartap-server-linux-armv7) |
| macOS Apple Silicon | [smartap-server-darwin-arm64](https://github.com/muurk/smartap/releases/latest/download/smartap-server-darwin-arm64) |
| macOS Intel | [smartap-server-darwin-amd64](https://github.com/muurk/smartap/releases/latest/download/smartap-server-darwin-amd64) |
| Windows x64 | [smartap-server-windows-amd64.exe](https://github.com/muurk/smartap/releases/latest/download/smartap-server-windows-amd64.exe) |

```bash
# Example for Linux ARM64 (Raspberry Pi 4/5)
wget https://github.com/muurk/smartap/releases/latest/download/smartap-server-linux-arm64
chmod +x smartap-server-linux-arm64
mv smartap-server-linux-arm64 smartap-server
```

## Start the Server

### Basic Usage (Auto-Generated Certificate)

The simplest way to start is with auto-generated certificates:

```bash
# Start server with auto-generated certificate (signed by embedded Root CA)
./smartap-server server
```

This will:

1. Generate a TLS certificate signed by the embedded Root CA
2. Listen on port 443 (requires root/sudo)
3. Log all WebSocket messages

!!! note "Port 443 Requires Root"
    To bind to port 443, run with sudo:
    ```bash
    sudo ./smartap-server server
    ```
    Or use a higher port with `--port 8443` and configure port forwarding.

### Custom Port

```bash
# Run on a non-privileged port
./smartap-server server --port 8443
```

### Debug Logging

```bash
# Enable debug logging for protocol analysis
./smartap-server server --log-level debug
```

### Custom Certificates

If you have your own certificates:

```bash
# Start with custom certificates
./smartap-server server --cert /path/to/fullchain.pem --key /path/to/privkey.pem
```

See [Custom Certificates](custom-certificates.md) for details.

## Command Reference

```
smartap-server server [flags]

Flags:
  --cert string       Path to TLS certificate file (optional, will auto-generate if not provided)
  --key string        Path to TLS private key file (optional, will auto-generate if not provided)
  --host string       Server hostname (empty = listen on all interfaces)
  --port int          Server port (default 443)
  --log-level string  Log level: debug, info, warn, error (default "info")
```

## Configure DNS

Your device tries to connect to `evalve.smartap-tech.com`. You need to redirect this to your server.

### Option A: Router DNS Override (Recommended)

1. Log into your router admin interface
2. Find "Custom DNS" or "DNS Override" settings
3. Add entry: `evalve.smartap-tech.com` → `<your-server-ip>`
4. Also add: `smartap-tech.com` → `<your-server-ip>`
5. Save and apply

### Option B: Pi-hole

If running Pi-hole:

```bash
# Add to /etc/pihole/custom.list
192.168.1.100 evalve.smartap-tech.com
192.168.1.100 smartap-tech.com
```

Then restart DNS:
```bash
pihole restartdns
```

### Verify DNS

From another device on your network:

```bash
nslookup evalve.smartap-tech.com
# Should return your server's IP
```

## Test Device Connection

1. **Start the server** with debug logging:
   ```bash
   sudo ./smartap-server server --log-level debug
   ```

2. **Power cycle your device** (the one with injected certificate)

3. **Watch the logs** for connection:
   ```
   INFO  Starting Smartap WebSocket Server
   INFO  Listening on :443
   INFO  Device connected from 192.168.1.x
   DEBUG TLS handshake successful
   DEBUG WebSocket connection established
   DEBUG Received message: [hex data]
   ```

## Expected Output

When everything works, you'll see:

```
╭─────────────────────────────────────────────────────────────────╮
│ Smartap WebSocket Server                                        │
│ smartap-server server                                           │
├─────────────────────────────────────────────────────────────────┤
│ Host:      0.0.0.0                                              │
│ Port:      443                                                  │
│ TLS:       Auto-generated (embedded Root CA)                    │
│ Log Level: debug                                                │
╰─────────────────────────────────────────────────────────────────╯

INFO  Starting Smartap WebSocket Server
INFO  Auto-generated server certificate signed by embedded Root CA
INFO  Listening on :443
```

## Troubleshooting

### "Permission denied" on port 443

```bash
# Run with sudo
sudo ./smartap-server server

# Or use non-privileged port
./smartap-server server --port 8443
```

### Device doesn't connect

- Verify DNS resolution: `nslookup evalve.smartap-tech.com`
- Ensure server is reachable from device's network
- Check firewall allows port 443/8443
- Verify certificate was successfully injected

### TLS handshake fails

- Device may have wrong certificate - re-run inject-certs
- Certificate domain mismatch - check server logs for details

### Device connects but immediately disconnects

- This may be normal - protocol not fully understood
- Check server logs for any error messages
- Report findings to help with protocol documentation!

## Running as a Service

For production use, run as a systemd service:

```ini
# /etc/systemd/system/smartap-server.service
[Unit]
Description=Smartap WebSocket Server
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/smartap-server server --log-level info
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable smartap-server
sudo systemctl start smartap-server
```

---

[:material-arrow-left: Previous: Overview](overview.md){ .md-button }
[:material-arrow-right: Next: Custom Certificates](custom-certificates.md){ .md-button .md-button--primary }
