# Adding New Firmware Support

When `smartap-jtag detect-firmware` doesn't recognise your device, someone needs to analyse the firmware and add it to the database. This page explains exactly how to do that.

!!! info "Contributing Without Reverse Engineering"
    If you're not comfortable with reverse engineering, you can still help. Submit a memory dump via GitHub issue (with safe WiFi credentials), and someone else can do the analysis. See [Unrecognised Firmware](../jailbreak/unrecognized-firmware.md) for instructions.

## What We Need to Find

For each firmware version, we need:

1. **Function addresses** — Where the 7 SimpleLink SDK functions live in memory
2. **Signatures** — The first 8 bytes at each function address
3. **Memory regions** — Safe locations for temporary data during injection

The first two are the challenging part. The third is usually consistent across versions.

## The Complete Process

### Step 1: Confirm Unknown Firmware

First, verify detection actually fails:

```bash
smartap-jtag detect-firmware
```

If you see `Firmware not recognised` with low confidence scores, proceed.

### Step 2: Create a Memory Dump

Dump the device's RAM:

```bash
smartap-jtag dump-memory --output firmware.bin
```

This creates a 256KB file containing the contents of RAM (0x20000000 - 0x20040000).

!!! danger "WiFi Credentials Warning"
    The dump contains your WiFi password in plaintext. Before sharing publicly, either:

    - Connect the device to a temporary network with throwaway credentials, OR
    - Manually scrub the dump (search for your SSID and password)

### Step 3: Load into Ghidra

