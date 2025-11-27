# Downloads

All Smartap tools are available as pre-built binaries. No installation required - just download and run.

---

## smartap-cfg - Device Configuration

Configure WiFi, outlets, and server settings on your Smartap device.

[:material-book-open: Documentation](getting-started/overview.md){ .md-button }

**This is what most users need.** If you just want your shower controls working, this is the only tool you need.

| Platform | Download |
|----------|----------|
| macOS Apple Silicon | [smartap-cfg-darwin-arm64](https://github.com/muurk/smartap/releases/latest/download/smartap-cfg-darwin-arm64) |
| macOS Intel | [smartap-cfg-darwin-amd64](https://github.com/muurk/smartap/releases/latest/download/smartap-cfg-darwin-amd64) |
| Windows x64 | [smartap-cfg-windows-amd64.exe](https://github.com/muurk/smartap/releases/latest/download/smartap-cfg-windows-amd64.exe) |
| Linux x86_64 | [smartap-cfg-linux-amd64](https://github.com/muurk/smartap/releases/latest/download/smartap-cfg-linux-amd64) |
| Linux ARM64 (Pi 4/5) | [smartap-cfg-linux-arm64](https://github.com/muurk/smartap/releases/latest/download/smartap-cfg-linux-arm64) |
| Linux ARMv7 (Pi 32-bit) | [smartap-cfg-linux-armv7](https://github.com/muurk/smartap/releases/latest/download/smartap-cfg-linux-armv7) |

---

## smartap-jtag - JTAG Operations

Inject certificates, detect firmware, and dump memory via JTAG.

[:material-book-open: Documentation](jailbreak/overview.md){ .md-button }

**Required for "jailbreaking" your device.** Enables your device to connect to your own server instead of the defunct cloud service.

| Platform | Download |
|----------|----------|
| macOS Apple Silicon | [smartap-jtag-darwin-arm64](https://github.com/muurk/smartap/releases/latest/download/smartap-jtag-darwin-arm64) |
| macOS Intel | [smartap-jtag-darwin-amd64](https://github.com/muurk/smartap/releases/latest/download/smartap-jtag-darwin-amd64) |
| Windows x64 | [smartap-jtag-windows-amd64.exe](https://github.com/muurk/smartap/releases/latest/download/smartap-jtag-windows-amd64.exe) |
| Linux x86_64 | [smartap-jtag-linux-amd64](https://github.com/muurk/smartap/releases/latest/download/smartap-jtag-linux-amd64) |
| Linux ARM64 (Pi 4/5) | [smartap-jtag-linux-arm64](https://github.com/muurk/smartap/releases/latest/download/smartap-jtag-linux-arm64) |
| Linux ARMv7 (Pi 32-bit) | [smartap-jtag-linux-armv7](https://github.com/muurk/smartap/releases/latest/download/smartap-jtag-linux-armv7) |

!!! note "Typically Run on Raspberry Pi"
    While available for all platforms, smartap-jtag is typically run on the Raspberry Pi that's connected to your device via JTAG. Download the ARM64 or ARMv7 version depending on your Pi's OS.

---

## smartap-server - WebSocket Server

Run your own server to receive connections from jailbroken devices.

[:material-book-open: Documentation](server/overview.md){ .md-button }

**Experimental.** The protocol is not fully documented yet. Running the server helps with protocol research.

| Platform | Download |
|----------|----------|
| macOS Apple Silicon | [smartap-server-darwin-arm64](https://github.com/muurk/smartap/releases/latest/download/smartap-server-darwin-arm64) |
| macOS Intel | [smartap-server-darwin-amd64](https://github.com/muurk/smartap/releases/latest/download/smartap-server-darwin-amd64) |
| Windows x64 | [smartap-server-windows-amd64.exe](https://github.com/muurk/smartap/releases/latest/download/smartap-server-windows-amd64.exe) |
| Linux x86_64 | [smartap-server-linux-amd64](https://github.com/muurk/smartap/releases/latest/download/smartap-server-linux-amd64) |
| Linux ARM64 (Pi 4/5) | [smartap-server-linux-arm64](https://github.com/muurk/smartap/releases/latest/download/smartap-server-linux-arm64) |
| Linux ARMv7 (Pi 32-bit) | [smartap-server-linux-armv7](https://github.com/muurk/smartap/releases/latest/download/smartap-server-linux-armv7) |

---

## Installation

### macOS

1. Download the appropriate binary
2. Make it executable:
   ```bash
   chmod +x smartap-cfg-darwin-arm64
   ```
3. macOS may block the app. Right-click → Open, or:
   ```bash
   xattr -d com.apple.quarantine smartap-cfg-darwin-arm64
   ```

### Windows

1. Download the `.exe` file
2. Windows Defender may warn - click "More info" → "Run anyway"
3. Run from Command Prompt or PowerShell

### Linux

1. Download the appropriate binary
2. Make it executable:
   ```bash
   chmod +x smartap-cfg-linux-amd64
   ```
3. Optionally move to PATH:
   ```bash
   sudo mv smartap-cfg-linux-amd64 /usr/local/bin/smartap-cfg
   ```

---

## Verifying Downloads

All releases are built via GitHub Actions from the public repository. You can verify the build by:

1. Checking the [GitHub Actions workflow](https://github.com/muurk/smartap/actions)
2. Building from source yourself (see [Development Setup](technical/development.md))

---

## Building from Source

If you prefer to build from source:

```bash
git clone https://github.com/muurk/smartap.git
cd smartap
make build
# Binaries are in ./bin/
```

Requires Go 1.21 or newer.

---

## Version History

See the [GitHub Releases](https://github.com/muurk/smartap/releases) page for version history and changelogs.
