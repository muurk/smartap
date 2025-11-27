# Hardware Access Guide

Detailed guide for JTAG access to the Smartap device.

## Safety First

!!! danger "Electrical Safety"
    The device operates at mains voltage. **Always disconnect power** before opening the device or making any connections.

!!! warning "Water Environment"
    This device is installed in a wet environment. Ensure all seals are properly reinstated after modification to prevent water ingress.

## Opening the Device

### Tools Needed

- Small Phillips screwdriver
- Plastic spudger or opening tool
- Anti-static wrist strap (recommended)

### Procedure

1. **Disconnect power** - Turn off the circuit breaker for your bathroom
2. **Remove control panel** - Carefully detach the front control panel
3. **Access main unit** - Remove screws securing the main enclosure
4. **Remove shielding** - Carefully remove RF shielding if present
5. **Locate JTAG header** - Find the 6-pin unpopulated header

!!! tip "Take Photos"
    Take photos at each step to help with reassembly. Document the position of all cables and connections.

## JTAG Interface

### Header Location

The CC3200 module has a 6-pin vertical header labeled with pin 1 marked by a dot (●) and "+" symbol.

```
Pin 1  [●] +        ← Pin 1 marker
Pin 2  [ ]
Pin 3  [ ]
Pin 4  [ ]
Pin 5  [ ]
Pin 6  [■] GND      ← Ground
```

### Pin Identification

Confirmed via JTAGenum:

| Pin | Function | Description |
|-----|----------|-------------|
| 1   | NC       | Not connected |
| 2   | TDO      | Test Data Out |
| 3   | TCK      | Test Clock |
| 4   | TMS      | Test Mode Select |
| 5   | TDI      | Test Data In |
| 6   | GND      | Ground |

**IR Length:** 6 bits

### Connection Options

#### Option 1: Direct Wire Connection (Temporary)

**Pros:**
- No soldering required
- Reversible
- Quick setup

**Cons:**
- Unreliable connection
- May disconnect during use
- Requires holding in place

**Method:**
Use female-to-female jumper wires, press firmly onto pads

#### Option 2: Soldered Pin Headers (Permanent)

**Pros:**
- Reliable connection
- Professional result
- Easy reconnection later

**Cons:**
- Requires soldering skills
- Permanent modification
- More time consuming

**Method:**

1. Obtain 2.54mm pitch pin headers
2. Cut to 6-pin length
3. Position headers in holes
4. Solder each pin carefully
5. Trim excess length if needed

!!! tip "Soldering Advice"
    Use a fine-tipped soldering iron (15-30W) and thin solder. Work quickly to avoid heat damage to the board. If inexperienced, practice on scrap electronics first.

#### Option 3: IC Test Clip

**Pros:**
- No soldering needed
- Secure connection
- Reusable

**Cons:**
- Requires suitable clip
- May not fit all board layouts
- Additional cost

## Raspberry Pi Connection

### GPIO Pinout

Connect Smartap JTAG to Raspberry Pi GPIO header:

```
Smartap        Wire Color      RPI GPIO        RPI Pin
Pin 2 (TDO)    [Suggested:     GPIO 26         Pin 37
Pin 3 (TCK)     Red/Blue/      GPIO 13         Pin 33
Pin 4 (TMS)     etc for        GPIO 19         Pin 35
Pin 5 (TDI)     easy           GPIO 6          Pin 31
Pin 6 (GND)     reference]     GND             Pin 39
```

### Physical Setup

1. **Power off everything** - Both Smartap and Raspberry Pi
2. **Connect wires** - Attach jumpers to Raspberry Pi first
3. **Connect to Smartap** - Carefully attach other end to Smartap
4. **Verify connections** - Double-check every pin
5. **Power on Raspberry Pi** - Boot the Pi first
6. **Power on Smartap** - Then power on Smartap

!!! warning "Connection Order"
    Always connect wires with power OFF. Never hot-plug JTAG connections as this can damage both devices.

### Connection Verification

Use a multimeter to verify:

1. **Continuity** - Each wire properly connected
2. **No shorts** - No accidental connections between pins
3. **Ground** - GND connection solid

## Testing JTAG Connection

### Install Software

```bash
# Update package list
sudo apt-get update

# Install OpenOCD
sudo apt-get install openocd

# Install GDB for ARM
sudo apt-get install gdb-multiarch
# Or ARM-specific:
sudo apt-get install gcc-arm-none-eabi gdb-arm-none-eabi
```

### JTAGenum Test (Optional)

