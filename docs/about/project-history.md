# Project History

The story behind this project.

## The Rise of Smartap

### The Product

The Smartap smart shower system was a premium IoT device sold primarily in the UK market, retailing for approximately £695 through retailers like VictoriaPlumb. It represented an ambitious attempt to bring smart home technology to the bathroom.

**Key Features:**

- Remote control via iOS and Android apps
- Integration with Google Assistant and Amazon Alexa
- IFTTT automation support
- Multiple valve control for different fixtures
- Temperature and pressure monitoring
- Usage analytics and scheduling
- Remote shower pre-heating

**The Promise:**

Start your shower warming up from bed, never step into a cold shower again, track water usage, and integrate your shower into your smart home ecosystem.

### The Technology

Built on the Texas Instruments CC3200 WiFi microcontroller, the device featured:

- ARM Cortex-M4 processor
- Integrated 2.4GHz WiFi
- On-chip TLS/SSL support
- Flash storage for certificates and configuration

The architecture was heavily cloud-dependent, with the device communicating with Smartap's servers at `evalve.smartap-tech.com` for all smart functionality.

## The Fall

### Company Closure

In approximately 2023, Smartap ceased trading. The reasons aren't publicly documented, but the outcome was clear: the cloud infrastructure went offline.

### The Impact

When the cloud services disappeared:

- Mobile apps stopped working
- Voice control integration broke
- IFTTT automation failed
- Scheduling and remote control vanished
- Usage analytics became inaccessible

**What Still Worked:**

Basic shower operation continued - the physical buttons on the control panel still activated valves. But all "smart" features were lost.

### The Problem

For owners who had these devices installed:

- Significant investment made (£695+ plus installation)
- Bathrooms potentially designed around the device
- Tiling and plumbing committed to the unit
- No path to restore functionality
- No firmware updates or recovery options
- Manufacturer no longer exists for support

## The Research Begins

### Initial Motivation

One of the project founders had a Smartap device installed during a complete bathroom renovation. The entire bathroom was designed, tiled, and plumbed around this unit. When the cloud went offline, it represented both a significant financial loss and a technical challenge.

Rather than accept an expensive device reduced to basic functionality, the decision was made to investigate: **Could it be revived?**

### Early Investigations (2024)

Initial research focused on understanding the device:

1. **Network Analysis**
   - Device had a local web server for WiFi configuration
   - Attempted to connect to smartap-tech.com
   - Used TLS with certificate validation
   - All smart features required cloud connectivity

2. **Software Vulnerabilities**
   - Identified the device was vulnerable to CVE-2021-21966 (a known memory disclosure vulnerability)
   - Attempted to extract useful information
   - Limited success with software-only approach

3. **Hardware Investigation**
   - Community members shared JTAG header photos
   - Identified unpopulated JTAG pins on CC3200 module
   - Recognized potential for hardware debugging

### The Breakthrough (November 2025)

After nearly a year of on-and-off research, a major breakthrough occurred:

**Certificate Replacement Success**

By combining:
- JTAG access via Raspberry Pi
- Memory analysis with Ghidra
- TI CC3200 SDK source code study
- GDB scripting for filesystem manipulation

It became possible to replace the device's CA certificate, allowing it to trust a custom certificate authority. This meant the device could establish TLS connections to a custom server, effectively breaking free from the defunct cloud infrastructure.

## The Revival Project

### Project Launch

With the certificate breakthrough achieved, this project was formally established to:

1. **Document the research** - Share findings with the community
2. **Develop server software** - Create replacement infrastructure
3. **Build configuration tools** - Make it accessible to non-technical users
4. **Support the community** - Help others restore their devices

### Philosophy

The project embodies several principles:

**Right to Repair**
- Devices should outlive their manufacturers
- Users should control devices they own
- Cloud dependency shouldn't mean obsolescence

**Open Source**
- All research shared publicly
- Software released under open licenses
- Community contributions welcome

**Accessibility**
- Tools for all skill levels
- Comprehensive documentation
- Support for newcomers

**Privacy and Security**
- Local-only operation (no cloud)
- Strong encryption maintained
- User controls their data

## Timeline

| Date | Milestone |
|------|-----------|
| ~2023 | Smartap ceases trading, cloud goes offline |
| Early 2024 | Research begins, initial investigations |
| Mid 2024 | Hardware access achieved, JTAG successful |
| Mid 2024 | Memory dumps obtained, analysis begins |
| Nov 2025 | Certificate replacement breakthrough |
| Nov 2025 | Project documentation published |

## Community Growth

### Home Assistant Community

The [Home Assistant community thread](https://community.home-assistant.io/t/smartap-shower-control-getting-started-with-reverse-engineering-a-smart-home-device/358251) has been instrumental in:

- Sharing hardware photos and pin mappings
- Discussing investigation approaches
- Documenting device variants
- Supporting each other through challenges

### GitHub Community

As the project grows, the GitHub repository serves as:

- Central code repository
- Issue tracking and feature requests
- Protocol documentation
- Community contributions

## The Future

### Short Term Goals

- Complete protocol documentation
- Implement core remote control features
- Expand device testing (multiple models/variants)
- Improve installation process

### Long Term Vision

- **Full Feature Restoration** - All original features working
- **Enhanced Features** - New capabilities beyond original
- **Home Automation** - Deep integration with HA, Node-RED, etc.
- **Easy Installation** - Streamlined setup process
- **Active Community** - Thriving contributor base

### Beyond Smartap

This project demonstrates that:

- IoT devices can be independent of manufacturers
- Community can restore abandoned products
- Open source benefits everyone
- Technical barriers can be overcome

The lessons learned here apply to countless other abandoned IoT devices. This project serves as a model for device revival and a statement about ownership, control, and sustainability in the IoT era.

## Acknowledgments

This project wouldn't exist without:

- **Community members** who shared JTAG pins and findings
- **bigclivedotcom** for hardware teardown videos
- **TI** for publishing CC3200 documentation
- **Open source tool developers** (OpenOCD, Ghidra, etc.)
- **Everyone who refused** to accept planned obsolescence

## Get Involved

The story continues with your help:

- Own a Smartap? Help test and document!
- Skilled developer? Contribute code!
- Good writer? Improve documentation!
- Experienced with IoT? Share knowledge!

See the [Contributing Guide](../contributing/overview.md) to get involved.

---

[:material-arrow-right: Next: FAQ](faq.md){ .md-button .md-button--primary }