[Ghidra](https://ghidra-sre.org/) is a free reverse engineering tool from the NSA. It's excellent for this work.

1. Create a new project
2. Import `firmware.bin`
3. When prompted for processor settings:
   - **Language**: ARM:LE:32:Cortex
   - **Compiler**: default
4. Set the base address to `0x20000000`

Let Ghidra run auto-analysis. This takes a few minutes.

### Step 4: Find Function Addresses

We need to locate these seven functions:

| Function | What It Does | Hints for Finding It |
|----------|--------------|---------------------|
| `sl_FsOpen` | Opens/creates files | Called before any file operation |
| `sl_FsRead` | Reads from files | Takes buffer pointer and size |
| `sl_FsWrite` | Writes to files | Similar signature to Read |
| `sl_FsClose` | Closes file handles | Takes single argument (handle) |
| `sl_FsDel` | Deletes files | Takes filename pointer |
| `sl_FsGetInfo` | Gets file metadata | Returns file size, etc. |
| `uart_log` | Internal logging | Often called with string pointers |

**Strategy 1: String References**

Search for strings that might be passed to these functions:

- `/cert/` — certificate file paths
- `sl_Fs` — error messages mentioning function names
- `.der` — certificate file extensions

In Ghidra: **Search → For Strings**, then look for cross-references.

**Strategy 2: Compare with Known Firmware**

If you have access to a dump from a known firmware (0x355), compare:

- Function order is often preserved
- Relative offsets between functions may be similar
- Instruction patterns at function entry points are recognisable

**Strategy 3: Function Signatures**

ARM Cortex-M4 functions typically start with a push instruction:

```
PUSH {R4-R7, LR}   ; Save registers
```

In hex, this looks like: `2DE9F041` or similar patterns.

Look for functions that:

- Take 4 arguments (sl_FsOpen, sl_FsWrite)
- Take 2 arguments (sl_FsDel)
- Take 1 argument (sl_FsClose first arg is handle)

### Step 5: Capture Signatures

Once you've identified the function addresses, capture the signatures using GDB:

```bash
arm-none-eabi-gdb -q
(gdb) target extended-remote localhost:3333
(gdb) monitor reset halt
```

For each function, read the first 8 bytes:

```gdb
(gdb) x/2xw 0x20015c64
0x20015c64:    0x4606b570    0x78004818
```

Record both hex values. These become the signature.

### Step 6: Create the YAML Entry

Add your firmware to `internal/gdb/firmwares/firmwares.yaml`:

```yaml
- version: "0xNEW"
  name: "Smartap 0xNEW"
  description: "Description of this firmware version"
  verified: false
  functions:
    sl_FsOpen: 0x200XXXXX
    sl_FsRead: 0x200XXXXX
    sl_FsWrite: 0x200XXXXX
    sl_FsClose: 0x200XXXXX
    sl_FsDel: 0x200XXXXX
    sl_FsGetInfo: 0x200XXXXX
    uart_log: 0x200XXXXX
  signatures:
    sl_FsOpen: [0xXXXXXXXX, 0xXXXXXXXX]
    sl_FsRead: [0xXXXXXXXX, 0xXXXXXXXX]
    sl_FsWrite: [0xXXXXXXXX, 0xXXXXXXXX]
    sl_FsClose: [0xXXXXXXXX, 0xXXXXXXXX]
    sl_FsDel: [0xXXXXXXXX, 0xXXXXXXXX]
    sl_FsGetInfo: [0xXXXXXXXX, 0xXXXXXXXX]
    uart_log: [0xXXXXXXXX, 0xXXXXXXXX]
  memory:
    work_buffer: 0x20030000
    file_handle_ptr: 0x20031000
    filename_ptr: 0x20031004
    token_ptr: 0x20031020
    stack_base: 0x20031d00
  notes: |
    Add any relevant notes here:
    - Device serial number pattern
    - Manufacturing date range
    - Any quirks observed
```

!!! tip "Memory Addresses"
    The `memory` section addresses are usually safe to copy from existing firmware entries. They're in a region of RAM that's typically unused. Only change them if you have evidence of conflicts.

### Step 7: Test Detection

Rebuild and test:

```bash
go build ./cmd/smartap-jtag
./smartap-jtag detect-firmware
```

Expected output:

```
Checking firmware 0xNEW... 100% (7/7 signatures matched)

✓ Firmware detected: Smartap 0xNEW
```

If you get less than 100%, one or more signatures are wrong. Double-check your addresses and byte values.

### Step 8: Test Injection

With detection working, test the full injection:

```bash
./smartap-jtag inject-certs
```

Watch for:

- All steps completing successfully
- Correct number of bytes written
- No error messages

If injection fails, the most likely causes are:

- Wrong function address (crashes or hangs)
- Wrong memory region (conflicts with firmware)
- Signature mismatch (shouldn't happen if detection passed)

### Step 9: Mark as Verified and Submit

Once injection succeeds:

1. Set `verified: true` in your YAML entry
2. Add testing notes
3. Submit a pull request

Include in your PR:

- The YAML changes
- Device information (serial pattern, photos if helpful)
- Testing results
- Any quirks or observations

## Common Pitfalls

### Endianness Confusion

ARM is little-endian. When Ghidra shows bytes, they may appear in different order than GDB reports. Always verify with GDB directly.

### Off-by-One Errors

Thumb mode (used by Cortex-M4) has 2-byte aligned instructions. Make sure your addresses are even numbers.

### Wrong Function Identification

The most common mistake is misidentifying a wrapper function as the actual SimpleLink function. Look for the actual implementation, not a thin wrapper that calls it.

### Incomplete Analysis

If you only find 5 of 7 functions, the firmware entry won't work. All seven are required for the 100% confidence check.

## Getting Help

If you're stuck:

1. **Open a GitHub issue** with your partial findings
2. **Share your Ghidra project** (if comfortable)
3. **Post in discussions** with specific questions

Even partial analysis is valuable. Someone else might have the missing pieces.

## Reference: Known Firmware 0x355

For comparison, here are the verified addresses for firmware 0x355:

```yaml
functions:
  sl_FsOpen: 0x20015c64
  sl_FsRead: 0x20014b54
  sl_FsWrite: 0x20014bf8
  sl_FsClose: 0x2001555c
  sl_FsDel: 0x20016ea8
  sl_FsGetInfo: 0x20016f44
  uart_log: 0x200046d4
```

Use these as a reference point. New firmware versions often have similar relative offsets between functions.

---

[:material-arrow-left: Previous: Certificate Injection](certificate-injection.md){ .md-button }
[:material-home: Back to How It Works](overview.md){ .md-button .md-button--primary }
