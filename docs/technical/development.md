# Development Setup

Setting up a development environment for contributing to this project.

## Prerequisites

- Go 1.21 or newer
- Git
- Code editor (VS Code, GoLand, vim, etc.)
- Basic Go knowledge

## Clone Repository

```bash
git clone https://github.com/muurk/smartap.git
cd smartap
```

## Project Structure

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
│   ├── version/          # Version info
│   └── wizard/           # TUI wizard
├── pkg/                  # Public packages (future)
├── docs/                 # Documentation source (MkDocs)
├── hardware/             # OpenOCD configs, GDB scripts
└── Makefile              # Build automation
```

## Build

```bash
# Build all binaries
make build-all

# Build just the server
make build

# Build configuration tool
make build-cfg

# Build JTAG tool
make build-jtag

# Build all for production (optimized, smaller binaries)
make build-prod-all
```

Output goes to `bin/` directory.

## Architecture Deep Dive

This section explains how the code works, not just how to run commands.

### How the Tools Fit Together

The three CLI tools share internal packages but serve different purposes:

**smartap-cfg** communicates with devices via HTTP:

- `internal/discovery/` - mDNS scanning for `_http._tcp` services
- `internal/deviceconfig/` - HTTP client with retry, caching, malformed JSON handling
- `internal/wizard/` - Bubble Tea TUI components

**smartap-jtag** communicates via JTAG/GDB:

- `internal/gdb/` - GDB script execution and firmware detection
- `internal/gdb/scripts/` - Script implementations with Go templates
- `internal/gdb/firmwares/` - YAML firmware catalog
- `internal/ui/` - CLI progress indicators

**smartap-server** accepts device connections:

- `internal/server/` - TLS and WebSocket handling
- `internal/protocol/` - Binary protocol parsing
- `internal/logging/` - Structured logging

### The GDB Executor (`internal/gdb/`)

The JTAG tool doesn't hardcode GDB commands. Instead, it uses a templated script system that generates GDB commands at runtime.

**Key components:**

| File | Purpose |
|------|---------|
| `executor.go` | Manages GDB process lifecycle, renders templates, parses output |
| `scripts/script.go` | Defines the `Script` interface all operations implement |
| `scripts/*.go` | Individual script implementations |
| `scripts/templates/*.gdb.tmpl` | Go templates for GDB commands |
| `firmwares/firmwares.yaml` | Firmware catalog with function addresses |
| `firmwares/catalog.go` | Loads and queries firmware definitions |

**The Script interface:**

Every GDB operation implements this interface:

```go
type Script interface {
    Name() string                           // Human-readable name
    Template() string                       // GDB script template
    Params() map[string]interface{}         // Template parameters
    Parse(output string) (*Result, error)   // Parse GDB output
    Streaming() bool                        // Real-time output?
}
```

**How script execution works:**

1. Script implementation provides template and parameters
2. Executor renders template with Go's `text/template`
3. Rendered script written to temporary file
4. Executor spawns `arm-none-eabi-gdb -batch -nx -x script.gdb`
5. Output captured and parsed via `Script.Parse()`
6. Temporary file cleaned up
7. Structured `Result` returned with success/failure, steps, data

**Example: Certificate injection:**

```go
// From internal/gdb/scripts/inject_certs.go
script := NewInjectCertsScript(
    firmware,           // Contains function addresses
    certData,           // DER-encoded certificate bytes
    "/cert/129.der",    // Target path on device
    "localhost", 3333,  // OpenOCD connection
)
result, err := executor.Execute(ctx, script)
```

The template uses firmware function addresses to call SimpleLink SDK functions via GDB:

```
# Set registers for sl_FsDel call (ARM EABI)
set $r0 = {{.Firmware.Memory.FilenamePtr}}  # filename pointer
set $r1 = 0                                  # token
set $pc = {{.Firmware.Functions.SlFsDel}}   # function address
set $lr = {{.Firmware.Memory.StackBase}}    # return address
continue
```

### Firmware Catalog (`internal/gdb/firmwares/`)

The catalog maps firmware versions to function addresses. This is critical—wrong addresses corrupt devices.

**YAML Structure (`firmwares.yaml`):**

```yaml
firmwares:
  - version: "0x355"
    name: "Smartap 0x355"
    verified: true

    # SimpleLink SDK function addresses
    functions:
      sl_FsOpen: 0x20015c64
      sl_FsRead: 0x20014b54
      sl_FsWrite: 0x20014bf8
      sl_FsClose: 0x2001555c
      sl_FsDel: 0x20016ea8
      sl_FsGetInfo: 0x2001590c
      uart_log: 0x20014f14

      # Signatures for firmware detection
      signatures:
        sl_FsOpen: [0x4606b570, 0x78004818]
        sl_FsRead: [0x43f0e92d, 0x48254680]
        # ... more signatures

    # Memory layout for GDB operations
    memory:
      work_buffer: 0x20030000      # 64KB scratch space
      file_handle_ptr: 0x20031000  # Storage for file handle
      filename_ptr: 0x20031004     # Storage for filename string
      token_ptr: 0x20031020        # Storage for token
      stack_base: 0x20031d00       # Stack pointer for function calls
```

**How signature-based detection works:**

1. GDB reads 8 bytes at each function address
2. Compares against signatures for all known firmware versions
3. Calculates confidence: `(matching / total) × 100%`
4. Requires 100% match before allowing write operations
5. 7 signatures provide statistical certainty—one match might be coincidence, seven cannot be

### Device Configuration Client (`internal/deviceconfig/`)

This package handles HTTP communication with Smartap devices.

**Key challenges solved:**

**1. Malformed JSON responses**

The device returns JSON with trailing HTML garbage:

```
{"serial":"12345","dns":"smartap-tech.com"}</script></body></html>
```

`CleanJSONResponse()` in `models.go` tracks brace depth to extract valid JSON:

```go
// Finds the end of the JSON object by tracking { } depth
// Handles escaped strings correctly (braces inside strings don't count)
// Returns clean JSON without trailing garbage
```

**2. Retry with exponential backoff**

Device responses can be slow. The client in `client.go` retries with configurable backoff:

```go
client := NewClient(ip, port)
client.MaxRetries = 3
client.RetryDelay = 1 * time.Second
client.UseExponentialBackoff = true  // 1s, 2s, 4s
client.MaxRetryDelay = 30 * time.Second
```

**3. Configuration verification**

After updates, `VerifyConfiguration()` re-reads configuration to verify changes took effect. This catches silent failures.

### CC3200 Compatibility (`internal/server/`)

The CC3200's TLS implementation has quirks that required careful handling.

**TLS Cipher Suite Restrictions**

The CC3200 doesn't support modern cipher suites. From `tls.go`:

```go
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

No ECDHE, no GCM, no TLS 1.3. This isn't a flaw—it's a hardware constraint from 2014.

**Custom HTTP 101 Response**

The device firmware validates the WebSocket upgrade response using `strstr()`. Standard Go libraries add headers that break validation. From `http.go`:

```go
// Device expects EXACTLY this - no extra headers
response := "HTTP/1.1 101 Switching Protocols\r\n" +
    "Upgrade: websocket\r\n" +
    "Connection: Upgrade\r\n" +
    "\r\n"
conn.Write([]byte(response))
```

Standard libraries would add `Sec-WebSocket-Accept`, `Server`, `Date` headers. The device's `strstr()` validation fails with any variation. This was discovered through trial-and-error—the device simply closed connections until this exact format was used.

## Run Tests

```bash
# Run all tests
go test ./...

# With coverage
go test -cover ./...

# Verbose output
go test -v ./...
```

## Run Locally

```bash
# Run server (uses embedded CA by default)
./bin/smartap-server server

# Run server with custom certificates
./bin/smartap-server server --cert /path/to/cert.pem --key /path/to/key.pem

# Run server with debug logging
./bin/smartap-server server --log-level debug

# Run config tool (launches wizard by default)
./bin/smartap-cfg

# Run config tool with specific device
./bin/smartap-cfg wizard --device 192.168.4.16
```

## Development Workflow

1. Create feature branch
2. Make changes
3. Run tests
4. Test manually with real device (if possible)
5. Submit pull request

## Code Style

Follow standard Go conventions:

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run
```

## Documentation

Build and serve docs locally:

```bash
# Install mkdocs if needed
pip install mkdocs-material

# Serve docs
make docs-serve

# Build docs
make docs
```

## Debugging

### Server Debugging

```bash
# Run with debug logging
./bin/smartap-server server --log-level debug

# Use delve debugger
dlv debug ./cmd/smartap-server -- server --log-level debug
```

## Contributing

See [Contributing Guide](../contributing/overview.md) for detailed contribution guidelines.

---

[:material-arrow-left: Previous: Releasing](releasing.md){ .md-button }
[:material-arrow-right: Next: Contributing](../contributing/overview.md){ .md-button .md-button--primary }
