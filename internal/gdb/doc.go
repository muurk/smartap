// Package gdb provides native GDB integration for CC3200 device operations.
//
// This package enables low-level device operations via arm-none-eabi-gdb and OpenOCD,
// including certificate injection, log capture, memory dumping, and file operations.
// It's designed for the smartap-cfg utility to provide convenient device provisioning
// and debugging capabilities.
//
// # Architecture
//
// The package follows a script-based architecture where GDB operations are defined
// as template scripts that get parameterized and executed:
//
//	┌─────────────────┐
//	│ CLI Command     │
//	│ (smartap-cfg)   │
//	└────────┬────────┘
//	         │
//	         v
//	┌─────────────────┐
//	│ Script          │  Implements: Name(), Template(), Params(), Parse()
//	│ (InjectCerts)   │
//	└────────┬────────┘
//	         │
//	         v
//	┌─────────────────┐
//	│ Executor        │  Renders template, executes GDB, cleans up
//	│ (GDB Runner)    │
//	└────────┬────────┘
//	         │
//	         v
//	┌─────────────────┐
//	│ Parser          │  Extracts structured results from GDB output
//	│ (Regex-based)   │
//	└────────┬────────┘
//	         │
//	         v
//	┌─────────────────┐
//	│ Result          │  Structured result with success/failure, steps, data
//	└─────────────────┘
//
// # Core Components
//
// Executor: Runs GDB scripts via os/exec with timeout and error handling
//
//	config := gdb.Config{
//	    GDBPath:      "arm-none-eabi-gdb",
//	    OpenOCDHost:  "localhost",
//	    OpenOCDPort:  3333,
//	    Timeout:      5 * time.Minute,
//	}
//	executor := gdb.NewExecutor(config, logger)
//	result, err := executor.Execute(ctx, script)
//
// Scripts: Implement Script interface for specific operations
//
//	type Script interface {
//	    Name() string                              // Human-readable name
//	    Template() string                          // GDB script template
//	    Params() map[string]interface{}            // Template parameters
//	    Parse(output string) (*Result, error)      // Parse GDB output
//	}
//
// Firmware Catalog: Maps firmware versions to function addresses
//
//	db, _ := gdb.LoadFirmwares()
//	firmware, ok := db.Get("0x355")
//	addr := firmware.Functions.sl_FsOpen  // 0x20015c64
//
// Certificate Manager: Manages embedded and custom certificates
//
//	certMgr := gdb.NewCertManager()
//	rootCA, _ := certMgr.GetRootCA("der")
//	serverCert, _ := certMgr.GenerateServerCert(params)
//
// # GDB Scripts
//
// Scripts are Go text templates with access to:
//   - Firmware: Function addresses and memory locations
//   - OpenOCD: Host and port configuration
//   - Script-specific params: Certificate data, filenames, sizes, etc.
//
// Example template:
//
//	target extended-remote {{.OpenOCDHost}}:{{.OpenOCDPort}}
//	monitor reset halt
//	set $handle = 0
//	call {{.Firmware.Functions.sl_FsOpen}}(...)
//	print $handle
//	quit
//
// Templates are embedded using //go:embed and rendered at execution time.
//
// # Certificate Injection
//
// The primary use case is injecting custom CA certificates into device flash:
//
//	script := &InjectCertsScript{
//	    firmware:     firmware,
//	    certData:     certDER,
//	    targetFile:   "/cert/129.der",
//	    openocdHost:  "localhost",
//	    openocdPort:  3333,
//	}
//	result, err := executor.Execute(ctx, script)
//
// The injection workflow:
//  1. Halt device via OpenOCD
//  2. Setup filename in device memory
//  3. Load certificate to work buffer
//  4. Delete old certificate (sl_FsDel)
//  5. Create new file (sl_FsOpen with create flags)
//  6. Write certificate data (sl_FsWrite)
//  7. Close file (sl_FsClose)
//  8. Resume device
//
// # Firmware Version Support
//
// The package uses signature-based detection to reliably identify firmware versions.
// Instead of reading a version number from a fixed memory location (which varies),
// detection works by reading the first 8 bytes at known function addresses and
// comparing them against stored signatures in the firmware catalog.
//
// The catalog maintains function addresses and signatures for each known firmware:
//
//	firmwares:
//	  - version: "0x355"
//	    name: "CC3200 ServicePack 1.32.0"
//	    functions:
//	      sl_FsOpen:    0x20015c64
//	      sl_FsWrite:   0x20014bf8
//	      sl_FsClose:   0x2001555c
//	      sl_FsDel:     0x20016ea8
//	      sl_FsGetInfo: 0x2001590c
//	      uart_log:     0x20014f14
//	    signatures:
//	      sl_FsOpen:    [0x4606b570, 0x78004818]
//	      sl_FsRead:    [0x43f0e92d, 0x48254680]
//	      sl_FsWrite:   [0x43f0e92d, 0x48244680]
//	      sl_FsClose:   [0x460db5f0, 0x461c4607]
//	      sl_FsDel:     [0x4604b510, 0x78004814]
//	      sl_FsGetInfo: [0x481d4603, 0x7800b530]
//	      uart_log:     [0x1c04b510, 0xe003d007]
//
// Detection calculates confidence as (matches/total × 100%). All GDB operations
// require 100% confidence to prevent device damage from incorrect addresses.
//
// Unknown firmware versions are detected and the user is guided to submit a memory
// dump for analysis. This allows the community to expand firmware support over time.
//
// # Error Handling
//
// The package defines specific error types for different failure modes:
//   - GDBExecutionError: GDB command failed (exit code, stderr)
//   - GDBConnectionError: Cannot connect to OpenOCD
//   - GDBParseError: Failed to parse GDB output
//   - FirmwareUnsupportedError: Unknown firmware version
//
// All errors include context and can be unwrapped with errors.Unwrap().
//
// # Embedded Assets
//
// The package embeds several assets using //go:embed:
//   - Root CA certificate (PEM, DER, private key)
//   - GDB script templates (*.gdb.tmpl)
//   - Firmware catalog (firmwares.yaml)
//
// This enables zero-dependency distribution - the smartap-cfg binary contains
// everything needed for certificate injection.
//
// # Progress Reporting
//
// Scripts can report progress via step markers in GDB output:
//
//	echo [1/6] Halting device...\n
//	# ... GDB commands ...
//	echo [2/6] Setting up filename...\n
//
// The parser extracts these markers and updates progress in real-time.
//
// # Prerequisites
//
// The package requires:
//   - arm-none-eabi-gdb: GNU ARM Embedded GDB
//   - OpenOCD: Running and connected to device via JTAG
//   - CC3200 device: Smartap controller with JTAG connection
//
// Use ValidatePrerequisites() to check for these before operations.
//
// # Security Considerations
//
// The package embeds the Smartap Revival Project Root CA for convenience.
// Security-conscious users can provide custom certificates via CLI flags:
//
//	smartap-cfg gdb inject-certs --cert-file /path/to/custom-ca.der
//
// The embedded Root CA is intended for community development and testing.
// Production deployments should consider using organization-specific CAs.
//
// # Thread Safety
//
// The package is designed for single-threaded use. Each operation acquires
// exclusive access to the device via GDB. Concurrent operations will fail
// when trying to connect to OpenOCD.
//
// # Testing
//
// Unit tests mock GDB execution using test fixtures in testdata/:
//   - Mock GDB output for parser testing
//   - Mock firmware catalog for version lookup
//   - Mock certificates for injection workflow
//
// Integration tests require actual hardware with OpenOCD running.
// Mark these with: // +build integration
//
// # Adding New Operations
//
// To add a new GDB operation:
//
//  1. Create template in templates/my_operation.gdb.tmpl
//  2. Embed template: //go:embed templates/my_operation.gdb.tmpl
//  3. Create script type in scripts/my_operation.go
//  4. Implement Script interface (Name, Template, Params, Parse)
//  5. Add CLI command in cmd/smartap-cfg/gdb_commands.go
//  6. Add tests in my_operation_test.go
//
// The executor handles all boilerplate (temp files, execution, cleanup, error handling).
//
// # Example: Complete Certificate Injection
//
//	// Setup
//	logger, _ := zap.NewDevelopment()
//	config := gdb.Config{
//	    GDBPath:      "arm-none-eabi-gdb",
//	    OpenOCDHost:  "localhost",
//	    OpenOCDPort:  3333,
//	    Timeout:      5 * time.Minute,
//	}
//	executor := gdb.NewExecutor(config, logger)
//	certMgr := gdb.NewCertManager()
//
//	// Get certificate
//	rootCA, _ := certMgr.GetRootCA("der")
//
//	// Detect firmware
//	detectScript := &DetectFirmwareScript{
//	    openocdHost: "localhost",
//	    openocdPort: 3333,
//	}
//	result, _ := executor.Execute(ctx, detectScript)
//	firmwareVersion := result.Data["version"].(string)
//
//	// Get firmware from catalog
//	db, _ := gdb.LoadFirmwares()
//	firmware, ok := db.Get(firmwareVersion)
//	if !ok {
//	    return gdb.FirmwareUnsupportedError{Version: firmwareVersion}
//	}
//
//	// Inject certificate
//	injectScript := &InjectCertsScript{
//	    firmware:     firmware,
//	    certData:     rootCA,
//	    targetFile:   "/cert/129.der",
//	    openocdHost:  "localhost",
//	    openocdPort:  3333,
//	}
//	result, err := executor.Execute(ctx, injectScript)
//	if err != nil {
//	    return fmt.Errorf("certificate injection failed: %w", err)
//	}
//
//	fmt.Printf("Certificate injected successfully (%d bytes)\n",
//	    result.BytesWritten)
//
// # References
//
// GDB Documentation: https://sourceware.org/gdb/documentation/
// OpenOCD Manual: https://openocd.org/doc/html/index.html
// TI CC3200 SDK: https://www.ti.com/tool/CC3200SDK
// SimpleLink API: TI's embedded file system API (sl_FsOpen, sl_FsWrite, etc.)
package gdb
