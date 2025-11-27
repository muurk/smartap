# Contributing

Welcome! Your contributions help keep abandoned IoT devices working.

## Ways to Contribute

### 1. Protocol Documentation

The most valuable contribution right now is helping document the device protocol.

**What you can do:**

- Run the server with debug logging
- Capture device messages
- Analyze message patterns
- Document findings

See [Protocol Research](../technical/protocol.md) for current knowledge.

### 2. Firmware Analysis

Help add support for new firmware versions.

**What you can do:**

- Submit memory dumps from unrecognized devices
- Analyze dumps with Ghidra or similar tools
- Identify SimpleLink function addresses
- Document firmware variations

See [Firmware Analysis](firmware-analysis.md) for detailed guidance.

### 3. Code Contributions

Improve the tools and server.

**Areas needing work:**

- Protocol implementation
- Server features
- CLI improvements
- Bug fixes

### 4. Testing

Test the software with real devices.

**What you can do:**

- Try features with your device
- Report bugs with detailed information
- Verify fixes work
- Test on different configurations

### 5. Documentation

Help improve these docs.

**What you can do:**

- Fix typos and errors
- Add examples and clarifications
- Report confusing sections
- Translate to other languages

## Getting Started

### 1. Fork and Clone

```bash
# Fork on GitHub, then:
git clone https://github.com/YOUR_USERNAME/smartap.git
cd smartap
```

### 2. Set Up Development Environment

```bash
# Install Go 1.21+
# Verify installation
go version

# Build all binaries
make build-all

# Binaries are output to ./bin/
```

### Make Targets Reference

| Target | Description |
|--------|-------------|
| `make build-all` | Build all three binaries (server, cfg, jtag) |
| `make build` | Build server binary only |
| `make build-cfg` | Build smartap-cfg only |
| `make build-jtag` | Build smartap-jtag only |
| `make build-prod-all` | Build all binaries optimized for production |
| `make test` | Run unit tests |
| `make test-coverage` | Run tests with coverage report |
| `make lint` | Run golangci-lint (must be installed) |
| `make fmt` | Format code with go fmt |
| `make tidy` | Tidy go.mod dependencies |
| `make clean` | Remove build artifacts |
| `make docs` | Build documentation with mkdocs |
| `make docs-serve` | Start local documentation server at http://127.0.0.1:8000 |
| `make check-gdb` | Verify GDB prerequisites are installed |
| `make help` | Show all available targets |

Run `make help` for the complete list with descriptions.

### 3. Create a Branch

```bash
git checkout -b feature/your-feature-name
```

### 4. Make Changes

- Follow existing code style
- Add tests for new features
- Update documentation if needed

### 5. Test

```bash
# Run tests
go test ./...

# Test with actual device if possible
```

### 6. Commit

Use clear commit messages:

```bash
git add .
git commit -m "feat: add temperature monitoring support"
```

Prefix types:

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation only
- `test:` - Test additions
- `refactor:` - Code refactoring

### 7. Create Pull Request

```bash
git push origin feature/your-feature-name
```

Then create a Pull Request on GitHub with:

- What the PR does
- Why it's needed
- How it was tested

## Code Style

### Go Code

- Run `go fmt` before committing
- Follow standard Go conventions
- Keep functions focused and small
- Add comments for non-obvious logic

### Documentation

- Use clear, simple language
- Test all code examples
- Include expected output where helpful

## Development Workflow

### Making Changes

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following existing code patterns

3. **Test locally**:
   ```bash
   make test
   make build-all
   # Test the binary manually if applicable
   ```

4. **Format and lint**:
   ```bash
   make fmt
   make lint  # Fix any issues reported
   ```

5. **Commit with conventional commit messages**:
   ```bash
   git commit -m "feat: add temperature monitoring support"
   ```

### Testing with Hardware

If your changes affect JTAG operations:

