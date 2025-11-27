# Firmware Detection

Before we can inject a certificate, we need to know exactly which firmware version we're dealing with. Get this wrong, and we risk calling functions at the wrong addresses—potentially bricking the device.

This page explains how the signature-based detection system works.

## The Problem

You might expect firmware to have a version number at a fixed memory address. It doesn't. Or rather, it might, but that address varies between versions. We can't read the version without already knowing where to look.

We need a way to identify firmware that doesn't depend on knowing anything about it in advance.

## The Solution: Signatures

We borrow a technique from antivirus software: **signature matching**.

Every function in the firmware starts with a sequence of machine code instructions. These bytes are deterministic—the same function compiled the same way produces the same bytes. We call the first 8 bytes of a function its "signature."

```
Function: sl_FsOpen (firmware 0x355)
Address:  0x20015c64

Memory contents at that address:
┌────────────────────────────────────────┐
│ 0x4606b570  0x78004818  ...            │
└────────────────────────────────────────┘
     ▲            ▲
     │            │
     └────────────┴─── These 8 bytes are the signature
```

Different firmware versions have the same functions, but at different addresses with potentially different compiled code. The signature changes.

## Why Seven Functions?

We don't rely on a single signature. The detection system checks seven functions from the SimpleLink SDK:

| Function | Purpose |
|----------|---------|
| `sl_FsOpen` | Open or create a file |
| `sl_FsRead` | Read from a file |
| `sl_FsWrite` | Write to a file |
| `sl_FsClose` | Close a file handle |
| `sl_FsDel` | Delete a file |
| `sl_FsGetInfo` | Get file metadata |
| `uart_log` | Internal logging function |

Why seven? Statistical confidence. If one signature matches by coincidence, that's possible. If all seven match, we're certain. This is a safety margin—we're about to manipulate flash storage, and we need to be sure.

## The Detection Process

Here's what happens when you run `smartap-jtag detect-firmware`:

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. Connect to device via OpenOCD/GDB                            │
├─────────────────────────────────────────────────────────────────┤
│ 2. Halt the processor                                           │
├─────────────────────────────────────────────────────────────────┤
│ 3. For each known firmware version:                             │
│    ┌─────────────────────────────────────────────────────────┐  │
│    │ For each of the 7 functions:                            │  │
│    │   • Read 8 bytes at the expected address                │  │
│    │   • Compare against the stored signature                │  │
│    │   • Count matches                                       │  │
│    └─────────────────────────────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│ 4. Calculate confidence: (matches / 7) × 100%                   │
├─────────────────────────────────────────────────────────────────┤
│ 5. If 100% match found → identified                             │
│    If no match → unknown firmware                               │
└─────────────────────────────────────────────────────────────────┘
```

The tool outputs something like:

```
Detecting firmware version...

Checking firmware 0x355... 100% (7/7 signatures matched)

✓ Firmware detected: Smartap 0x355
  Confidence: 100%
  Status: Verified
```

## The 100% Rule

!!! danger "We require 100% confidence"
    The injection process will not proceed unless all seven signatures match. This isn't paranoia—it's the only safe approach.

Consider what happens if we're wrong:

- We call `sl_FsDel` at the wrong address → unknown code executes
- We write certificate data to the wrong memory region → firmware corruption
- We call `sl_FsClose` incorrectly → file left in inconsistent state

Any of these could brick the device. The 100% rule exists because partial matches aren't good enough when the stakes are permanent hardware damage.

## The Firmware Database

Firmware definitions live in a YAML file embedded in the binary:

```yaml
firmwares:
  - version: "0x355"
    name: "Smartap 0x355"
    description: "Primary verified firmware version"
    verified: true
    functions:
      sl_FsOpen: 0x20015c64
      sl_FsRead: 0x20014b54
      sl_FsWrite: 0x20014bf8
      sl_FsClose: 0x2001555c
      sl_FsDel: 0x20016ea8
      sl_FsGetInfo: 0x20016f44
      uart_log: 0x200046d4
    signatures:
      sl_FsOpen: [0x4606b570, 0x78004818]
      sl_FsRead: [0x43f0e92d, 0x48254680]
      # ... remaining signatures
    memory:
      work_buffer: 0x20030000
      file_handle_ptr: 0x20031000
      filename_ptr: 0x20031004
      token_ptr: 0x20031020
      stack_base: 0x20031d00
```

Each entry contains:

- **functions**: The memory addresses of SimpleLink SDK functions
- **signatures**: The expected bytes at each function's address (two 32-bit words)
- **memory**: Safe memory regions for temporary data during injection

## What Happens With Unknown Firmware

If detection fails, you'll see:

```
Detecting firmware version...

Checking firmware 0x355... 28% (2/7 signatures matched)

✗ Firmware not recognised
  No known firmware matched with sufficient confidence.

  See: https://docs.smartap.dev/jailbreak/unrecognized-firmware/
```

This means either:

1. Your device has a firmware version we haven't catalogued yet
2. Something is wrong with the JTAG connection (bad reads)

The first case is common—devices shipped with various firmware versions. The solution is to dump your device's memory and either analyse it yourself or submit it for community analysis.

See [Adding New Firmware](adding-firmware.md) for the complete process.

## Why Not Just Read the Version String?

You might wonder: why not search memory for a version string like "3.5.5"?

We tried. The version string exists, but:

1. Its location varies between firmware versions
2. String searching is slow and unreliable via GDB
3. Even finding the string doesn't tell us function addresses

Signatures solve all three problems. They're fast to check, deterministic, and directly tied to the information we actually need (function locations).

## Technical Details

For those interested in the implementation:

**Signature format**: Two 32-bit little-endian words (8 bytes total)

**Why little-endian**: ARM Cortex-M4 is little-endian. Memory reads return bytes in that order.

**GDB commands used**:
```gdb
# Read 8 bytes as two 32-bit words
set $word1 = *(unsigned int*)0x20015c64
set $word2 = *(unsigned int*)(0x20015c64 + 4)
```

**Early exit optimisation**: Detection stops as soon as a 100% match is found. No need to check remaining firmware versions.

---

[:material-arrow-left: Previous: Overview](overview.md){ .md-button }
[:material-arrow-right: Next: Certificate Injection](certificate-injection.md){ .md-button .md-button--primary }
