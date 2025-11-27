# Frequently Asked Questions

Common questions about the project.

## General Questions

### What is this project?

This project aims to restore functionality to Smartap smart shower devices after the manufacturer ceased trading and their cloud services went offline.

### Will this work with my Smartap device?

Most likely! The project targets the CC3200-based Smartap devices sold in the UK. If you have:

- Smartap E-Valve model
- A device that used to connect to smartap-tech.com
- A device with the original Smartap mobile apps

Then yes, this should work for your device.

### Is this legal?

Yes. You own the device, and you have the right to modify it. The manufacturer no longer exists and the original cloud service is defunct. This project doesn't violate any copyrights or patents - it simply enables devices to function independently of dead infrastructure.

### Do I need to be technical to use this?

**For basic configuration:** No. The configuration tool is designed for non-technical users with step-by-step guides.

**For smart features:** Some technical knowledge is helpful, as it requires JTAG hardware access and server setup. However, detailed guides are provided.

## Setup Questions

### How long does setup take?

- **Basic configuration only:** 15-30 minutes - See [Configuring Your Device](../getting-started/configuring.md)
- **Full smart feature restoration:** 3-6 hours (first time) - See [Complete Jailbreak Guide](../jailbreak/overview.md)

The first-time JTAG setup takes longest (hardware connection, OpenOCD configuration). Subsequent operations are much faster.

### What hardware do I need?

**For basic configuration:**

- Just a computer connected to your network
- No hardware modification required

**For smart features (jailbreak):**

- Raspberry Pi (any model with GPIO) or similar SBC
- 5 jumper wires (female-to-female recommended)
- Optional: Soldering iron for permanent JTAG header

See [Hardware Setup](../jailbreak/hardware-setup.md) for the complete list and wiring diagrams.

### Will this damage my device?

If you follow instructions carefully, the risk is minimal. The certificate injection:

- Uses the device's own SDK functions
- Requires 100% firmware signature match before writing
- Has been tested on multiple devices

**Risks to be aware of:**

- JTAG wiring errors could damage GPIO pins (rare)
- Opening the device voids any remaining warranty
- Incorrect reassembly could compromise water sealing

### Can I undo the changes?

**Certificate changes:** Reversible. You can inject the original Comodo CA certificate to restore factory trust (though the original servers are still dead).

**Configuration changes:** Fully reversible via `smartap-cfg`.

**Hardware changes:** If you soldered a JTAG header, it's permanent but doesn't affect functionality. Jumper wire connections leave no trace.

## Feature Questions

### What features currently work?

**Working now:**

| Feature | Tool | Notes |
|---------|------|-------|
| Outlet configuration | smartap-cfg | Which buttons control which outlets |
| WiFi setup | smartap-cfg | Connect device to your network |
| Server address change | smartap-cfg | Point device to custom server |
| Certificate injection | smartap-jtag | Replace CA for custom server trust |
| Device connections | smartap-server | TLS + WebSocket established |
| Message logging | smartap-server | Capture all device traffic |

**In development (blocked on protocol documentation):**

- Remote outlet control
- Temperature monitoring
- Scheduling / timers
- Usage analytics

See [Server Limitations](../server/limitations.md) for details on what's blocking these features and how you can help.

### Will the mobile apps work?

The original Smartap mobile apps will not work. They're hardcoded to communicate with the defunct cloud service at `smartap-tech.com`.

**Alternatives:**

- **smartap-cfg** - TUI for device configuration (works now)
- **Web interface** - Planned, pending protocol documentation
- **Home Assistant** - Planned integration
- **Custom apps** - Community contributions welcome

### Can I control it with Alexa/Google Assistant?

Not directly yet. Once Home Assistant integration is complete:

- Use Home Assistant's voice integrations
- Control via HA's Alexa/Google bridges
- Set up custom voice commands through HA automations

This depends on completing the protocol documentation so we can send commands the device understands.

### Does this work outside my home network?

Currently, the server operates on your local network only. The device connects to your server via your local network.

**For remote access:**

- **VPN** - Connect to your home network remotely
- **Home Assistant Cloud** - When integration is available
- **Reverse proxy** - Advanced users can expose the server (not recommended for security reasons)

The device itself doesn't support remote connections—it only connects to one server.

## Technical Questions

### Why does this require JTAG?

The device validates TLS certificates against a CA certificate stored in flash memory at `/cert/129.der`. There's no software API to replace this certificate—it's locked down.

**The technical solution:** We use JTAG to connect GDB to the running device, then call the device's own SimpleLink SDK functions (`sl_FsDel`, `sl_FsOpen`, `sl_FsWrite`, `sl_FsClose`) to delete the old certificate and write a new one. It's the device's own code doing the work—we just orchestrate it via GDB.

