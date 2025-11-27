# Smartap Protocol Constructor Library

## Overview

This library provides functions to construct protocol messages for communication with the Smartap IoT shower controller device. All message formats are based on verified Ghidra decompilation analysis of the device firmware.

**Key Design Principles:**
- **Interface-agnostic**: Pure message construction, no I/O dependencies
- **Well-documented**: Every function references the Ghidra source
- **Type-safe**: Clear Go interfaces for all message types
- **Validated**: Parameter checking before building messages
- **Testable**: Comprehensive unit test coverage

## Protocol Structure

### Frame Format

Every message sent to the device follows this structure:

```
[0]     0x7e           Sync byte (ProtocolSync)
[1]     0x03           Version byte (ProtocolVersion)
[2-5]   message_id     Message ID (little-endian uint32)
[6-7]   length         Payload length (little-endian uint16)
[8+]    payload        Message payload bytes
[N+]    padding        Zero padding to 38 bytes minimum
```

**Source**: Ghidra `FUN_00006472` @ lines 2456-2490 (header construction), `FUN_0000650e` @ lines 2494-2516 (padding)

**Important Notes:**
- All multi-byte integers use **little-endian** encoding
- Minimum frame size is **38 bytes** (enforced by firmware)
- Message IDs should be unique per request (use `GenerateMessageID()`)
- Reserved message ID `0x0FFFFFFF` is used for device broadcasts

## Message Types Reference

| Type | Hex  | Name                  | Direction      | Purpose                          |
|------|------|-----------------------|----------------|----------------------------------|
| 0x01 | 0x01 | TelemetryBroadcast    | Device→Server  | Unsolicited telemetry updates    |
| 0x05 | 0x05 | OTA                   | Bidirectional  | Firmware update messages         |
| 0x29 | 0x29 | TelemetryResponse     | Device→Server  | Response to telemetry queries    |
| 0x42 | 0x42 | Command               | Server→Device  | Device control and configuration |
| 0x44 | 0x44 | Extended              | Bidirectional  | Extended protocol messages       |
| 0x55 | 0x55 | PressureMode          | Server→Device  | Low pressure mode control        |

## Constructor Functions

### BuildProtocolFrame

**Purpose**: Low-level frame construction with header and padding

**Signature**:
```go
func BuildProtocolFrame(messageID uint32, payload []byte) ([]byte, error)
```

**Parameters**:
- `messageID`: Unique message identifier (use `GenerateMessageID()`)
- `payload`: The message payload bytes (message type + data)

**Returns**:
- Complete protocol frame ready to send via WebSocket
- Error if payload exceeds maximum size (1024 bytes)

**Example**:
```go
payload := []byte{0x42, 0x05, 0x01, 0x34, 0x12, 0x00, 0x00}
frame, err := BuildProtocolFrame(GenerateMessageID(), payload)
if err != nil {
    return err
}
```

**Ghidra Reference**: `FUN_00006472` @ lines 2456-2490, `FUN_0000650e` @ lines 2494-2516

---

### BuildCommandMessage

**Purpose**: Construct device command messages (type 0x42)

**Signature**:
```go
func BuildCommandMessage(messageID uint32, category uint32, data []byte) ([]byte, error)
```

**Payload Structure**:
```
[0]     0x42           Message type
[1]     len(data)+5    Total length + 5
[2]     0x01           Field marker
[3-6]   category       Category/command code (little-endian)
[7+]    data           Variable length command data
```

**Parameters**:
- `messageID`: Unique message ID
- `category`: Command category/code (device-specific)
- `data`: Additional command data (can be empty slice)

**Returns**:
- Complete protocol frame with command message payload
- Error if construction fails

**Example Usage**:
```go
// Send a command with category 0x1234 and two data bytes
msgID := GenerateMessageID()
msg, err := BuildCommandMessage(msgID, 0x1234, []byte{0x01, 0x02})
if err != nil {
    return err
}
err = SendMessage(conn, remoteAddr, msg)
```

**Use Cases**:
- Device control commands
- Configuration requests
- Status queries

**Ghidra Reference**: `FUN_00006546` @ lines 2520-2571

---

### BuildTelemetryQuery

**Purpose**: Request specific telemetry data from device

**Signature**:
```go
func BuildTelemetryQuery(messageID uint32, queryType uint8) ([]byte, error)
```

