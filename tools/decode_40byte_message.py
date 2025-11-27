#!/usr/bin/env python3
"""
Decode the 40-byte message that appears 2,024 times in our captures.
This is 97% of all messages, so understanding it is critical.
"""

import struct

# The one and only 40-byte pattern (2,024 occurrences)
hex_data = "7e03ffffff0f1e0001110f0000000800008055030000507d6dca1200000000000000000000000029"
data = bytes.fromhex(hex_data)

print("=" * 80)
print("DECODING THE 40-BYTE MESSAGE (2,024 occurrences = 97% of all messages)")
print("=" * 80)
print()

print("Raw hex:")
print(hex_data)
print()

print("Byte-by-byte breakdown:")
print()

# Protocol frame header (8 bytes)
print("[PROTOCOL FRAME HEADER - 8 bytes]")
print(f"  [0]     0x{data[0]:02x}         Sync byte (expected 0x7e)")
print(f"  [1]     0x{data[1]:02x}         Version (expected 0x03)")

msg_id = struct.unpack('<I', data[2:6])[0]
print(f"  [2-5]   {data[2:6].hex()}   Message ID = {msg_id} (0x{msg_id:08x}) little-endian")

length = struct.unpack('<H', data[6:8])[0]
print(f"  [6-7]   {data[6:8].hex()}       Length = {length} bytes (0x{length:02x}) little-endian")
print()

# Payload (starts at byte 8)
payload = data[8:]
print(f"[PAYLOAD - {len(payload)} bytes (matches length field)]")
print(f"  Raw payload: {payload.hex()}")
print()

print("[PAYLOAD DECODED]")
print(f"  [8]     0x{payload[0]:02x}         **Message Type**")
print(f"  [9]     0x{payload[1]:02x}         Field 1")
print(f"  [10]    0x{payload[2]:02x}         Field 2")
print()

print("  [11-18] (8 bytes):")
for i in range(8):
    print(f"    [{11+i}]   0x{payload[3+i]:02x}")
print()

print(f"  [19]    0x{payload[11]:02x}         **Nested message type?**")
print(f"  [20]    0x{payload[12]:02x}         Nested field 1")
print(f"  [21-22] {payload[13:15].hex()}       Nested field 2-3")
print()

print("  [23-30] (8 bytes):")
for i in range(8):
    print(f"    [{23+i}]   0x{payload[15+i]:02x}")
print()

print(f"  [31]    0x{payload[23]:02x}         **Possible message type?**")
print()

# Last byte
print(f"[TRAILING BYTE]")
print(f"  [39]    0x{data[39]:02x}         Last byte (0x29 = telemetry type from Ghidra)")
print()

print("=" * 80)
print("GHIDRA CROSS-REFERENCE")
print("=" * 80)
print()

print("Message type 0x01 at offset 8:")
print("  - NOT found in FUN_00006546 (that writes 0x42 then 0x01)")
print("  - Must be a DIFFERENT constructor function")
print()

print("Byte 0x55 at offset 19:")
print("  - Matches type 0x55 (PressureMode) from Ghidra line 4762")
print("  - FUN at line 4762: local_28 = 0x55; local_27 = 4; local_26 = value")
print("  - Our data: [19]=0x55, [20]=0x03, [21-22]=0x00 0x00")
print()

print("Byte 0x29 at offset 39:")
print("  - Matches type 0x29 (Telemetry) from Ghidra line 2949")
print("  - This is the TRAILING byte, might be a marker or checksum")
print()

print("=" * 80)
print("WORKING HYPOTHESIS")
print("=" * 80)
print()

print("This message has LAYERED structure:")
print()
print("  Layer 1: Protocol frame (0x7e 0x03 + ID + length)")
print("  Layer 2: Envelope type 0x01 (offset 8)")
print("           └─ Fields: 0x11 0x0f at offsets 9-10")
print("           └─ 8 unknown bytes at offsets 11-18")
print("  Layer 3: Nested PressureMode type 0x55 (offset 19)")
print("           └─ Subtype: 0x03 at offset 20")
print("           └─ Fields: at offsets 21-30")
print("  Layer 4: Trailing marker 0x29 (telemetry indicator?)")
print()

print("OFFSET 10 vs OFFSET 19:")
print("  - We previously thought nested message at offset 10")
print("  - But 0x0f at offset 10 is NOT a known message type")
print("  - Actual nested message (0x55) is at offset 19")
print("  - Offset 19 = 8 (frame header) + 11 (envelope header)")
print()

print("NEXT STEP:")
print("  Search Ghidra for function that writes:")
print("  1. Type 0x01 as FIRST payload byte")
print("  2. Followed by 0x11 0x0f")
print("  3. Embeds type 0x55 message at offset 11 within envelope")
