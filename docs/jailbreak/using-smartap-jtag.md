# Using smartap-jtag

## What is smartap-jtag?

`smartap-jtag` is a command-line utility that communicates with your Smartap device through a JTAG debug interface. It works by sending GDB (GNU Debugger) scripts to an OpenOCD server, which translates these commands into JTAG signals that interact with the device's CC3200 microcontroller.

**Key points:**

- **smartap-jtag does NOT communicate directly with the device** - it talks to OpenOCD, which handles the JTAG protocol
- **Can run locally or remotely** - smartap-jtag can run on the same Raspberry Pi as OpenOCD, or on a different computer with network access to the OpenOCD server
- **Requires `arm-none-eabi-gdb`** - This is the only local dependency. It must be installed on the machine running smartap-jtag
- **Firmware-aware** - The tool contains a catalog of known firmware versions with their specific memory addresses. It will refuse to operate on unrecognized firmware to prevent corruption

## Prerequisites Workflow

Before performing any operations, you should run these commands in order:

1. **`verify-setup`** - Checks that `arm-none-eabi-gdb` is installed locally and can communicate with the OpenOCD instance
2. **`detect-firmware`** - Identifies your device's firmware version and confirms it's in the known catalog

Only after both checks pass should you proceed to certificate injection or other operations.

!!! warning "Why Firmware Detection Matters"
    Each firmware version stores functions at different memory addresses. If smartap-jtag doesn't recognize your firmware, it cannot safely call filesystem functions - doing so could corrupt your device. The tool enforces 100% confidence before allowing write operations.

## Global Flags

These flags work with all commands:

| Flag | Default | Description |
|------|---------|-------------|
| `--openocd-host` | `localhost` | OpenOCD server hostname |
| `--openocd-port` | `3333` | OpenOCD GDB port |
| `--gdb-path` | `arm-none-eabi-gdb` | Path to GDB binary |
| `--timeout` | `5m` | Operation timeout |
| `-v, --verbose` | `false` | Show detailed GDB output |

## Commands

### verify-setup

**Purpose**: Verify that your local system can communicate with the OpenOCD instance.

**This should be your first command** - run it before attempting any device operations.

```bash
smartap-jtag verify-setup
```

**What it checks:**

1. **GDB binary** - Confirms `arm-none-eabi-gdb` (or the binary specified with `--gdb-path`) is installed and executable
2. **OpenOCD connection** - Attempts to connect to the OpenOCD server and confirms it responds

**Example output (success):**

```
╭─────────────────────────────────────────────────────────────────╮
│ Setup Verification                                              │
│ smartap-jtag verify-setup                                       │
├─────────────────────────────────────────────────────────────────┤
│ GDB Path:     arm-none-eabi-gdb                                 │
│ OpenOCD Host: localhost:3333                                    │
╰─────────────────────────────────────────────────────────────────╯

✓ Setup verification complete
  GDB:     arm-none-eabi-gdb (found)
  OpenOCD: localhost:3333 (connected)
  Status:  Ready for JTAG operations

⚠ Firmware not yet detected
  Next step: Run 'smartap-jtag detect-firmware'
```

**Example output (failure):**

```
✗ Setup verification failed
  OpenOCD: connection refused

  Troubleshooting:
  • Ensure OpenOCD is running: openocd -f <your-config.cfg>
  • Check OpenOCD is listening on the correct host/port
  • Verify firewall settings allow connection
```

---

### detect-firmware

**Purpose**: Identify the firmware version running on your device and confirm it's supported.

**Why this is essential**: The smartap-jtag tool needs to know exact memory addresses to call filesystem functions on the device. These addresses vary between firmware versions. Running operations with wrong addresses could corrupt your device's flash storage.

```bash
smartap-jtag detect-firmware
```

**How it works:**

1. Connects to the device via GDB/OpenOCD
2. Reads memory at specific "signature" locations
3. Compares signatures against the built-in firmware catalog
4. Reports confidence level (percentage of signatures matched)