1. You'll need a Raspberry Pi with JTAG connection to a Smartap device
2. Run `make check-gdb` to verify prerequisites
3. Test against a real device before submitting

### Documentation Changes

If updating documentation:

```bash
make docs-serve
# Open http://127.0.0.1:8000 and verify your changes render correctly
```

### Project Structure

```
smartap/
├── cmd/
│   ├── smartap-cfg/      # Configuration TUI - uses Bubble Tea
│   ├── smartap-jtag/     # JTAG tool - uses internal/gdb package
│   └── smartap-server/   # WebSocket server
├── internal/
│   ├── gdb/              # GDB scripting and execution
│   │   ├── scripts/      # GDB script builders
│   │   └── firmware/     # Firmware catalog (YAML definitions)
│   ├── deviceconfig/     # HTTP client for device configuration
│   ├── discovery/        # mDNS device discovery
│   ├── server/           # WebSocket server implementation
│   ├── wizard/           # TUI wizard components
│   └── ui/               # Shared UI components
├── docs/                 # MkDocs documentation source
└── hardware/             # OpenOCD configurations
```

## Understanding the Codebase

Before diving into code, understand the key patterns and design decisions.

### The Internal Packages

```
internal/
├── gdb/                 # JTAG/GDB operations
│   ├── executor.go      # Runs GDB with templated scripts
│   ├── scripts/         # GDB script implementations
│   │   ├── script.go    # Script interface definition
│   │   ├── inject_certs.go
│   │   ├── detect_firmware.go
│   │   └── templates/   # .gdb.tmpl template files
│   └── firmwares/       # Firmware catalog
│       ├── catalog.go   # YAML loader and queries
│       └── firmwares.yaml
├── deviceconfig/        # HTTP device configuration
│   ├── client.go        # HTTP client with retry logic
│   ├── models.go        # Configuration data structures
│   └── rollback.go      # Transactional updates
├── discovery/           # mDNS device discovery
├── server/              # WebSocket server
│   ├── server.go        # Main server loop
│   ├── websocket.go     # Frame handling
│   ├── http.go          # HTTP 101 response
│   └── tls.go           # CC3200-compatible TLS
├── protocol/            # Binary protocol parsing
│   ├── frame.go         # Frame structure
│   ├── parser.go        # Message parsing
│   └── types.go         # Message type constants
├── wizard/              # TUI components (Bubble Tea)
└── ui/                  # CLI output formatting
```

### Design Patterns Used

**Template-based GDB scripting**: Instead of hardcoding GDB commands, we use Go templates (`text/template`). This makes operations configurable and testable. See `internal/gdb/scripts/script.go` for the `Script` interface that all GDB operations implement.

**Embedded assets**: Certificates, GDB scripts, and firmware catalogs are embedded via `//go:embed`. No external files required at runtime.

**Structured logging**: All components use `internal/logging` for consistent, leveled output via zap.

**Malformed response handling**: The device returns JSON with trailing HTML garbage. `CleanJSONResponse()` in `internal/deviceconfig/models.go` tracks brace depth to extract valid JSON.

## High-Impact Contribution Areas

Not all contributions are equal. Here's where help is most needed:

### 1. Protocol Documentation (Critical)

The binary protocol is only partially understood. Every new field documented helps everyone.

**How to contribute:**

1. Run the server with `--analysis-dir ./captures`
2. Perform physical actions (press buttons, trigger valves)
3. Capture the messages immediately before/after
4. Document which bytes changed
5. Submit findings to `docs/technical/protocol.md`

Even partial findings help. "Byte 15 changes when I press button 2" is valuable data.

### 2. Firmware Support (High Impact)

Only firmware 0x355 is currently supported. Many devices have different versions.

**How to contribute:**

1. If `detect-firmware` doesn't recognize your device, run `dump-memory`
2. Analyze the dump with Ghidra (ARM Cortex-M4, base 0x20000000)
3. Find the SimpleLink functions: `sl_FsOpen`, `sl_FsWrite`, `sl_FsClose`, `sl_FsDel`
4. Capture signatures (first 8 bytes of each function)
5. Add an entry to `internal/gdb/firmwares/firmwares.yaml`
6. Submit a PR with the memory dump and your analysis