See [How Certificate Injection Works](../how-it-works/certificate-injection.md) for the full technical explanation, or [Development Setup](../technical/development.md#the-gdb-executor-internalgdb) for how the GDB scripting system works.

### Could this be done via software?

We haven't found a software-only method. The device:

- Has no certificate update API
- Validates certificates before allowing any TLS connection
- Doesn't expose the filesystem over HTTP
- Has no firmware update mechanism we can exploit

The HTTP configuration interface (`smartap-cfg`) can change WiFi settings and outlet assignments, but cannot touch the certificate store. That's why JTAG is necessary for the "jailbreak."

### What if I don't have JTAG access?

**You can still use `smartap-cfg`** for basic configuration without any hardware modification:

- Configure WiFi networks
- Set outlet assignments (which buttons control which outlets)
- Change the server address the device connects to

However, without certificate injection, the device will only trust the original (now-defunct) Smartap servers. It won't connect to your replacement server.

See [Configuring Your Device](../getting-started/configuring.md) for what's possible without JTAG, and [Hardware Setup](../jailbreak/hardware-setup.md) if you decide to proceed with the full jailbreak.

### How does firmware detection work?

The JTAG tool needs to know exact memory addresses for the SimpleLink SDK functions. These addresses differ between firmware versions.

**Signature-based detection:** The tool reads 8 bytes at 7 known function addresses and compares them against stored signatures. If all 7 match a known firmware, we have 100% confidence we're using the correct addresses.

```
sl_FsOpen:  0x20015c64 → [0x4606b570, 0x78004818]
sl_FsRead:  0x20014b54 → [0x43f0e92d, 0x48254680]
... (5 more signatures)
```

**Why 7 signatures?** Statistical certainty. One signature matching by coincidence is possible. Seven independent signatures across different memory regions matching is statistically impossible unless it's the correct firmware.

See [Firmware Analysis](../contributing/firmware-analysis.md) if your device has an unrecognized firmware version.

### Why can't I use modern TLS (TLS 1.3, ECDHE)?

The CC3200 chip was released in 2014 with Texas Instruments' SimpleLink SDK. Its TLS implementation only supports:

| Feature | CC3200 Support | Modern Standard |
|---------|----------------|-----------------|
| TLS Version | 1.2 only | 1.3 preferred |
| Key Exchange | RSA only | ECDHE preferred |
| Cipher Mode | CBC | GCM preferred |

This isn't a limitation of our server—it's a hardware constraint. The server is configured with CC3200-compatible cipher suites. See [Server Limitations](../server/limitations.md#the-cc3200-tls-stack) for technical details.

### Is the protocol fully documented?

Not yet. The device uses a custom binary protocol over WebSocket—not JSON or any standard format:

```
[0x7e] [0x03] [msg_id: 4 bytes] [length: 2 bytes] [payload] [padding]
```

We've identified message types through static analysis:

- `0x01`: Telemetry broadcast (97% of traffic, sent every ~1.8s)
- `0x29`: Telemetry response
- `0x42`: Command message
- `0x55`: Pressure mode status

But the payload structure within these messages is still being documented. See [Protocol Documentation](../technical/protocol.md) for current knowledge and how to contribute findings.

### What's the `--analysis-dir` flag for?

It enables protocol research mode. When you run:

```bash
./smartap-server server --analysis-dir ./captures
```

The server writes JSON Lines files with every message received:

- Timestamps
- Raw hex payloads
- Parsed frame structure
- Message type identification

This data is invaluable for understanding the protocol. If you capture interesting patterns (especially correlating physical actions with message changes), please share your findings!

### Can I contribute to development?

Absolutely! The most impactful contributions right now:

1. **Protocol documentation** (critical) - Capture messages, correlate with device actions, document findings
2. **Firmware support** (high impact) - Analyze memory dumps to add support for new firmware versions
3. **Server features** (medium) - REST API, Home Assistant integration, MQTT bridge
4. **Testing** (always valuable) - Unit tests, integration tests, real-device testing

See the [Contributing Guide](../contributing/overview.md) for detailed guidance on each area, including the [high-impact contribution areas](../contributing/overview.md#high-impact-contribution-areas).

## Troubleshooting Questions

### My device won't connect to WiFi

Check:

- **2.4GHz only** - The CC3200 doesn't support 5GHz networks
- **Password correct** - Case sensitive, no trailing spaces
- **No AP isolation** - Router must allow device-to-device communication
- **Setup mode** - Device should have LED flashing yellow

See [Troubleshooting Guide](../getting-started/troubleshooting.md) for detailed WiFi diagnostics.

### Configuration tool can't find my device

The tool uses mDNS discovery to find devices. Verify:

- Computer and device on same network subnet
- Firewall allows mDNS (UDP port 5353)
- Device powered on and connected to WiFi
- Router supports mDNS/Bonjour (most do)

**Manual connection:** If discovery fails, you can specify the IP directly:
```bash
./smartap-cfg wizard --device 192.168.1.100
```

### Certificate flashing failed

Common causes:

1. **JTAG wiring incorrect** - Double-check pin connections against [Hardware Setup](../jailbreak/hardware-setup.md)
2. **OpenOCD not connected** - Verify OpenOCD shows "halted" state
3. **Unknown firmware** - Run `detect-firmware` first; if unrecognized, see [Firmware Analysis](../contributing/firmware-analysis.md)
4. **Certificate format wrong** - Must be DER format, not PEM

The tool requires 100% firmware signature match before writing. This protects against corrupting devices with wrong addresses.

### Device connects but doesn't respond

If TLS handshake succeeds but the device doesn't send messages:

1. **Check server logs** - Run with `--log-level debug`
2. **Verify HTTP 101 response** - The device is picky about the exact WebSocket upgrade format (see [Server Limitations](../server/limitations.md#the-websocket-validation-bug))
3. **Check certificate chain** - Device must trust the server's certificate via your injected CA

If messages appear but nothing happens, that's expected—we're still documenting the protocol. Your captured messages help!

## Safety Questions

### Is it safe to modify a device in my bathroom?

Important safety considerations:

**Electrical Safety:**
- Always disconnect power before opening device
- Work with a qualified electrician if unsure
- Ensure proper sealing after modification

**Water Safety:**
- Reinstall all seals properly after opening
- Check for water ingress regularly
- Consider IP rating requirements

**Functional Safety:**
- Test thoroughly before relying on device
- Have backup manual controls
- Don't use for critical applications

### Can this cause water damage?

If seals are properly reinstalled after opening the device, there should be no additional risk. However:

- Any time you open a bathroom device, you risk compromising seals
- Take care during reassembly
- Monitor for any signs of water ingress
- Consider having a professional verify your work

### What about electromagnetic interference?

JTAG wires temporarily connected during flashing could theoretically cause interference, but:

- JTAG is only used briefly for certificate flashing
- Normal operation doesn't involve JTAG
- Device still complies with original EMC specifications

## Support Questions

### Where can I get help?

- [Troubleshooting Guide](../getting-started/troubleshooting.md)
- [Community Forum](community.md)
- [GitHub Issues](https://github.com/muurk/smartap/issues)
- Home Assistant Community Thread

### How do I report a bug?

Open an issue on GitHub with:
- Clear description of the problem
- Steps to reproduce
- Your environment (OS, versions, device model)
- Relevant log output

### Can you help me install this?

We provide documentation and community support, but cannot provide individual installation services. However:

- Detailed guides are available
- Community can answer questions
- Consider finding a local tech-savvy friend
- Some community members may offer help

### Is there commercial support available?

Currently, this is a community project without commercial support. However:

- Documentation is comprehensive
- Community is helpful and responsive
- Most issues are solvable with the guides

## Future Questions

### What features are planned?

See [Server Limitations](../server/limitations.md) for current state. Planned highlights:

- Remote control
- Scheduling
- Temperature monitoring
- Home Assistant integration
- MQTT support
- Web interface

### When will X feature be ready?

We don't provide specific timelines as this is a volunteer project. Features will be released when:

- Thoroughly tested
- Properly documented
- Stable and reliable

### Can I request a feature?

Yes! Open a GitHub issue with:
- Description of the feature
- Your use case
- Why it would be valuable

### Will you support other devices?

The techniques used here could apply to other abandoned IoT devices. If you have:

- Another defunct IoT device based on the TI CC3200
- Technical documentation or access
- Willingness to research

## Business Questions

### Why did Smartap fail?

The exact reasons aren't publicly known, but common factors in IoT company failures include:

- Expensive hardware manufacturing
- Ongoing cloud infrastructure costs
- Small market size
- Competition from established brands
- Difficulty achieving profitability

### Could this happen to other devices?

Unfortunately, yes. Many IoT devices are cloud-dependent. When companies fail or discontinue products, the devices can become unusable.

**Protect yourself:**
- Research before buying IoT devices
- Prefer devices with local control
- Look for open APIs and documentation
- Consider company stability and commitment

### Does this prove cloud-dependent IoT is bad?

Not necessarily bad, but risky. This project demonstrates:

- Cloud dependency creates obsolescence risk
- Devices should have local fallback options
- Open standards benefit everyone
- Right to repair is important

## Still Have Questions?

- Check the relevant documentation sections
- Search [GitHub Discussions](https://github.com/muurk/smartap/discussions) (your question may already be answered)
- Ask in the [Q&A category](https://github.com/muurk/smartap/discussions/categories/q-a) - best for setup help and troubleshooting
- Open a [GitHub Issue](https://github.com/muurk/smartap/issues) only for specific bugs or feature requests

---

[:material-arrow-left: Previous: Project History](project-history.md){ .md-button }
[:material-arrow-right: Next: Community](community.md){ .md-button .md-button--primary }
