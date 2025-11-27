# Contributing Without Hardware Modification

**What if I'm not confident undertaking the JTAG modification, but could help with the server software?**

This page is for you.

## The Reality

The JTAG modification process involves:

- Extracting the CC3200 module from its sealed case
- Soldering a header onto the board
- Running OpenOCD and GDB scripts
- Reconnecting everything and testing

For many people, especially those without electronics experience, this is understandably daunting. Yet you might be exactly the kind of person who could make a real difference to this project - if only you had access to a modified device.

## Who Should Read This

You should consider this path if:

- You have software development experience (Go, embedded systems, networking)
- You're comfortable analyzing binary protocols
- You understand TLS/WebSocket communications
- You're willing to invest time in reverse engineering
- You genuinely believe you could contribute to the server implementation

## Before You Proceed

### Understand What's Needed

The server software (`smartap-server`) is experimental. The device communicates using an **unknown binary protocol** over WebSocket. What we need most is people who can:

1. Analyze captured message data
2. Identify patterns in the binary frames
3. Implement protocol handlers
4. Test with real devices

### Read the Source Code

Before requesting a pre-modified device, familiarize yourself with:

- **Server source**: `cmd/smartap-server/` and `internal/server/`
- **Protocol documentation**: [Protocol Documentation](../technical/protocol.md)
- **Architecture overview**: [System Architecture](../technical/architecture.md)

If you read through these and feel "I could work on this" - you're the right person.

### Understand What a Modified Device Gets You

A "jailbroken" Smartap device has:

- Its CA certificate replaced with the project's embedded CA
- The ability to connect to your own server (instead of the defunct cloud)
- No other modifications - it's still the same device, just trusting a different certificate authority

With a modified device and the server running, you can:

- Capture all device-server communications
- Analyze message structures
- Test protocol implementations
- Iterate on server code

Without a modified device, you'd be working blind - you can read the code but can't test anything.

## Expressing Interest

Currently, there's no organized system for distributing pre-modified devices. This is a community project, and we're all volunteers.

However, if there's genuine interest from capable contributors, we may be able to connect you with someone in the community who has a spare device or is willing to help.

**To express interest:**

1. [Open a GitHub issue using the "Pre-Modified Device Request" template](https://github.com/muurk/smartap/issues/new?template=device-request.md)
2. Explain your background and what you could contribute
3. **Do NOT share personal contact information in the issue** - keep it public and anonymous
4. Wait for community response

!!! warning "No Guarantees"
    There's no guarantee anyone will be able to help. This depends entirely on community availability and interest. Don't spend money or make plans based on the assumption you'll receive a device.

## What Happens Next

If someone in the community is able to help:

- They'll respond to your issue
- You'll work out logistics privately (shipping, etc.)
- You'll receive a device with the certificate already replaced
- You can start contributing immediately

If no one responds:

- Don't be discouraged - the community is small
- Consider whether the hardware modification is really beyond your abilities
- Perhaps find a local makerspace or electronics hobbyist who could help with just the soldering part

## Alternative: Find Local Help

If you're comfortable with most of the process but just need help with soldering:

- Local makerspaces often have members happy to help
- Electronics repair shops might do a simple solder job cheaply
- University electronics clubs may have volunteers
- A friend or colleague with soldering experience

The actual soldering is quite straightforward - just attaching a 6-pin header to clearly marked pads. The rest of the process (software, JTAG commands) you can do yourself.

## Questions?

Open a general GitHub issue if you have questions about contributing or the skill level required. 

---

[:material-arrow-left: Back to Contributing Overview](overview.md){ .md-button }