If you want to verify pin assignments:

```bash
# Clone JTAGenum
git clone https://github.com/cyphunk/JTAGenum.git
cd JTAGenum

# Edit header to set pins
nano JTAGenum.sh
# Set: pins=(26 19 13 6 5)
# Set: pinnames=(26 19 13 6 5)

# Run scan
source JTAGenum.sh
scan
```

Expected output:
```
FOUND!  ntrst:5 tck:13 tms:19 tdo:26 tdi:6 IR length: 6
```

### OpenOCD Configuration

Create configuration files (see [Research Background](research-background.md) for complete configs):

**sysfsgpio-smartap.cfg:**
```cfg
adapter driver sysfsgpio
sysfsgpio jtag_nums 13 19 6 26
sysfsgpio trst_num 5
adapter speed 100
reset_config trst_only
```

**cc3200-complete.cfg:**
See full configuration in research background documentation.

### Test Connection

```bash
cd /path/to/configs
openocd -f ./sysfsgpio-smartap.cfg \
        -c "transport select jtag" \
        -c "bindto 0.0.0.0" \
        -f ./cc3200-complete.cfg
```

**Success looks like:**
```
Info : Listening on port 4444 for telnet connections
Info : Listening on port 3333 for gdb connections
Info : JTAG tap: cc32xx.jrc tap/device found
Info : cc32xx.cpu: hardware has 6 breakpoints, 4 watchpoints
```

**If it fails:**
- Check all connections
- Verify power to both devices
- Try slower speed: `adapter speed 50`
- Check GPIO permissions: Add user to `gpio` group

## Memory Access

Once OpenOCD is running, connect via telnet:

```bash
telnet localhost 4444
```

Try basic commands:

```
> halt
> reg
> mdw 0x20004000 16
> resume
```

## Troubleshooting

### "Error: JTAG scan chain interrogation failed"

**Causes:**
- Wires not connected properly
- Wrong GPIO pins configured
- Device not powered on
- JTAG disabled in device fuses

**Solutions:**
- Recheck all connections with multimeter
- Verify pin numbers in config file
- Ensure Smartap has power
- Try different JTAG speed

### "Cannot access memory at address 0x..."

**Causes:**
- Device not halted
- Watchdog timer reset device
- Invalid memory address

**Solutions:**
- Issue `halt` command first
- Check watchdog handling in config
- Verify address is valid for CC3200

### Unstable Connection

**Causes:**
- Poor physical connection
- Wire too long
- Electrical interference
- Incorrect speed setting

**Solutions:**
- Solder header pins for reliable connection
- Use shorter wires (<20cm)
- Keep wires away from power supplies
- Reduce JTAG speed: `adapter speed 50`

### Device Keeps Resetting

**Causes:**
- Watchdog timer not being serviced
- Power issue
- JTAG configuration problem

**Solutions:**
- Ensure watchdog handling in OpenOCD config
- Check power supply stability
- Verify reset configuration

## Advanced: Manual Pin Finding

If you don't have documentation and need to find JTAG pins manually:

1. **Visual inspection** - Look for unpopulated headers near MCU
2. **Continuity test** - Trace pins to MCU if visible
3. **JTAGenum scan** - Systematically test pin combinations
4. **Try standard TI layouts** - CC3200 has common JTAG arrangements

## Closing the Device

After JTAG work:

1. **Verify functionality** - Test device works before closing
2. **Photograph setup** - Document for future access
3. **Label wires** - If leaving wires attached, label clearly
4. **Replace shielding** - Reinstall RF shielding carefully
5. **Check seals** - Ensure water seals intact
6. **Secure enclosure** - Replace all screws properly
7. **Test operation** - Full functional test before closing walls

!!! danger "Water Sealing"
    This is critical! The device will be in a humid bathroom environment. Ensure all seals are properly reinstated to prevent water damage.

## Resources

- [CC3200 Datasheet](https://www.ti.com/product/CC3200)
- [OpenOCD Documentation](http://openocd.org/documentation/)
- [JTAGenum Project](https://github.com/cyphunk/JTAGenum)
- [Home Assistant Community Discussion](https://community.home-assistant.io/t/smartap-shower-control-getting-started-with-reverse-engineering-a-smart-home-device/358251/206) - Includes photos of JTAG header

---

[:material-arrow-left: Previous: Architecture](architecture.md){ .md-button }
[:material-arrow-right: Next: Certificate Details](certificate-details.md){ .md-button .md-button--primary }
