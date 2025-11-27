# Firmware Analysis

Guide for analyzing memory dumps to add support for new firmware versions.

!!! tip "Start with the overview"
    This page focuses on Ghidra-specific techniques. For the complete process including YAML structure and testing, see [How It Works: Adding New Firmware](../how-it-works/adding-firmware.md).

## Background

The `smartap-jtag` tool needs specific memory addresses to interact with the device's filesystem. These addresses vary between firmware versions.

When `detect-firmware` fails to recognize a device, someone needs to analyze a memory dump to find the correct addresses.

## What We're Looking For

The device uses Texas Instruments SimpleLink SDK. We need to find these function addresses:

| Function | Purpose |
|----------|---------|
| `sl_FsOpen` | Open a file on the device |
| `sl_FsClose` | Close a file handle |
| `sl_FsRead` | Read from a file |
| `sl_FsWrite` | Write to a file |
| `sl_FsDel` | Delete a file |

We also need:

- Firmware version identifier
- Memory layout validation

## Prerequisites

### Software

- **Ghidra** (free, recommended) - [ghidra-sre.org](https://ghidra-sre.org/)
- Or **IDA Pro** (commercial)
- Or **Binary Ninja** (commercial)

### Knowledge

Helpful to know:

- ARM Cortex-M4 architecture
- Basic reverse engineering concepts
- C function calling conventions
- Embedded systems basics

## Analysis Process

### Step 1: Obtain Memory Dump

If you're analyzing your own device:

```bash
smartap-jtag dump-memory --output firmware.bin
```

If analyzing someone else's submission, download their dump from the GitHub issue.

### Step 2: Load into Ghidra

1. Create a new project
2. Import the `firmware.bin` file
3. When prompted for processor:
   - **Language:** ARM:LE:32:Cortex
   - **Compiler:** default
4. Set base address to `0x20000000` (CC3200 RAM start)

### Step 3: Initial Analysis

Let Ghidra run auto-analysis:

1. **File → Auto Analyze**
2. Enable all analyzers
3. Wait for completion (may take several minutes)

### Step 4: Find String References

Search for SimpleLink strings:

1. **Search → For Strings**
2. Look for:
   - `sl_FsOpen`
   - `sl_FsWrite`
   - Error strings containing "SL_"

These may not exist as direct strings, but error handlers often reference them.

### Step 5: Identify Function Patterns

SimpleLink functions have recognizable patterns:

**sl_FsOpen signature:**
```c
int sl_FsOpen(unsigned char *pFileName, unsigned long AccessModeAndMaxSize, unsigned long *pToken, long *pFileHandle)
```

Look for:

- Functions taking 4 arguments
- First argument is pointer to string
- Returns integer (error code)
- Called before file read/write operations

**Pattern hints:**

- `sl_FsOpen` is called before any file operation
- `sl_FsClose` is called with single argument (file handle)
- `sl_FsRead` takes buffer pointer and size
- `sl_FsWrite` similar to read

### Step 6: Cross-Reference Known Firmware

Compare against known firmware (0x355):

```
sl_FsOpen:  0x200094a0
sl_FsClose: 0x2000aa30
sl_FsRead:  0x2000bba0
sl_FsWrite: 0x2000bd90
sl_FsDel:   0x2000abc0
```

New firmware versions often have:

- Same function order
- Similar offsets between functions
- Recognizable patterns at function entry points

### Step 7: Verify Function Signatures

For each candidate function:

1. Check argument count matches expected
2. Look for characteristic operations
3. Verify return value handling

**sl_FsWrite verification:**

- Takes file handle, buffer, length, offset
- Writes to flash storage
- Returns bytes written or negative error

### Step 8: Find Version Identifier

Look for:

- Version strings like "3.5.5"
- Hex version values (0x355 = 3.5.5)
- Build timestamps
- Firmware header structures

Common locations:

- Start of RAM dump
- Near reset vector
- In firmware metadata section

### Step 9: Create Signature Entry

Document your findings in a format like:

```yaml
firmware:
  version: 0x360
  name: "CC3200 v3.6.0"
  signatures:
    - address: 0x200094a0
      value: 0xb5102de9
      name: "sl_FsOpen prologue"
    - address: 0x2000aa30
      value: 0xb5102de9
      name: "sl_FsClose prologue"
  functions:
    sl_FsOpen: 0x200094a0
    sl_FsClose: 0x2000aa30
    sl_FsRead: 0x2000bba0
    sl_FsWrite: 0x2000bd90
    sl_FsDel: 0x2000abc0
```

### Where to Add Your Firmware Definition

The firmware catalog lives in the Go source code at:

```
internal/gdb/firmware/catalog.yaml
```

If submitting via pull request, add your firmware entry to this file. The structure must match existing entries exactly.

If submitting via GitHub issue, include your findings in the YAML format shown above and a maintainer will add it to the catalog.

## Submitting Your Analysis

### Option 1: GitHub Issue

If you're not comfortable with code changes:

1. Open an issue titled "[FIRMWARE] Analysis for version X.X.X"
2. Include:
   - Memory dump (with clean WiFi credentials)
   - Your function addresses
   - Confidence level for each
   - Any notes or uncertainties

### Option 2: Pull Request

If you can modify code:

1. Add firmware definition to `internal/firmware/catalog.go`
2. Add test case if possible
3. Submit PR with analysis notes

## Tips

### Finding sl_FsOpen

This is usually the easiest to find:

1. Look for code that constructs filename strings like `/cert/129.der`
2. Find the function that receives this string
3. That function likely calls sl_FsOpen internally

### Using Known Patterns

ARM Cortex-M4 function prologues often look like:

```
PUSH {R4-R7, LR}  ; Save registers
; ... function body ...
POP {R4-R7, PC}   ; Return
```

Hex: `2DE9F041` or similar

### When Stuck

- Compare side-by-side with known firmware
- Look for library code (usually at consistent offsets)
- Check for SimpleLink SDK version strings
- Ask in GitHub issues for help

## Reference

### CC3200 Memory Map

| Region | Address Range | Size |
|--------|---------------|------|
| ROM | 0x00000000 - 0x00080000 | 512KB |
| **SRAM** | 0x20000000 - 0x20040000 | 256KB |
| Peripherals | 0x40000000+ | Various |

Memory dumps are SRAM only (where firmware runs from).

### SimpleLink SDK Functions

For reference, from TI SDK documentation:

```c
_i32 sl_FsOpen(_u8 *pFileName, _u32 AccessModeAndMaxSize, _u32 *pToken, _i32 *pFileHandle);
_i32 sl_FsClose(_i32 FileHdl, _u8 *pCertificateChainName, _u8 *pCertificateFileName, _u32 Signature);
_i32 sl_FsRead(_i32 FileHdl, _u32 Offset, _u8 *pData, _u32 Len);
_i32 sl_FsWrite(_i32 FileHdl, _u32 Offset, _u8 *pData, _u32 Len);
_i32 sl_FsDel(_u8 *pFileName, _u32 Token);
```

## Questions?

- Open a GitHub issue with questions
- Share partial progress - others may help
- No contribution is too small

---

[:material-arrow-left: Previous: Contributing Overview](overview.md){ .md-button }
[:material-home: Back to Technical Docs](../technical/architecture.md){ .md-button .md-button--primary }
