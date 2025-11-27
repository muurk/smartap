# Connecting to Your Smartap Device

Before you can configure your Smartap, you need to connect your computer to it. This page explains how.

!!! tip "Quick Version"
    1. Hold the power button on your Smartap for 5-10 seconds until the LED flashes
    2. On your computer, connect to the WiFi network called `evalve`
    3. Run the smartap-cfg tool

---

## Understanding the Connection

Your Smartap device creates its own WiFi network when in setup mode. This is how you communicate with it directly - no internet required.

**Think of it like this:** Your Smartap becomes a tiny WiFi router. You connect your laptop to it, just like connecting to any WiFi network.

---

## Step 1: Put Your Device in Setup Mode

You need to tell your Smartap to start broadcasting its WiFi network.

**To enter setup mode:**

1. Find the **power button** on your Smartap control panel
2. **Press and hold** for 5-10 seconds
3. Watch for the **LED to flash yellow**
4. Release the button

!!! info "Where is the LED?"
    The LED is on the front of the Smartap control panel. It may be hidden or subtle - look for a small light near the buttons.

!!! warning "LED Not Flashing?"
    - Try holding the button longer (up to 15 seconds)
    - Make sure the device has power (check your circuit breaker)
    - Try pressing and releasing a few times, then holding again
    - If the LED shows a steady colour but won't flash, the device may be stuck - try turning off power at the isolator for 30 seconds, then try again

---

## Step 2: Connect Your Computer to the Device

Now connect your computer to the device's WiFi network.

=== "Windows"

    1. Click the **WiFi icon** in the bottom-right of your screen (system tray)
    2. Look for a network named **evalve**
    3. Click it and select **Connect**
    4. No password is needed - the network is open
    5. Windows may warn "No internet" - this is normal and expected

=== "macOS"

    1. Click the **WiFi icon** in the top-right menu bar
    2. Look for a network named **evalve**
    3. Click to connect
    4. No password is needed
    5. macOS may show "No Internet Connection" - this is normal

=== "Linux"

    1. Click your **network manager icon**
    2. Look for a network named **evalve**
    3. Click to connect
    4. No password is needed

!!! info "Can't Find 'evalve' Network?"
    - Confirm the LED is flashing yellow (setup mode is active)
    - Move your computer closer to the Smartap device
    - Try refreshing the WiFi list
    - Make sure your computer supports 2.4GHz WiFi (the device doesn't use 5GHz)

---

## Step 3: Verify the Connection

Once connected, your computer should show:

- **Network:** evalve
- **Status:** Connected (no internet)

**No internet connection is expected.** You're connected directly to your Smartap device, not to the internet.

---

## What's Next?

Now that you're connected, you can configure your device.

[:material-arrow-right: Continue to Configuration Guide](configuring.md){ .md-button .md-button--primary }

The configuration guide shows you exactly how to use the wizard to:

- Configure which buttons control which outlets
- Change your device's WiFi settings (to connect it to your home network)
- Test your configuration

---

## Already Connected to Home WiFi?

If your Smartap is already connected to your home WiFi network (perhaps from before the company shut down), you may not need to do any of this.

**Try this first:**

1. Connect your computer to your **home WiFi** (not `evalve`)
2. Run the smartap-cfg wizard
3. If it finds your device automatically, you're good to go!

If the wizard can't find your device on your home network, you'll need to put it in setup mode and connect via `evalve` as described above.

---

[:material-arrow-left: Previous: Download](download.md){ .md-button }
[:material-arrow-right: Next: Configuring Outlets](configuring.md){ .md-button .md-button--primary }
