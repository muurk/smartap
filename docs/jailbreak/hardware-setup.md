# Hardware Setup

This guide covers connecting your Raspberry Pi to the CC3200 module's JTAG interface.

!!! warning "Soldering Required"
    These instructions require soldering a header onto the unpopulated JTAG pads on the CC3200 board. If you haven't already done this, see [Prerequisites - Physical Access](prerequisites.md#physical-access-to-the-cc3200-module) first.

## Overview

You will:

1. Solder a header to the JTAG pads on the CC3200 board (if not already done)
2. Connect jumper wires between the header and your Raspberry Pi's GPIO pins
3. Reconnect the CC3200 to the Smartap control unit
4. Power on the Smartap and start OpenOCD

## JTAG Pin Mapping

Connect these pins between your Raspberry Pi and the Smartap device:

| Smartap Pin | Function | Raspberry Pi GPIO | RPi Physical Pin |
|-------------|----------|-------------------|------------------|
| Pin 2 | TDO (data out) | GPIO 26 | Pin 37 |
| Pin 3 | TCK (clock) | GPIO 13 | Pin 33 |
| Pin 4 | TMS (mode select) | GPIO 19 | Pin 35 |
| Pin 5 | TDI (data in) | GPIO 6 | Pin 31 |
| Pin 6 | GND (ground) | GND | Pin 39 |

!!! warning "Double-Check Connections"
    Incorrect wiring can damage your Raspberry Pi or Smartap device. Verify each connection before powering on.

## Locating the JTAG Header

The CC3200 module inside the Smartap device has an unpopulated JTAG header - you'll see pads/holes without pins soldered in.

