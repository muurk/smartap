# OpenOCD Configuration Files

Part of the [Smartap Project](https://github.com/muurk/smartap) | [Documentation](https://muurk.github.io/smartap/)

---

OpenOCD configuration files for connecting to the Smartap device via JTAG.

## Files

| File | Description |
|------|-------------|
| `sysfsgpio-smartap.cfg` | Raspberry Pi GPIO pin mapping for bit-banged JTAG |
| `cc3200-complete.cfg` | CC3200 target configuration with watchdog handling |

## Quick Start

On a Raspberry Pi with OpenOCD installed:

```bash
# From the repository root
openocd -f ./hardware/openocd/sysfsgpio-smartap.cfg \
        -c "transport select jtag" \
        -c "bindto 0.0.0.0" \
        -f ./hardware/openocd/cc3200-complete.cfg
```

Successful connection looks like:

```
Info : Listening on port 4444 for telnet connections
Info : Listening on port 3333 for gdb connections
Info : JTAG tap: cc32xx.jrc tap/device found
Info : cc32xx.cpu: hardware has 6 breakpoints, 4 watchpoints
```

## Hardware Setup

For detailed instructions on:

- Opening the Smartap device safely
- Locating the JTAG header
- Wiring the Raspberry Pi to Smartap
- Troubleshooting connection issues

See the **[Hardware Access Guide](https://muurk.github.io/smartap/technical/hardware-access/)** in the online documentation.

## GPIO Pin Mapping

| Smartap Pin | Function | Raspberry Pi GPIO | RPi Physical Pin |
|-------------|----------|-------------------|------------------|
| Pin 2 | TDO | GPIO 26 | Pin 37 |
| Pin 3 | TCK | GPIO 13 | Pin 33 |
| Pin 4 | TMS | GPIO 19 | Pin 35 |
| Pin 5 | TDI | GPIO 6 | Pin 31 |
| Pin 6 | GND | GND | Pin 39 |

## Requirements

- Raspberry Pi (any model with GPIO header)
- OpenOCD installed (`sudo apt-get install openocd`)
- Physical connection between Pi and Smartap JTAG header

## Troubleshooting

**"JTAG scan chain interrogation failed"**
- Check all wire connections
- Verify Smartap device has power
- Try reducing speed: edit `adapter speed 50` in sysfsgpio-smartap.cfg

**"Permission denied" on GPIO**
- Add your user to the gpio group: `sudo usermod -aG gpio $USER`
- Log out and back in

**Device keeps resetting**
- The cc3200-complete.cfg includes watchdog handling, but ensure you're using it
- Check that both config files are loaded in the correct order
