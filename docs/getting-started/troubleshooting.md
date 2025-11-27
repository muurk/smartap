# Troubleshooting

Common issues and solutions when configuring your Smartap device.

!!! warning "When Software Can't Help"
    The smartap-cfg tool can only configure **software settings** (WiFi, outlet assignments, server address). It **cannot fix hardware problems** such as:

    - **Low water pressure** - This is a plumbing issue, not a software issue
    - **Water leaks** - Requires professional plumber inspection
    - **Valve not opening/closing** - May be mechanical failure or scale buildup
    - **Temperature issues** - Related to your water heater or mixing valve
    - **Physical button not responding** - May be hardware failure

    If your problem is hardware-related, contact a plumber or the original installer.

---

## Device Won't Enter Setup Mode

**Problem:** LED doesn't flash yellow when holding the power button.

**Solutions:**

1. **Hold longer** - Try holding for up to 15 seconds
2. **Check power** - Verify the device has power and other LEDs work
3. **Try different button** - Make sure you're pressing the main power button
4. **Power cycle** - Turn off the device completely, wait 30 seconds, turn back on, then try again

## Can't Find "evalve" WiFi Network

**Problem:** The evalve network doesn't appear in your WiFi list.

**Solutions:**

1. **Confirm setup mode** - Make sure the device LED is flashing yellow
2. **Move closer** - Get your computer physically closer to the device
3. **Refresh WiFi list** - Close and reopen your WiFi menu
4. **Check 2.4GHz support** - Make sure your computer supports 2.4GHz WiFi (the device uses 2.4GHz only)
5. **Disable VPN** - Temporarily disable any VPN software
6. **Try another device** - If your computer can't see it, try a phone or tablet

## Configuration Tool Can't Find Device

**Problem:** Tool says "No device found" or times out.

### When Connected to "evalve" Network:

1. **Verify connection** - Make sure you're actually connected to evalve network
2. **Check IP address** - Your computer should have an IP like 192.168.99.x
3. **Firewall** - Temporarily disable firewall software
4. **Restart tool** - Close and reopen the configuration tool

### When Connected to Home WiFi:

1. **Wait longer** - Device may still be connecting (wait 2-3 minutes)
2. **Check same network** - Computer and device must be on same WiFi network
3. **Router issues** - Some routers block device discovery:
    - Check router has "AP isolation" or "client isolation" disabled
    - Look for "device discovery" settings
4. **Firewall** - Temporarily disable firewall software
5. **Try direct connection** - Connect to evalve network instead and try again

## Device Won't Connect to WiFi

**Problem:** Device doesn't connect after entering WiFi credentials.

**Solutions:**

1. **Check WiFi name** - Verify SSID is exactly correct (case sensitive!)
2. **Check password** - Re-enter your WiFi password carefully
3. **2.4GHz only** - Device only works with 2.4GHz WiFi networks
    - If you have a dual-band router, make sure 2.4GHz is enabled
    - Some newer routers only use 5GHz
4. **WiFi security** - Device works with WPA/WPA2
    - WPA3-only networks may not work
    - Open networks should work
    - WEP is not recommended
5. **Special characters** - If your WiFi password has special characters, try changing it temporarily
6. **Hidden network** - Device may not work with hidden SSIDs

## Outlet Configuration Not Working

**Problem:** Configuration applied but outlets don't respond correctly.

**Solutions:**

1. **Power cycle** - Turn device off, wait 30 seconds, turn back on
2. **Re-apply configuration** - Run tool again and reapply settings
3. **Check values** - Verify outlet values are correct (1, 2, or 4)
4. **Physical connections** - Check that outlets are physically connected to fixtures
5. **Button mapping** - Try different button assignments to test
6. **Factory reset** - As a last resort, reset device and start over

## Wrong Outlets Activating

**Problem:** Pressing a button activates the wrong fixture.

**Solutions:**

1. **Check outlet numbers** - You may have outlets numbered differently than expected
2. **Try systematic testing:**
    ```
    Set Outlet 1 = 1, Outlet 2 = 0, Outlet 3 = 0
    Test which fixture activates with Button 1

    Set Outlet 1 = 0, Outlet 2 = 1, Outlet 3 = 0
    Test which fixture activates with Button 1

    Set Outlet 1 = 0, Outlet 2 = 0, Outlet 3 = 1
    Test which fixture activates with Button 1
    ```
3. **Physical wiring** - Plumbing may be connected differently than labeled
4. **Document your findings** - Once you identify which outlet controls which fixture, note it down

## macOS Security Warning

**Problem:** macOS won't let me run the configuration tool.

**Solutions:**

1. **Right-click method:**
    - Right-click (or Control-click) on the application
    - Select "Open"
    - Click "Open" in the security dialog

2. **System Settings method:**
    - Try to open the app normally (it will be blocked)
    - Go to System Settings → Privacy & Security
    - Scroll down to see "smartap-cfg was blocked"
    - Click "Open Anyway"

3. **Command line method:**
    ```bash
    xattr -d com.apple.quarantine /path/to/smartap-cfg
    ```

## Windows SmartScreen Warning

**Problem:** Windows Defender blocks the application.

**Solutions:**

1. Click "More info"
2. Click "Run anyway"

This is normal for new applications that aren't code-signed.

## Device Shows Offline After Configuration

**Problem:** Device worked initially but now shows offline.

**Solutions:**

1. **WiFi connection** - Device may have lost WiFi connection
    - Check router is working
    - Check other devices can connect
    - Device may have been assigned a new IP address
2. **Power cycle** - Turn device off and on
3. **Router restart** - Restart your WiFi router
4. **Reconfigure WiFi** - Use setup mode to reconfigure WiFi settings

## Advanced: Device Won't Reset

**Problem:** Nothing works and you want to factory reset.

**Solutions:**

!!! danger "Factory Reset"
    This will erase all settings including WiFi configuration.

The Smartap device reset procedure varies by model. Try:

1. **Long press** - Hold power button for 20-30 seconds
2. **Power cycle with button** - Turn off power, hold button, turn on power while still holding
3. **Check manual** - Refer to your original installation guide

If reset doesn't work, see [Community](../about/community.md) for help.

## Still Having Issues?

### Check These Resources:

- [FAQ](../about/faq.md) - More common questions
- [Community Forum](../about/community.md) - Ask for help
- [GitHub Issues](https://github.com/muurk/smartap/issues) - Report bugs

### When Asking for Help:

Please provide:

1. **Device info** - Serial number, software version (shown in tool)
2. **Your setup** - What you're trying to do
3. **What happened** - Error messages, unexpected behavior
4. **What you tried** - Solutions you've already attempted
5. **Your environment** - Operating system, network setup

### Emergency: Device Completely Unresponsive

If the device won't respond at all:

1. **Check power** - Verify device has power
2. **Check circuit breaker** - Make sure bathroom circuit is on
3. **Check device fuse** - Some models have internal fuses
4. **Contact installer** - May need professional inspection
5. **Community help** - Ask in [community forum](../about/community.md)

---

[:material-arrow-left: Back to Getting Started](overview.md){ .md-button }
[:material-help-circle: Visit FAQ](../about/faq.md){ .md-button }