**100% confidence required**: Certificate injection and file operations are blocked unless firmware detection achieves 100% confidence. This is a safety measure.

**If your firmware is unknown**: See [Unrecognized Firmware](unrecognized-firmware.md) for how to submit a memory dump so your version can be added to the catalog.

**Example output (known firmware):**

```
╭─────────────────────────────────────────────────────────────────╮
│ Firmware Detection                                              │
│ smartap-jtag detect-firmware                                    │
├─────────────────────────────────────────────────────────────────┤
│ Device: localhost:3333                                          │
│ Method: Signature matching                                      │
╰─────────────────────────────────────────────────────────────────╯

Please wait... Detecting firmware (this may take up to 30 seconds)

✓ Firmware detected
  Version:    0x355
  Name:       CC3200 v3.5.5
  Confidence: 100% (all 6 signatures matched)
  Status:     Verified
```

**Example output (unknown firmware):**

```
✗ Firmware unknown
  No known firmware signatures matched

  Troubleshooting:
  • Dump device memory: smartap-jtag dump-memory --output firmware.bin
  • Analysis guide: https://muurk.github.io/smartap/technical/contributing/
  • Submit findings: https://github.com/muurk/smartap/issues/new?template=firmware-submission.md

⚠ JTAG operations blocked
  Reason: 100% confidence required to ensure correct function addresses
  Risk:   Without reliable addresses, operations may corrupt device memory
```

!!! warning "100% Confidence Required"
    Certificate injection only works with 100% firmware confidence. If your firmware isn't recognized, see [Unrecognized Firmware](unrecognized-firmware.md).

---

### inject-certs

**Purpose**: Replace the CA (Certificate Authority) certificate stored on the device.

**Background**: Your Smartap device originally shipped with a Comodo root CA certificate. The device uses this CA to validate the TLS certificate of any server it connects to. Since the original Smartap servers are offline (and their certificates expired), the device can no longer establish trusted connections.

**What this command does**: Injects a new root CA certificate (by default, the embedded Smartap project CA) into the device's flash storage at `/cert/129.der`. After injection, the device will trust servers presenting certificates signed by this new CA.

**This enables**: Running your own server with a certificate signed by the injected CA, allowing the device to connect to your local infrastructure instead of the defunct cloud service.

```bash
# Inject the embedded Root CA (recommended)
smartap-jtag inject-certs

# Inject a custom certificate
smartap-jtag inject-certs --cert-file /path/to/custom-ca.der
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--cert` | `root_ca` | Embedded certificate name |
| `--cert-file` | | Custom certificate file (overrides --cert) |
| `--target-file` | `/cert/129.der` | Target file path on device |
| `--no-detect` | `false` | Skip firmware detection |
| `--firmware-version` | | Firmware version (required if --no-detect) |

**What it does:**

1. Validates prerequisites (GDB, OpenOCD)
2. Loads certificate (embedded or custom)
3. Detects firmware version (unless --no-detect)
4. Prompts for confirmation
5. Deletes existing certificate from device
6. Creates new certificate file
7. Writes certificate data
8. Verifies write success

**Example output:**

```
╭─────────────────────────────────────────────────────────────────╮
│ Certificate Injection                                           │
│ smartap-jtag inject-certs                                       │
├─────────────────────────────────────────────────────────────────┤
│ Device:      localhost:3333                                     │
│ Certificate: Embedded: root_ca                                  │
│ Target:      /cert/129.der                                      │
╰─────────────────────────────────────────────────────────────────╯

  Firmware verified: 0x355  ✓  (100% confidence)

⚠ Warning: This will write to device flash memory
  Target: /cert/129.der

  Continue? [y/N]: y

Please wait... Injecting certificate (this may take up to 60 seconds)

  Halting device  ✓
  Deleting old certificate  ✓
  Creating new file  ✓
  Writing certificate data  ✓  (1501 bytes)
  Closing file  ✓
  Resuming device  ✓

✓ Certificate injection complete
  Target File:   /cert/129.der
  Bytes Written: 1501
  Firmware:      0x355 (verified)
  Duration:      12.3s
  Device:        Resumed and detached
```

