# GDB Scripts (Reference Only)

Part of the [Smartap Project](https://github.com/muurk/smartap) | [Documentation](https://muurk.github.io/smartap/)

---

These GDB scripts are provided for **reference and educational purposes**. They document the low-level approach used to interact with the CC3200's filesystem via JTAG/GDB.

> **Note:** These scripts have been matured and integrated into the `smartap-jtag` utility, which provides a more reliable and user-friendly interface. For actual certificate flashing operations, use `smartap-jtag` instead of running these scripts directly.

## Files

| Script | Description |
|--------|-------------|
| `read-file.gdb` | Reads a certificate file from the CC3200 filesystem |
| `replace-cert.gdb` | Replaces the root CA certificate on the device |

## How They Work

These scripts exploit the fact that the CC3200's SimpleLink library functions remain in memory at known addresses. By manipulating CPU registers and the program counter via GDB, we can call these functions directly:

| Function | Address | Purpose |
|----------|---------|---------|
| `sl_FsOpen` | `0x20015c64` | Open a file on the device |
| `sl_FsRead` | `0x20014b54` | Read file contents |
| `sl_FsWrite` | `0x20014bf8` | Write file contents |
| `sl_FsClose` | `0x2001555c` | Close file handle |
| `sl_FsDel` | `0x20016ea8` | Delete a file |

The scripts:

1. Halt the device
2. Write parameters to a work buffer in RAM
3. Set up registers (r0-r3) with function arguments
4. Set the program counter to the function address
5. Execute and capture the result
6. Resume the device

## Why Use smartap-jtag Instead

The `smartap-jtag` utility improves on these scripts by:

- **Automatic connection handling** - Manages OpenOCD and GDB sessions
- **Error recovery** - Handles timeouts and connection issues gracefully
- **Progress feedback** - Shows clear status during operations
- **Validation** - Verifies certificates before and after flashing
- **Cross-platform** - Works on Linux, macOS, and Windows

## Usage (smartap-jtag)

```bash
# Flash certificates using the integrated utility
./bin/smartap-jtag inject-certs

# Or with custom certificate path
./bin/smartap-jtag inject-certs --cert /path/to/ca-root-cert.der
```

## Running Scripts Directly (Advanced)

If you need to run these scripts directly (e.g., for debugging or development):

```bash
# Ensure OpenOCD is running first (see ../openocd/README.md)

# Then run a script with GDB
arm-none-eabi-gdb -x read-file.gdb
```

**Prerequisites:**

- OpenOCD running with CC3200 connected
- `arm-none-eabi-gdb` or `gdb-multiarch` installed
- Edit the `target remote` line to match your OpenOCD host/port

## Further Reading

- [Hardware Access Guide](https://muurk.github.io/smartap/technical/hardware-access/) - JTAG connection setup
- [Certificate Flashing Guide](https://muurk.github.io/smartap/user-guide/certificate-flashing/) - End-to-end certificate replacement