**Reference photo**: See [this image from the Home Assistant community](https://community.home-assistant.io/t/smartap-shower-control-getting-started-with-reverse-engineering-a-smart-home-device/358251/120) showing the board layout. Note that the photo shows the board with the RF shield removed - your board will have the shield in place, but you do NOT need to remove it to access the JTAG header.

The JTAG pads are located near the CC3200 chip. They are NOT labeled on the board, so use the reference photo to identify them.

!!! warning "Soldering Required"
    You will need to solder a 2.54mm pitch header to the JTAG pads. Soldering a header (rather than wires directly) is recommended because:

    - Headers allow you to disconnect and reassemble the device easily
    - Direct wires make it difficult to close up the device afterwards
    - Test clips or "balancing" wires on pads is not practical for the extended operations required

    If you're not comfortable soldering, consider asking a local electronics hobbyist or makerspace for help.

## Raspberry Pi GPIO Diagram

```
Raspberry Pi GPIO Header (looking at the board, USB ports at bottom)

                    3V3  (1)  (2)  5V
                  GPIO2  (3)  (4)  5V
                  GPIO3  (5)  (6)  GND
                  GPIO4  (7)  (8)  GPIO14
                    GND  (9)  (10) GPIO15
                 GPIO17 (11)  (12) GPIO18
                 GPIO27 (13)  (14) GND
                 GPIO22 (15)  (16) GPIO23
                    3V3 (17)  (18) GPIO24
                 GPIO10 (19)  (20) GND
                  GPIO9 (21)  (22) GPIO25
                 GPIO11 (23)  (24) GPIO8
                    GND (25)  (26) GPIO7
                  GPIO0 (27)  (28) GPIO1
                  GPIO5 (29)  (30) GND
    TDI ───────► GPIO6 (31)  (32) GPIO12
    TCK ───────► GPIO13 (33) (34) GND
    TMS ───────► GPIO19 (35) (36) GPIO16
    TDO ───────► GPIO26 (37) (38) GPIO20
    GND ───────► GND   (39)  (40) GPIO21
```

## Reconnecting for JTAG Access

Before you can use OpenOCD, the CC3200 must be powered on - which means reconnecting it to the Smartap control unit.

!!! warning "Safety"
    Ensure the shower remains OFF at the isolator until everything is reconnected.

### Step 1: Connect Jumper Wires to the CC3200

With the CC3200 board still accessible (outside its case or with the case open):

1. Connect the 5 jumper wires to the header you soldered onto the JTAG pads
2. Route the wires so they won't be pinched when you close the case
3. Leave enough slack to reach the Raspberry Pi

### Step 2: Reconnect the CC3200 to the Control Unit

1. Screw the WIFI cable terminal back into the Smartap control unit
2. The CC3200 is now electrically connected but the case may still be open - this is OK for testing
3. Position the Raspberry Pi within reach of the jumper wires

### Step 3: Connect to the Raspberry Pi

1. Connect the other end of each jumper wire to the Raspberry Pi GPIO pins as shown in the pin mapping table above
2. Double-check each connection against the table

### Step 4: Power On

1. Turn the Smartap back on at the isolator
2. The CC3200 module will now be powered
3. You're ready to start OpenOCD

!!! tip "Testing Setup"
    For initial testing, you can leave the CC3200 case open. Once you've confirmed everything works, you can close and reseal the case (with the jumper wires routed out through a small gap or hole).

## Starting OpenOCD

Once wired and powered, start OpenOCD on the Raspberry Pi:

```bash
cd /path/to/smartap/hardware/openocd

# Start OpenOCD with the Smartap configuration
openocd -f sysfsgpio-smartap.cfg \
        -c "transport select jtag" \
        -c "bindto 0.0.0.0" \
        -f cc3200-complete.cfg
```

**Successful connection looks like:**

```
Open On-Chip Debugger 0.11.0
Info : SysfsGPIO JTAG/SWD bitbang driver
Info : Note: sysfsgpio "adapter speed" is not setable, rely on defaults
Info : Listening on port 4444 for telnet connections
Info : Listening on port 3333 for gdb connections
Info : JTAG tap: cc32xx.jrc tap/device found: 0x0b97c02f
Info : cc32xx.cpu: hardware has 6 breakpoints, 4 watchpoints
```

!!! success "Key indicators"
    - "JTAG tap: cc32xx.jrc tap/device found" = Device detected
    - "Listening on port 3333 for gdb connections" = Ready for smartap-jtag

**Connection failure looks like:**

```
Error: JTAG scan chain interrogation failed: all zeroes
Error: Check JTAG interface, timing, target power, etc.
```

## Troubleshooting Connection Issues

### "JTAG scan chain interrogation failed"

- **Check wiring:** Verify all 5 connections
- **Check device power:** Smartap must be powered on
- **Reduce speed:** Edit `sysfsgpio-smartap.cfg` and add `adapter speed 50`
- **Check GPIO permissions:** Run OpenOCD with `sudo` or add user to gpio group

### "Permission denied" on GPIO

```bash
# Add your user to the gpio group
sudo usermod -aG gpio $USER

# Log out and back in, or:
newgrp gpio
```

### Device keeps resetting

The CC3200 has an aggressive watchdog timer. The `cc3200-complete.cfg` file includes handling for this, but if the device resets during operations:

1. Power cycle the Smartap device
2. Restart OpenOCD immediately after power-on
3. Run smartap-jtag commands quickly

### OpenOCD shows "Error: timed out"

- The device may have reset
- Power cycle and try again
- Ensure you're using both config files (sysfsgpio AND cc3200-complete)

## Verify the Setup

With OpenOCD running, from your machine (or the Pi):

```bash
smartap-jtag verify-setup --openocd-host <pi-ip-address>
```

Expected output:

```
✓ Setup verification complete
  GDB:     arm-none-eabi-gdb (found)
  OpenOCD: 192.168.1.100:3333 (connected)
  Status:  Ready for JTAG operations
```

## Next Steps

Once setup is verified, you're ready to detect your firmware and inject the certificate.

[:material-arrow-right: Continue to Using smartap-jtag](using-smartap-jtag.md){ .md-button .md-button--primary }

---

[:material-arrow-left: Previous: Prerequisites](prerequisites.md){ .md-button }