!!! tip "After Injection"
    After successful injection:

    1. The device will continue running (it was resumed)
    2. OpenOCD connection may drop (this is normal)
    3. Power cycle the device to use the new certificate
    4. Configure the device to point to your server

---

### dump-memory

**Purpose**: Dump the device's RAM contents to a file for analysis.

**When to use this**:

1. **Unrecognized firmware** - If `detect-firmware` doesn't recognize your device, dump the memory and submit it so developers can add support for your firmware version
2. **Debugging** - Capture device state for troubleshooting
3. **Research** - Analyze firmware behavior

**Key point**: Unlike other operations, `dump-memory` does NOT require recognized firmware. It simply reads raw memory and doesn't call any device functions that could cause corruption.

**What it captures**: The CC3200's 256KB RAM region (0x20000000 - 0x20040000), which contains the running firmware code and data.

```bash
smartap-jtag dump-memory --output firmware.bin
```

**Flags:**

| Flag | Required | Description |
|------|----------|-------------|
| `--output` | Yes | Output file path |

**Example output:**

```
╭─────────────────────────────────────────────────────────────────╮
│ Memory Dump                                                     │
│ smartap-jtag dump-memory                                        │
├─────────────────────────────────────────────────────────────────┤
│ Device:  localhost:3333                                         │
│ Address: 0x20000000 - 0x20040000                                │
│ Size:    256 KB                                                 │
│ Output:  firmware.bin                                           │
╰─────────────────────────────────────────────────────────────────╯

Please wait... Dumping device memory (this may take up to 2 minutes)

✓ Memory dump complete
  Output File:   firmware.bin
  File Size:     256 KB (verified)
  Start Address: 0x20000000
  End Address:   0x20040000
  Duration:      45.2s

⚠ Next steps for firmware analysis
  Step 1: Create issue: https://github.com/muurk/smartap/issues/new?template=firmware-submission.md
  Step 2: Attach the memory dump file
  Step 3: Include device model information
```

!!! danger "WiFi Credentials Warning"
    Memory dumps contain your WiFi SSID and password in plaintext. Before dumping:

    1. Connect device to a **temporary WiFi** (phone hotspot)
    2. Use credentials you don't mind sharing publicly
    3. Then create the dump

    See [Unrecognized Firmware](unrecognized-firmware.md) for details.

---

### read-file

Read a file from the device filesystem.

```bash
smartap-jtag read-file --remote-file /cert/129.der --output cert.der
```

**Flags:**

| Flag | Required | Description |
|------|----------|-------------|
| `--remote-file` | Yes | File path on device |
| `--output` | Yes | Output file path |
| `--max-size` | No | Max file size (default: 256KB) |

**What it does:**

1. Detects firmware version (required for function addresses)
2. Uses SimpleLink sl_FsRead to read the file
3. Saves to specified output file

**Example output:**

```
╭─────────────────────────────────────────────────────────────────╮
│ File Read                                                       │
│ smartap-jtag read-file                                          │
├─────────────────────────────────────────────────────────────────┤
│ Device:      localhost:3333                                     │
│ Remote File: /cert/129.der                                      │
│ Output:      cert.der                                           │
│ Max Size:    256 KB                                             │
╰─────────────────────────────────────────────────────────────────╯

Please wait... Reading file from device (this may take up to 60 seconds)

✓ File read complete
  Remote File: /cert/129.der
  Output File: cert.der
  Bytes Read:  1501 (1 KB)
  Firmware:    0x355 (100% confidence)
  Duration:    8.7s
```

**Common files:**

