# Jailbreak Prerequisites

Everything you need before starting the certificate injection process.

## Hardware Requirements

### Raspberry Pi

Any Raspberry Pi with GPIO headers works:

- **Recommended:** Raspberry Pi 4 or 5 (fastest)
- **Also works:** Raspberry Pi 3, Zero W, or older models
- Must have network access to transfer files

### Jumper Wires

You need **5 female-to-female jumper wires** to connect the Raspberry Pi GPIO header to the JTAG header you'll solder onto the CC3200 board.

!!! note "Header to Header"
    After soldering a male header onto the CC3200 JTAG pads, you'll use female-to-female jumper wires to connect it to the Raspberry Pi's GPIO header.

!!! tip "Dupont Wires"
    Standard "Dupont" jumper wires from any electronics supplier work perfectly. You'll find these in most Arduino/Raspberry Pi starter kits.

### Physical Access to the CC3200 Module

The CC3200 WiFi module is NOT inside the main Smartap control unit. It's housed in a separate sealed plastic case connected to the control unit via a cable labelled "WIFI".

!!! warning "Safety First"
    Ensure the shower is switched OFF at the isolator before disconnecting any cables. Keep it off until you have reconnected everything.

#### Step 1: Locate the Smartap Hardware Unit

The Smartap system consists of:
- **Control unit** - The main box with 4 cables connected to it (valves, power, etc.)
- **CC3200 WiFi module** - A separate sealed plastic case connected via the cable marked "WIFI"

See [this photo of the inside of the main housing](https://community.home-assistant.io/t/smartap-shower-control-getting-started-with-reverse-engineering-a-smart-home-device/358251/24) for reference.

!!! note "You Don't Need to Open the Control Unit"
    You do NOT need to disassemble the main control unit (the box with 4 cables). You only need to disconnect the "WIFI" cable and work with the CC3200 module on the other end.

#### Step 2: Disconnect the CC3200 Module

1. Locate the cable marked "WIFI" connected to the control unit
2. Unscrew the terminal connector
3. Disconnect the cable - this allows you to remove the CC3200 module from its mounting location
4. The CC3200 module is the sealed plastic case on the other end of this cable

#### Step 3: Open the CC3200 Case

The CC3200 is enclosed in a sealed plastic case that is glued shut.

1. Identify the seams where the plastic case is glued together
2. Use a thin blade (craft knife or similar) to carefully pry open the case
3. Work slowly around the seam - the glue may be strong in places

!!! warning "Be Careful"
    - The CC3200 board fits snugly inside the case - don't force anything
    - Avoid damaging the plastic housing - you'll need to seal it again afterwards
    - Don't cut into the board or damage any components

#### Step 4: Access the JTAG Pads

Once the case is open:
- You'll see the CC3200 board with the cable still attached
- The JTAG pads are unpopulated holes on the board
- The cable can be disconnected from the board, but this isn't necessary for soldering
- You need access to both sides of the board to solder the header

See [this photo of the CC3200 board](https://community.home-assistant.io/t/smartap-shower-control-getting-started-with-reverse-engineering-a-smart-home-device/358251/120) for reference. Note: This photo shows the board with the RF shield removed - you do NOT need to remove the RF shield; the JTAG pads are accessible without removing it.

#### Step 5: Solder the JTAG Header

You will need to solder a 2.54mm pitch header to the JTAG pads. See [Hardware Setup](hardware-setup.md) for pin mapping details.

!!! tip "Soldering Tips"
    - Use a header rather than individual wires - this makes reconnection easier
    - The header allows you to disconnect the jumper wires when reassembling
    - If you're not comfortable soldering, ask at a local makerspace or electronics club

!!! question "Not comfortable with this hardware work?"
    If soldering and hardware modification isn't for you, but you have software skills and want to help with the server/protocol development, there may be another way to contribute.

    **[Read: Contributing Without Hardware Modification](../contributing/help-without-hardware.md)**

## Software Requirements

### On the Raspberry Pi

#### OpenOCD

OpenOCD provides the JTAG communication layer.

```bash
# Install on Raspberry Pi OS
sudo apt update
sudo apt install openocd
```

Verify installation:
```bash
openocd --version
# Should show: Open On-Chip Debugger 0.11.x or newer
```

#### arm-none-eabi-gdb

The ARM GDB debugger executes commands on the device.

```bash
# Install on Raspberry Pi OS
sudo apt install gdb-multiarch

# Or install the full ARM toolchain
sudo apt install gcc-arm-none-eabi
```

Verify installation:
```bash
arm-none-eabi-gdb --version
# Or if using gdb-multiarch:
gdb-multiarch --version
```

!!! note "GDB Path"
    If you installed `gdb-multiarch`, you'll need to use `--gdb-path gdb-multiarch` with smartap-jtag commands.

### smartap-jtag Binary

Download the appropriate binary for your Raspberry Pi:

| Raspberry Pi Model | Binary |
|-------------------|--------|
| Pi 4/5 (64-bit OS) | [smartap-jtag-linux-arm64](https://github.com/muurk/smartap/releases/latest/download/smartap-jtag-linux-arm64) |
| Pi 3/4 (32-bit OS) | [smartap-jtag-linux-armv7](https://github.com/muurk/smartap/releases/latest/download/smartap-jtag-linux-armv7) |
| Any x86_64 Linux | [smartap-jtag-linux-amd64](https://github.com/muurk/smartap/releases/latest/download/smartap-jtag-linux-amd64) |

```bash
# Download (example for 64-bit Pi)
wget https://github.com/muurk/smartap/releases/latest/download/smartap-jtag-linux-arm64

# Make executable
chmod +x smartap-jtag-linux-arm64

# Optional: rename for convenience
mv smartap-jtag-linux-arm64 smartap-jtag
```

### OpenOCD Configuration Files

The repository includes pre-configured OpenOCD files:

```bash
# Clone the repository (or just download the hardware/ directory)
git clone https://github.com/muurk/smartap.git

# The config files are in:
# hardware/openocd/sysfsgpio-smartap.cfg
# hardware/openocd/cc3200-complete.cfg
```

## Network Setup

The Raspberry Pi running OpenOCD must be reachable from wherever you run smartap-jtag.

**Option A: Run everything on the Pi**

- Install smartap-jtag on the Pi itself
- Use `--openocd-host localhost` (default)

**Option B: Run smartap-jtag remotely**

- Run OpenOCD on the Pi with `-c "bindto 0.0.0.0"`
- Run smartap-jtag from your laptop with `--openocd-host <pi-ip-address>`

## Checklist

Before proceeding, verify you have:

**Hardware:**

- [ ] Raspberry Pi with network access
- [ ] 5 female-to-female jumper wires
- [ ] Soldering iron and solder
- [ ] 2.54mm pitch header (at least 6 pins)
- [ ] Thin blade or craft knife (to open CC3200 case)
- [ ] Physical access to your Smartap installation

**Software (on Raspberry Pi):**

- [ ] OpenOCD installed: `openocd --version`
- [ ] GDB installed: `arm-none-eabi-gdb --version` or `gdb-multiarch --version`
- [ ] smartap-jtag binary downloaded and executable
- [ ] OpenOCD config files from the repository

**Completed:**

- [ ] CC3200 module extracted from sealed case
- [ ] JTAG header soldered onto CC3200 board

## Ready?

[:material-arrow-right: Continue to Hardware Setup](hardware-setup.md){ .md-button .md-button--primary }

---

[:material-arrow-left: Previous: Overview](overview.md){ .md-button }