**Payload Structure**:
```
[0]     0x29           Message type (TelemetryResponse)
[1]     0x11           Subtype (telemetry marker)
[2]     queryType      Which telemetry value to query
[3-18]  zeros          Zero padding
```

**Parameters**:
- `messageID`: Unique message ID
- `queryType`: Type of telemetry to query (device-specific)

**Returns**:
- Complete protocol frame with telemetry query payload
- Error if construction fails

**Example Usage**:
```go
// Query telemetry type 0x80
msgID := GenerateMessageID()
msg, err := BuildTelemetryQuery(msgID, 0x80)
if err != nil {
    return err
}
err = SendMessage(conn, remoteAddr, msg)
```

**Use Cases**:
- Request current sensor readings
- Poll device state
- Health checks

**Ghidra Reference**: Reverse-engineered from `FUN_00006928` @ lines 2921-2955 (response format)

---

### BuildPressureModeSet

**Purpose**: Control low pressure mode feature

**Signature**:
```go
func BuildPressureModeSet(messageID uint32, enabled bool) ([]byte, error)
```

**Payload Structure**:
```
[0]     0x55           Message type
[1]     0x04           Subtype/length indicator
[2]     value          0x00 = disabled, 0x01 = enabled
```

**Parameters**:
- `messageID`: Unique message ID
- `enabled`: true to enable pressure mode, false to disable

**Returns**:
- Complete protocol frame with pressure mode payload
- Error if construction fails

**Example Usage**:
```go
// Enable low pressure mode
msgID := GenerateMessageID()
msg, err := BuildPressureModeSet(msgID, true)
if err != nil {
    return err
}
err = SendMessage(conn, remoteAddr, msg)

// Disable low pressure mode
msg, err = BuildPressureModeSet(GenerateMessageID(), false)
```

**Use Cases**:
- Enable/disable low pressure mode
- System configuration

**Ghidra Reference**: Inline code @ line 4762

---

## Helper Functions

### GenerateMessageID

**Purpose**: Generate unique message IDs for outgoing messages

**Signature**:
```go
func GenerateMessageID() uint32
```

**Behavior**:
- Generates sequential IDs starting from 1
- Skips reserved message ID `0x0FFFFFFF`
- Handles counter overflow (wraps to 1)
- **Thread-safe** using atomic operations

**Returns**: Unique message ID (never returns reserved IDs)

**Example**:
```go
msgID := GenerateMessageID()
msg, err := BuildCommandMessage(msgID, category, data)
```

---

### ValidateFrame

**Purpose**: Validate protocol frame structure

**Signature**:
```go
func ValidateFrame(frame []byte) error
```

**Validation Checks**:
- Minimum frame size (38 bytes)
- Sync byte (0x7e)
- Version byte (0x03)
- Length field matches actual payload
- Message type is recognized

**Parameters**:
- `frame`: The complete protocol frame to validate

**Returns**:
- `nil` if frame is valid
- Error describing what is wrong with the frame

**Example**:
```go
if err := ValidateFrame(frame); err != nil {
    log.Printf("Invalid frame: %v", err)
    return err
}
```

---

### CalculateHeaderChecksum

**Purpose**: Calculate header checksum used by firmware

**Signature**:
```go
func CalculateHeaderChecksum(header []byte) uint8
```

**Algorithm**: Sum of header bytes (0-7) plus 3

**Parameters**:
- `header`: First 8 bytes of protocol frame

**Returns**: Checksum value (sum of header bytes + 3)

**Note**: This checksum does NOT appear in captured protocol frames. It may be used internally by the firmware for validation but not transmitted. Included for completeness based on Ghidra code.

**Ghidra Reference**: Line 2684

---

## WebSocket Integration

### Sending Messages to Device

Use the `SendMessage` function from `server/websocket.go` to send constructed messages:

**Signature**:
```go
func SendMessage(conn net.Conn, remoteAddr string, message []byte) error
```

**Features**:
- Validates message frame before sending
- Wraps message in WebSocket binary frame (opcode 0x82)
- Sets write deadline (10 seconds)
- Logs sent messages with hex dump
- Returns descriptive errors on failure

**Example Usage**:
```go
import (
    "github.com/muurk/smartap/internal/protocol"
    "github.com/muurk/smartap/internal/server"
)

// Enable pressure mode on connected device
msgID := protocol.GenerateMessageID()
msg, err := protocol.BuildPressureModeSet(msgID, true)
if err != nil {
    return fmt.Errorf("failed to build message: %w", err)
}

err = server.SendMessage(conn, remoteAddr, msg)
if err != nil {
    return fmt.Errorf("failed to send message: %w", err)
}
```

