# Unrecognized Firmware

What to do when `smartap-jtag detect-firmware` doesn't recognize your device.

## What "Unrecognized" Means

The smartap-jtag tool identifies firmware by checking specific memory addresses for known signatures. If your firmware version hasn't been analyzed yet, the tool can't find the function addresses it needs.

**This doesn't mean your device is broken.** It just means developers need a memory dump from your device to add support.

!!! tip "Want to understand how detection works?"
    See [How It Works: Firmware Detection](../how-it-works/firmware-detection.md) for a detailed explanation of the signature-based detection system.

## Why This Happens

Smartap devices shipped with various firmware versions. The project currently supports:

- Firmware 0x355 (most common)
- (More versions being added as users contribute)

If your device has a different version, you can help by submitting a memory dump.

---

## ⚠️ CRITICAL: WiFi Credentials Warning

!!! danger "Your Memory Dump Contains Your WiFi Password"
    The memory dump includes your device's stored WiFi credentials **in plaintext**. If you submit a dump from a device connected to your home WiFi, you're sharing your WiFi password publicly.

### How to Safely Submit

**Before creating the dump:**

1. **Create a temporary WiFi network**
   - Use your phone as a mobile hotspot
   - Or create a guest network on your router with a throwaway password

2. **Connect your Smartap device to the temporary network**
   - Use `smartap-cfg` to change the WiFi settings
   - Or put device in pairing mode and configure new WiFi

3. **Create the memory dump**
   - Now the dump contains only the throwaway credentials

4. **Submit the dump**
   - Safe to share publicly on GitHub

5. **Reconnect to your real WiFi**
   - After submission, reconfigure to your home network

---

## Step-by-Step: Submitting Your Firmware

### Step 1: Set Up Temporary WiFi

```bash
# Connect device to temporary WiFi using smartap-cfg
smartap-cfg wizard --device <device-ip>
# Navigate to WiFi settings and enter temporary credentials
```

Or put device in pairing mode (hold power button) and connect directly.

### Step 2: Create Memory Dump

Make sure OpenOCD is running and connected to your device.

```bash
# Dump device memory
smartap-jtag dump-memory --output firmware.bin
```

Expected output:
```
✓ Memory dump complete
  Output File:   firmware.bin
  File Size:     256 KB (verified)
```

Verify the file:
```bash
ls -la firmware.bin
# Should show 262144 bytes (exactly 256KB)
```

### Step 3: Create GitHub Issue

Go to: [Create Firmware Submission Issue](https://github.com/muurk/smartap/issues/new?template=firmware-submission.md)

The issue template will guide you through:

1. Confirming you used temporary WiFi credentials
2. Providing device information
3. Pasting the `detect-firmware` output
4. Attaching the `firmware.bin` file

### Step 4: Wait for Analysis

A maintainer will:

1. Download your memory dump
2. Analyze it with tools like Ghidra
3. Identify the SimpleLink function addresses
4. Add your firmware version to the catalog
5. Release a new version of smartap-jtag

You'll be notified on the GitHub issue when support is added.

### Step 5: Re-run With New Version

Once support is added:

```bash
# Download updated smartap-jtag
wget https://github.com/muurk/smartap/releases/latest/download/smartap-jtag-linux-arm64

# Try detection again
smartap-jtag detect-firmware
# Should now show 100% confidence

# Proceed with certificate injection
smartap-jtag inject-certs
```

---

## What Information Is Needed

The memory dump contains everything needed:

| Data | Location | Purpose |
|------|----------|---------|
| Firmware version | Header area | Identification |
| sl_FsOpen address | Code section | File operations |
| sl_FsWrite address | Code section | Writing certificates |
| sl_FsRead address | Code section | Reading files |
| sl_FsClose address | Code section | Closing files |
| sl_FsDel address | Code section | Deleting old cert |

All of these can be identified by analyzing the dump.

---

## Alternative: Manual Analysis

If you're comfortable with reverse engineering, you can analyze the dump yourself:

1. Load `firmware.bin` into Ghidra or IDA Pro
2. Set base address to `0x20000000`
3. Set processor to ARM Cortex-M4
4. Find SimpleLink function references by searching for strings like `sl_FsOpen`
5. Document the addresses and submit a pull request

See [How It Works: Adding New Firmware](../how-it-works/adding-firmware.md) for the complete step-by-step process, or [Firmware Analysis Guide](../contributing/firmware-analysis.md) for Ghidra-specific techniques.

---

## Frequently Asked Questions

### Can I use the device while waiting?

**For basic outlet control:** Yes! The `smartap-cfg` tool works without jailbreaking.

**For smart features:** No - these require certificate injection.

### How long does analysis take?

Typically 1-7 days, depending on maintainer availability.

### What if my firmware is completely different?

If the CC3200 memory layout is different, it may take longer to analyze. Very old or very new firmware versions might require additional research.

### Can I help speed things up?

Yes! If you have reverse engineering experience, you can analyze the dump yourself and submit a PR with the function addresses. See [Firmware Analysis Guide](../contributing/firmware-analysis.md).

---

[:material-arrow-left: Previous: Using smartap-jtag](using-smartap-jtag.md){ .md-button }
[:material-arrow-right: Next: Troubleshooting](troubleshooting.md){ .md-button .md-button--primary }
