# Smartap Revival Project

**Bringing abandoned smart showers back to life.**

Smartap was a smart shower system (~£695) which used the Texas Instruments CC3200 wifi module to  offer app control, scheduling, and voice assistant integration. When the company ceased trading in 2023, their cloud servers went offline, leaving thousands of devices without smart functionality.

This community project provides tools to restore and extend these orphaned devices.

## What's Included

| Tool | Purpose | Difficulty |
|------|---------|------------|
| **smartap-cfg** | Configure WiFi and outlet settings | Easy - no hardware modification |
| **smartap-jtag** | Inject custom certificates via JTAG | Advanced - requires soldering |
| **smartap-server** | Replacement WebSocket server | Experimental - for research |

## Quick Start

### Basic Configuration (No Hardware Modification)

Download `smartap-cfg` for your platform and run it. The wizard will guide you through configuring your device's WiFi and outlet settings.

**[Download smartap-cfg](https://github.com/muurk/smartap/releases/latest)**

### Full Smart Features (Requires Jailbreak)

To restore remote control, scheduling, and other smart features, you'll need to:

1. Solder a JTAG header onto your device's CC3200 module
2. Use a Raspberry Pi to inject a custom CA certificate
3. Run the replacement server

This is documented in detail in our guides.

## Documentation

**[Read the full documentation](https://muurk.github.io/smartap/)**

- [Getting Started](https://muurk.github.io/smartap/getting-started/overview/) - Configure your device
- [Jailbreak Guide](https://muurk.github.io/smartap/jailbreak/overview/) - Hardware modification for full control
- [Server Setup](https://muurk.github.io/smartap/server/overview/) - Run your own server
- [Technical Details](https://muurk.github.io/smartap/technical/architecture/) - Architecture and protocol research
- [FAQ](https://muurk.github.io/smartap/about/faq/) - Common questions answered

## Project Status

| Component | Status |
|-----------|--------|
| Device configuration (smartap-cfg) | Stable |
| Certificate injection (smartap-jtag) | Stable |
| Server (smartap-server) | Experimental |
| Protocol documentation | In progress |

The device communication protocol is not fully documented. The server can accept connections and log messages, but full remote control is not yet implemented. Contributions welcome.

## Hardware Requirements

**For basic configuration:** Just a computer (Windows, macOS, or Linux)

**For jailbreak:**
- Raspberry Pi (3, 4, 5, or Zero W)
- Soldering iron
- 5 female-to-female jumper wires
- Header pins

## Contributing

We welcome contributions of all kinds:

- **Testing** - Try the tools and report issues
- **Documentation** - Improve guides or fix errors
- **Protocol research** - Help decode the device communication protocol
- **Code** - Implement features or fix bugs

See [Contributing Guide](https://muurk.github.io/smartap/contributing/overview/) for details.

## Building from Source

```bash
# Clone the repository
git clone https://github.com/muurk/smartap.git
cd smartap

# Build all tools
make build-all

# Run tests
make test

# Build documentation
make docs
```

Requires Go 1.21+.

## License

This project is licensed under the **GNU Affero General Public License v3.0** (AGPL-3.0).

You are free to use, modify, and distribute this software. If you run a modified version as a network service, you must make your source code available.

See [LICENSE](LICENSE) for full terms.

## Disclaimer

This is an independent community project with **no connection** to the original Smartap company.

- The original Smartap company has ceased trading
- We cannot provide hardware repairs or replacement parts
- Use these tools at your own risk
- No warranty is provided

## Acknowledgements

This project exists thanks to the right-to-repair community and everyone who contributed research, code, documentation, and testing.

---

**[Documentation](https://muurk.github.io/smartap/)** · **[Releases](https://github.com/muurk/smartap/releases)** · **[Issues](https://github.com/muurk/smartap/issues)** · **[Discussions](https://github.com/muurk/smartap/discussions)**
