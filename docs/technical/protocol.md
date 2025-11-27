# Protocol Documentation

Device communication protocol details.

!!! danger "Experimental and Unverified"
    **This is the most immature part of the project.**

    The protocol documentation reflects our current understanding based on:

    - Static analysis of firmware memory dumps using Ghidra
    - Limited packet captures from a small number of devices
    - Educated guesses about field meanings

    **No real-world testing or verification has been performed.** The server can receive and log messages, but we haven't confirmed that our interpretations are correct. Field meanings are speculative. Message construction code exists but hasn't been tested against live devices.

    **This documentation reflects the current state of the project, not proven facts.**

## Overview

The Smartap device communicates with the server using a custom binary protocol over WebSocket. This is **not JSON** - all messages are binary frames with specific byte-level structure.

Communication happens in two layers:

1. **WebSocket framing** - Standard RFC 6455 WebSocket protocol
2. **Smartap protocol** - Custom binary format inside WebSocket payloads

## Connection Establishment

### Phase 1: DNS Resolution

The device looks up `evalve.smartap-tech.com`. You can reconfigure the address which your smartapp device connects to using the smartap-cfg utility.

### Phase 2: TLS Handshake

```
1. TCP connection to port 443
2. TLS ClientHello
3. Server responds with certificate
4. Device validates against CA in /cert/129.der
5. TLS session established
```

### Phase 3: WebSocket Upgrade

```http
GET / HTTP/1.1
Host: evalve.smartap-tech.com
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: <base64-key>
Sec-WebSocket-Version: 13
```

The server responds with HTTP 101 Switching Protocols.

### Phase 4: Binary Communication

Once the WebSocket is established, all communication uses binary frames (opcode 0x2). The device sends periodic telemetry broadcasts and responds to commands.

## WebSocket Frame Layer