---

## Common Patterns

### Request-Response Pattern

```go
// Generate unique ID for correlation
msgID := protocol.GenerateMessageID()

// Build and send request
query, err := protocol.BuildTelemetryQuery(msgID, 0x80)
if err != nil {
    return err
}
err = server.SendMessage(conn, remoteAddr, query)
if err != nil {
    return err
}

// Wait for response with matching msgID
// (Response handling logic in protocol/handler.go)
```

### Command with Data

```go
// Command with variable data payload
category := uint32(0x1234)
data := []byte{0x01, 0x02, 0x03}

msg, err := protocol.BuildCommandMessage(
    protocol.GenerateMessageID(),
    category,
    data,
)
if err != nil {
    return err
}

err = server.SendMessage(conn, remoteAddr, msg)
```

### Simple Toggle Command

```go
// Enable/disable a feature
enabled := true
msg, err := protocol.BuildPressureModeSet(
    protocol.GenerateMessageID(),
    enabled,
)
if err != nil {
    return err
}

err = server.SendMessage(conn, remoteAddr, msg)
```

---

## Future REST/MQTT Interface Example

This library is designed to be interface-agnostic. Here's how it would integrate with a future REST API:

```go
// Future REST endpoint handler
func HandleSetPressureMode(w http.ResponseWriter, r *http.Request) {
    // Parse request
    var req struct {
        Enabled bool `json:"enabled"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Get device connection from registry
    conn := deviceRegistry.GetConnection(deviceID)
    if conn == nil {
        http.Error(w, "device not connected", http.StatusNotFound)
        return
    }

    // Build and send message using constructor library
    msg, err := protocol.BuildPressureModeSet(
        protocol.GenerateMessageID(),
        req.Enabled,
    )
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    err = server.SendMessage(conn, deviceID, msg)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Return success
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "success",
    })
}
```

---

## Testing

Run unit tests:
```bash
cd smartap-server/internal/protocol
go test -v
```

All constructor functions have comprehensive unit tests in `constructor_test.go`:
- `TestBuildProtocolFrame` - Frame structure validation
- `TestBuildCommandMessage` - Command message construction
- `TestBuildTelemetryQuery` - Query message construction
- `TestBuildPressureModeSet` - Pressure mode message construction
- `TestGenerateMessageID` - ID generation and uniqueness
- `TestValidateFrame` - Frame validation logic
- `TestCalculateHeaderChecksum` - Checksum calculation

---

## Ghidra Function Cross-Reference

| Constructor Function      | Ghidra Function | Lines      | Purpose                           |
|---------------------------|-----------------|------------|-----------------------------------|
| BuildProtocolFrame        | FUN_00006472    | 2456-2490  | Frame header construction         |
| BuildProtocolFrame        | FUN_0000650e    | 2494-2516  | Frame padding                     |
| BuildCommandMessage       | FUN_00006546    | 2520-2571  | Command message structure         |
| BuildTelemetryQuery       | FUN_00006928    | 2921-2955  | Telemetry response (reversed)     |
| BuildPressureModeSet      | Inline code     | 4762       | Pressure mode message             |
| CalculateHeaderChecksum   | Checksum code   | 2684       | Header checksum algorithm         |

---

## Known Limitations

1. **OTA messages not implemented**: Firmware update protocol (type 0x05) not yet supported
2. **Extended messages not implemented**: Extended protocol format (type 0x44) not yet supported
3. **No response correlation**: Library builds messages but doesn't track responses
4. **No retry logic**: Caller must implement retry on send failure
5. **No batch sending**: Send one message at a time

These limitations are intentional - the library focuses on pure message construction. Higher-level features (retry, correlation, batching) should be implemented in application layers.

---

## References

- **Ghidra Analysis**: `memory-analysis-export-ghidra.c` - Complete firmware decompilation
- **Protocol Parser**: `parser.go` - Complementary message parsing functions
- **WebSocket Handler**: `../server/websocket.go` - Integration and I/O layer
- **Implementation Plan**: `MESSAGE-CONSTRUCTOR-PLAN.md` - Development roadmap
- **Test Suite**: `constructor_test.go` - Comprehensive unit tests

---

**Version**: 1.0
**Last Updated**: 2025-11-21
**Status**: Production Ready
