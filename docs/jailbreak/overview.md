# Device Jailbreak Overview

This guide explains how to "jailbreak" your Smartap device by replacing its CA certificate, enabling it to connect to your own server.

!!! warning "Technical Complexity"
    This process requires hardware access (JTAG), a Raspberry Pi, and comfort with command-line tools. It's not for casual users.

## Why Jailbreak?

The Smartap device validates TLS certificates against a CA certificate stored in its flash memory. Since the original company's servers are offline, the device can't connect to anything.

By replacing the CA certificate with one you control, you can:

- **Connect to smartap-server** - The included server with the embedded Root CA
- **Contribute to protocol research** - Capture and analyze device messages
- **Run your own infrastructure** - Complete control over your device

!!! info "Main Current Benefit"
    The primary value of jailbreaking right now is **contributing to protocol research**. The smartap-server is still experimental and doesn't yet provide full device control. Your captured traffic helps improve the project.

## What This Process Does

1. **Detects your firmware version** - Different versions have different memory layouts
2. **Injects a new CA certificate** - Replaces `/cert/129.der` on the device
3. **Enables custom server connections** - Device will trust certificates signed by the new CA

## What This Process Does NOT Do

- **Modify device firmware** - The device software remains unchanged
- **Unlock hidden features** - No new hardware capabilities are enabled
- **Provide remote control** - That requires a working server (still in development)

## Requirements Summary

### Hardware

- Raspberry Pi (3, 4, or 5)
- 5 jumper wires
- Physical access to device's JTAG header

### Software

- OpenOCD (for JTAG communication)
- arm-none-eabi-gdb (ARM debugger)
- smartap-jtag (included in this project)

### Time

- **First time setup:** 2-4 hours
- **Certificate injection:** 15-30 minutes (once set up)

## The Process

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. Install Prerequisites                                         │
│    OpenOCD, GDB toolchain, smartap-jtag binary                  │
├─────────────────────────────────────────────────────────────────┤
│ 2. Hardware Setup                                                │
│    Connect Raspberry Pi to Smartap JTAG header                   │
├─────────────────────────────────────────────────────────────────┤
│ 3. Start OpenOCD                                                 │
│    Establish JTAG connection to device                          │
├─────────────────────────────────────────────────────────────────┤
│ 4. Verify Setup                                                  │
│    smartap-jtag verify-setup                                     │
├─────────────────────────────────────────────────────────────────┤
│ 5. Detect Firmware                                               │
│    smartap-jtag detect-firmware                                  │
├─────────────────────────────────────────────────────────────────┤
│ 6. Inject Certificate                                            │
│    smartap-jtag inject-certs                                     │
├─────────────────────────────────────────────────────────────────┤
│ 7. Configure Device Server                                       │
│    Point device to your server IP                               │
└─────────────────────────────────────────────────────────────────┘
```

## Ready to Start?

[:material-arrow-right: Continue to Prerequisites](prerequisites.md){ .md-button .md-button--primary }

---

## Not Sure This Is For You?

- **Just need basic outlet control?** → [Getting Started Guide](../getting-started/overview.md) (no jailbreak needed)
- **Want to understand how this works?** → [How It Works](../how-it-works/overview.md)
- **Want to understand the research?** → [Research Background](../technical/research-background.md)
- **Have questions?** → [FAQ](../about/faq.md)

!!! question "Want to help but not comfortable with hardware modification?"
    If you have software development skills and want to contribute to the server/protocol work, but the soldering and JTAG process feels daunting, there may be another path.

    **[Read: Contributing Without Hardware Modification](../contributing/help-without-hardware.md)**
