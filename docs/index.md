# Smartap

**Open-source tools to restore your Smartap smart shower after the company shut down.**

!!! note "This is an independent open source community project"
    This project has no connection with the original Smartap company or product. We are researchers, hobbyists, and software developers working to revive these devices and give back to the community. All tools and documentation are provided as-is, with no warranty.

---

## The Problem

Smartap was a premium smart shower system (~£695) that offered app control, voice assistant integration, scheduling, and remote pre-heating. When the company ceased trading in 2023, their cloud servers went offline - and thousands of devices lost all their "smart" features overnight.

If you own one of these devices, you're probably here because:

- Your shower stopped responding to the app
- You can't reconfigure WiFi or outlet settings
- You paid a lot of money for a "smart" device that's now dumb

**This project exists to fix that.**

## What We Provide

| Tool | Purpose | Difficulty |
|------|---------|------------|
| **smartap-cfg** | Configure WiFi and outlet settings | Easy - no hardware modification |
| **smartap-jtag** | Replace the device's CA certificate | Advanced - requires soldering and Raspberry Pi |
| **smartap-server** | Replacement server for smart features | Experimental - protocol research ongoing |

---

## Choose Your Path

### I just want my shower controls working

**You need: `smartap-cfg`**

![smartap-cfg wizard main menu](assets/images/screenshots/smartap-cfg.png){ loading=lazy }

Configure WiFi and outlet settings without needing the original cloud service. No hardware modifications required.

!!! warning "Won't Fix Hardware Issues"
    This tool configures software settings only. It cannot fix low water pressure, leaking valves, or other physical problems.

[:material-arrow-right: Get Started](getting-started/overview.md){ .md-button .md-button--primary }

---

### I want to restore smart features (remote control, etc.)

**You need: `smartap-jtag` + Raspberry Pi + `smartap-server`**

"Jailbreak" your device by injecting a CA certificate, then run your own server.

!!! info "What This Involves"
    - Raspberry Pi with JTAG connection to device
    - Physical access to device internals (soldering required)
    - Running a home server
    - Experimental software (protocol not fully documented)

[:material-arrow-right: Device Jailbreak Guide](jailbreak/overview.md){ .md-button .md-button--primary }

---

### I want to contribute or understand the technical details

Help reverse-engineer the protocol, add firmware support, or improve the tools.

[:material-arrow-right: Technical Documentation](technical/architecture.md){ .md-button }
[:material-arrow-right: Contributing Guide](contributing/overview.md){ .md-button }

---

## Quick Links

- [Downloads](downloads.md) - Get the tools
- [FAQ](about/faq.md) - Common questions
- [GitHub Repository](https://github.com/muurk/smartap) - Source code and issues

## Need Help?

- [FAQ](about/faq.md) - Common questions
- [Troubleshooting](getting-started/troubleshooting.md) - Solve common problems
- [Community](about/community.md) - Get help from others
- [GitHub Issues](https://github.com/muurk/smartap/issues) - Report bugs
