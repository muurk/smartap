# Hardware Resources

Part of the [Smartap Project](https://github.com/muurk/smartap) | [Documentation](https://muurk.github.io/smartap/)

---

This directory contains configuration files and scripts for hardware-level access to the Smartap device via JTAG.

## Contents

| Directory | Description |
|-----------|-------------|
| [openocd/](openocd/) | OpenOCD configuration for connecting to the CC3200 |
| [gdb-scripts/](gdb-scripts/) | Reference GDB scripts for filesystem operations |

## Getting Started

For complete instructions on hardware access, including:

- Safely opening the Smartap device
- Locating and connecting to the JTAG header
- Wiring a Raspberry Pi for JTAG debugging
- Troubleshooting connection issues

See the **[Hardware Access Guide](https://muurk.github.io/smartap/technical/hardware-access/)** in the online documentation.

## Quick Reference

### JTAG Pin Mapping

| Smartap Pin | Function | Raspberry Pi GPIO |
|-------------|----------|-------------------|
| Pin 2 | TDO | GPIO 26 |
| Pin 3 | TCK | GPIO 13 |
| Pin 4 | TMS | GPIO 19 |
| Pin 5 | TDI | GPIO 6 |
| Pin 6 | GND | GND |

### Starting OpenOCD

```bash
openocd -f ./hardware/openocd/sysfsgpio-smartap.cfg \
        -c "transport select jtag" \
        -c "bindto 0.0.0.0" \
        -f ./hardware/openocd/cc3200-complete.cfg
```

## Related Documentation

- [Hardware Access Guide](https://muurk.github.io/smartap/technical/hardware-access/) - Complete JTAG setup instructions
- [Certificate Flashing](https://muurk.github.io/smartap/user-guide/certificate-flashing/) - Replacing device certificates
- [Research Background](https://muurk.github.io/smartap/technical/research-background/) - How the JTAG interface was discovered