See [Firmware Analysis Guide](firmware-analysis.md) for detailed Ghidra techniques.

### 3. Server Features (Medium Impact)

The server accepts connections but doesn't do much with them yet.

**Valuable additions:**

- Message response handling (when we understand the protocol)
- Device state tracking
- REST API for external integrations
- Home Assistant integration
- MQTT bridge

### 4. Testing (Always Valuable)

We have limited test coverage. Every test helps:

- Unit tests for protocol parsing
- Integration tests for device configuration
- End-to-end tests (requires hardware)

## The Firmware Catalog

The firmware catalog (`internal/gdb/firmwares/firmwares.yaml`) is critical infrastructure.

### Structure

```yaml
firmwares:
  - version: "0x355"           # Version identifier (from device)
    name: "Smartap 0x355"      # Human-readable name
    verified: true             # Tested with real device

    functions:                  # SimpleLink SDK addresses
      sl_FsOpen: 0x20015c64
      sl_FsRead: 0x20014b54
      sl_FsWrite: 0x20014bf8
      sl_FsClose: 0x2001555c
      sl_FsDel: 0x20016ea8
      sl_FsGetInfo: 0x2001590c
      uart_log: 0x20014f14

      signatures:               # Detection signatures
        sl_FsOpen: [0x4606b570, 0x78004818]
        sl_FsRead: [0x43f0e92d, 0x48254680]
        # ... more signatures

    memory:                     # Memory layout
      work_buffer: 0x20030000   # Safe scratch space
      file_handle_ptr: 0x20031000
      filename_ptr: 0x20031004
      token_ptr: 0x20031020
      stack_base: 0x20031d00
```

### Adding a New Firmware Version

1. **Get function addresses** from Ghidra analysis
2. **Capture signatures** using GDB:
   ```
   (gdb) x/2xw 0x20015c64
   0x20015c64: 0x4606b570  0x78004818
   ```
   The two 32-bit words form the signature: `[0x4606b570, 0x78004818]`
3. **Test thoroughly** before submitting
4. **Include your memory dump** with the PR for verification

### Why 7 Signatures?

Statistical confidence. A single signature might match by coincidence. Seven independent signatures matching is statistically certain. The functions are spread across memory, so partial matches indicate either corruption or an unknown firmware variant.

## Areas Needing Help

### High Priority

- [ ] Complete protocol documentation
- [ ] Implement device control commands
- [ ] Add more firmware versions
- [ ] Create automated tests

### Medium Priority

- [ ] Web UI for device control
- [ ] Home Assistant integration
- [ ] Improve error messages
- [ ] Add usage examples

### Good First Issues

Check [GitHub Issues](https://github.com/muurk/smartap/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22) for beginner-friendly tasks.

## Communication

### GitHub Discussions

**[github.com/muurk/smartap/discussions](https://github.com/muurk/smartap/discussions)**

Best for:

- Questions and help requests
- Protocol research coordination
- Ideas and feature brainstorming
- Sharing your setup or findings

### GitHub Issues

**[github.com/muurk/smartap/issues](https://github.com/muurk/smartap/issues)**

Best for:

- Bug reports (something is broken)
- Well-defined feature requests
- Documentation errors

### Pull Requests

Best for:

- Code changes
- Documentation updates
- Any contribution

### Be Respectful

- Be kind and patient
- Assume good intentions
- Provide constructive feedback
- Welcome newcomers

## Recognition

Contributors are:

- Listed in release notes
- Credited in documentation
- Thanked publicly

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.

## Questions?

Open a GitHub issue or ask in the community discussions. We're happy to help!

---

[:material-arrow-right: Next: Firmware Analysis](firmware-analysis.md){ .md-button .md-button--primary }