Standard WebSocket framing per RFC 6455:

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-------+-+-------------+-------------------------------+
|F|R|R|R| opcode|M| Payload len |    Extended payload length    |
|I|S|S|S|  (4)  |A|     (7)     |             (16/64)           |
|N|V|V|V|       |S|             |   (if payload len==126/127)   |
| |1|2|3|       |K|             |                               |
+-+-+-+-+-------+-+-------------+ - - - - - - - - - - - - - - - +
|     Extended payload length continued, if payload len == 127  |
+ - - - - - - - - - - - - - - - +-------------------------------+
|                               |Masking-key, if MASK set to 1  |
+-------------------------------+-------------------------------+
| Masking-key (continued)       |          Payload Data         |
+-------------------------------- - - - - - - - - - - - - - - - +
:                     Payload Data continued ...                :
+ - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - +
|                     Payload Data (continued)                  |
+---------------------------------------------------------------+
```

Device-to-server frames are masked (per WebSocket spec for client frames). Server-to-device frames are unmasked.

## Smartap Protocol Frame

Inside each WebSocket binary payload is a Smartap protocol frame:

```
Offset  Size  Field         Description
------  ----  -----         -----------
0       1     sync          Always 0x7e
1       1     version       Always 0x03
2       4     message_id    Little-endian uint32
6       2     length        Payload length (little-endian uint16)
8       N     payload       Message payload
N+8     M     padding       Zero padding to minimum 38 bytes
```

### Constants

| Constant | Value | Notes |
|----------|-------|-------|
| Sync byte | `0x7e` | Frame start marker |
| Version | `0x03` | Protocol version |
| Minimum frame size | 38 bytes | Includes padding |

### Message ID

The message ID is a 4-byte little-endian integer. For periodic broadcasts, the device uses the special ID `0x0FFFFFFF`. For request/response pairs, the server should generate unique IDs.

## Message Types

Based on Ghidra analysis, we've identified these message types:

| Type | Hex | Name | Direction | Description |
|------|-----|------|-----------|-------------|
| 0x01 | `01` | TelemetryBroadcast | Device → Server | Periodic status (~1.8s interval) |
| 0x05 | `05` | OTA | Unknown | Over-the-air update (unused) |
| 0x29 | `29` | TelemetryResponse | Device → Server | Response to telemetry query |
| 0x42 | `42` | Command | Bidirectional | Generic command/response |
| 0x44 | `44` | Extended | Unknown | Extended command format |
| 0x55 | `55` | PressureMode | Device → Server | Low pressure mode status |

### Telemetry Broadcast (0x01)

The most common message. Sent every ~1.8 seconds with message ID `0x0FFFFFFF`.

```
Offset  Size  Field           Observed Value    Notes
------  ----  -----           --------------    -----
0       1     type            0x01              Message type
1       1     telemetry_type  0x11              Consistent marker
2       1     status_type     0x0f              Format indicator
3       4     field1          0x08000000        Meaning unknown (LE)
7       4     field2          0x55800000        Contains 0x55? (LE)
11      1     subtype         0x03              Consistent in captures
12      19    data_fields     varies            Sensor/state data
31      1     trailing        0x29              Telemetry marker
```

This message appears in 97% of captured traffic. The `data_fields` section likely contains sensor readings, but the exact mapping is unknown.

### Telemetry Response (0x29)

Response to explicit telemetry queries:

```
Offset  Size  Field      Notes
------  ----  -----      -----
0       1     type       0x29
1       1     subtype    0x11 typically
2       1     field      0x80 typically
3       4     value      Sensor value (little-endian)
7       12    padding    Zero padding to 19 bytes
```

### Command Message (0x42)

Generic command format for device control:

```
Offset  Size  Field       Notes
------  ----  -----       -----
0       1     type        0x42
1       1     length      Length of data + 5
2       1     marker      0x01
3       4     category    Command category (little-endian)
7       N     data        Variable length data
```

The `category` field likely determines the specific command. Values are device-specific.

### Pressure Mode (0x55)

Status message for low pressure mode:

```
Offset  Size  Field      Notes
------  ----  -----      -----
0       1     type       0x55
1       1     subtype    0x04 typically
2       1     enabled    0 = disabled, 1 = enabled
```

## Dual-Valve Message

At connection start, the device sends a 77-byte message containing status for both valves:

```
Bytes 0-36:   Cold valve status (37 bytes, missing sync byte)
Bytes 37-76:  Hot valve status (40 bytes, complete frame)
```

The cold valve frame is missing its leading `0x7e` sync byte (TCP concatenation artifact).

### Valve Identifiers

| ID | Hex | Valve | Notes |
|----|-----|-------|-------|
| 202 | `0xca` | Cold | No temperature sensor |
| 109 | `0x6d` | Hot | Has temperature sensor |

The hot valve frame ends with `0x29`, indicating temperature sensor presence.

## What We Don't Know

This section is honest about the gaps:

### Unknown Field Meanings

- `field1` and `field2` in telemetry broadcasts
- The 19-byte `data_fields` section - likely contains sensor readings
- Command category codes and their effects
- Error codes and failure responses

### Untested Functionality

- Server-to-device commands (code exists, never tested)
- Telemetry queries
- OTA update mechanism
- Extended command format (0x44)

### Uncertain Interpretations

- Whether `subtype` fields are consistent across devices
- The exact timing and triggering of broadcasts
- Authentication mechanism (if any)
- Whether devices validate message IDs

## Protocol Research

The smartap-server is the right place to investigate the protocol. It terminates TLS and has direct access to decrypted WebSocket payloads - no need for external packet capture tools.

### Enabling Message Logging

```bash
smartap-server server --analysis-dir ./captures
```

This writes all received messages to JSON Lines files in the specified directory. Each message includes:

- Timestamp
- Raw payload (hex encoded)
- Parsed frame structure
- Message type identification

### Adding Protocol Analysis

The server codebase is designed for extension. Key files for protocol research:

| File | Purpose |
|------|---------|
| `internal/protocol/parser.go` | Message parsing and type detection |
| `internal/protocol/handler.go` | Message handling and logging |
| `internal/protocol/constructor.go` | Building messages to send |
| `internal/server/websocket.go` | WebSocket frame handling |

To investigate a specific aspect of the protocol:

1. Add logging or analysis code to the handler
2. Rebuild: `go build ./cmd/smartap-server`
3. Run with a connected device
4. Observe the output and captured data

### Correlating Device Actions

The most valuable research correlates messages with physical device actions:

1. Start the server with analysis logging enabled
2. Connect a jailbroken device
3. Perform a specific action (press a button, trigger a valve)
4. Examine the captured messages immediately before/after
5. Document which bytes changed

This approach has already identified the pressure mode message (0x55) and valve status structures.

## Contributing Protocol Research

If you discover new information:

1. **Document the context** - What action triggered the message?
2. **Include raw hex** - Exact bytes, not interpreted values
3. **Note patterns** - What's consistent vs. variable?
4. **Submit findings** - Open an issue with label `protocol-research`

Even partial findings help. The protocol will only be fully documented through community effort.

## Implementation Status

| Component | Status | Notes |
|-----------|--------|-------|
| WebSocket framing | Working | Can receive and parse frames |
| Protocol frame parsing | Working | Header and payload extraction |
| Message type detection | Working | Identifies known types |
| Telemetry broadcast parsing | Partial | Structure known, fields unclear |
| Command construction | Untested | Code exists, never sent to device |
| Response handling | Minimal | Logs messages, no action taken |
| Device control | Not implemented | Awaiting protocol understanding |

## References

- [How It Works: Certificate Injection](../how-it-works/certificate-injection.md) - How we communicate with devices
- [Architecture Overview](architecture.md) - System components
- [Research Background](research-background.md) - Reverse engineering journey
- [WebSocket RFC 6455](https://tools.ietf.org/html/rfc6455) - WebSocket protocol specification

---

[:material-arrow-left: Previous: Research Background](research-background.md){ .md-button }
[:material-arrow-right: Next: Certificate Details](certificate-details.md){ .md-button .md-button--primary }