| Path | Description |
|------|-------------|
| `/cert/129.der` | CA root certificate |
| `/sys/mcuimg0.bin` | Firmware image |

---

### capture-logs

!!! warning "Not Implemented"
    This command is not available in the current version. It's listed in the CLI but returns a "not implemented" message.

---

## Complete Workflow: Replacing Your Device's CA Certificate

This is the full process to replace the CA certificate on your Smartap device, enabling it to connect to your own server.

### Step 1: Start OpenOCD on the Raspberry Pi

OpenOCD must be running and connected to your device via JTAG. On the Raspberry Pi:

```bash
cd /path/to/smartap/hardware/openocd
openocd -f sysfsgpio-smartap.cfg \
        -c "transport select jtag" \
        -c "bindto 0.0.0.0" \
        -f cc3200-complete.cfg
```

**Keep this terminal open** - OpenOCD needs to run throughout the entire process.

**What to look for**: You should see "Listening on port 3333 for gdb connections" and "JTAG tap: cc32xx.jrc tap/device found".

### Step 2: Verify Your Setup

From your workstation (or the Pi itself if running locally):

```bash
smartap-jtag verify-setup --openocd-host <raspberry-pi-ip>
```

Replace `<raspberry-pi-ip>` with your Pi's IP address, or use `localhost` if running on the Pi.

**What to look for**: "Setup verification complete" with both GDB and OpenOCD showing as connected.

**If this fails**: Check that OpenOCD is running, the IP is correct, and port 3333 is accessible (firewall settings).

### Step 3: Detect Your Firmware

```bash
smartap-jtag detect-firmware --openocd-host <raspberry-pi-ip>
```

**What to look for**: "Firmware detected" with 100% confidence.

**If confidence is below 100%**: Your firmware version isn't in the catalog yet. You'll need to:

1. Run `dump-memory` to capture your firmware
2. Submit it following the [Unrecognized Firmware](unrecognized-firmware.md) guide
3. Wait for your version to be added to the catalog

### Step 4: Inject the Certificate

Once you have 100% firmware confidence:

```bash
smartap-jtag inject-certs --openocd-host <raspberry-pi-ip>
```

The tool will:

1. Confirm the firmware version
2. Ask you to confirm the flash write operation
3. Delete the old certificate
4. Write the new certificate
5. Resume the device

**What to look for**: "Certificate injection complete" with bytes written confirmation.

### Step 5: Power Cycle and Configure

After successful injection:

1. **Power cycle the device** - Turn it off and on again
2. **Configure the server address** - Use `smartap-cfg` to point the device to your server:
   ```bash
   smartap-cfg set-server your-server.local 443 --device <device-ip>
   ```
3. **Set up DNS** - Ensure `evalve.smartap-tech.com` resolves to your server's IP
4. **Start your server** - Run `smartap-server` with appropriate certificates

Your device should now connect to your local server instead of the defunct cloud service.

---

## Troubleshooting

### "Connection refused" to OpenOCD

- Verify OpenOCD is running: `ps aux | grep openocd`
- Check binding: OpenOCD must have `-c "bindto 0.0.0.0"` for remote connections
- Check firewall: Port 3333 must be open

### "Firmware unknown"

- Your firmware version isn't in the catalog
- Run `dump-memory` and submit the dump
- See [Unrecognized Firmware](unrecognized-firmware.md)

### Certificate injection fails

- Ensure device is freshly booted
- OpenOCD connection may have dropped - restart it
- Check device hasn't reset (LED behavior)
- Run with `--verbose` for detailed GDB output

### GDB "target remote" fails

- OpenOCD may have lost connection to device
- Check JTAG wiring
- Power cycle device and restart OpenOCD

---

[:material-arrow-left: Previous: Hardware Setup](hardware-setup.md){ .md-button }
[:material-arrow-right: Next: Unrecognized Firmware](unrecognized-firmware.md){ .md-button .md-button--primary }
