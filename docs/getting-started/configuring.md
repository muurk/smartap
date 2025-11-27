# Configuring Your Smartap

This guide walks you through using the smartap-cfg wizard to configure your device. You'll see exactly what appears on screen at each step.

!!! tip "Before You Start"
    Make sure you've [downloaded the tool](download.md). You don't need to connect to WiFi first - the wizard handles everything.

---

## Launching the Wizard

Open your terminal and run the tool:

```bash
./smartap-cfg
```

The wizard starts automatically. You don't need any extra commands.

---

## Step 1: Finding Your Device

When the wizard starts, it immediately searches for your Smartap device:

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ SMARTAP CONFIGURATION WIZARD v1.0.0                    github.com/muurk/smartap│
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│                        ⣾ SEARCHING FOR DEVICES                               │
│                                                                              │
│                  Scanning your network for Smartap devices...                │
│                                                                              │
│                  ████████████████░░░░░░░░░░░░░░  45%                         │
│                                                                              │
│                              Elapsed: 4s                                     │
│                                                                              │
├──────────────────────────────────────────────────────────────────────────────┤
│ m: manual IP • q: quit                                                       │
└──────────────────────────────────────────────────────────────────────────────┘
```

**What's happening:** The tool is looking for your device on the network. This takes about 10 seconds.

---

### If Your Device Is Found

When the scan completes successfully, you'll see your device listed:

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ SMARTAP CONFIGURATION WIZARD v1.0.0                    github.com/muurk/smartap│
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ╭────────────────────────────────────────────────────────────────────────╮  │
│  │ → eValve12345                                                          │  │
│  │                                                                        │  │
│  │   Serial:   12345                                                      │  │
│  │   IP:       192.168.4.1:80                                             │  │
│  │   Firmware: 1.2.3                                                      │  │
│  │   Status:   Ready                                                      │  │
│  ╰────────────────────────────────────────────────────────────────────────╯  │
│                                                                              │
├──────────────────────────────────────────────────────────────────────────────┤
│ ↑/↓: navigate • enter: configure • r: rescan • m: manual IP • q: quit        │
└──────────────────────────────────────────────────────────────────────────────┘
```

**Press Enter** to select your device and continue.

---

### If No Device Is Found

If the scan doesn't find anything, you'll see:

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ SMARTAP CONFIGURATION WIZARD v1.0.0                    github.com/muurk/smartap│
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│    ⚠ No devices found on your network                                        │
│                                                                              │
│    Troubleshooting:                                                          │
│      • Ensure device is powered on                                           │
│      • Check that device is in pairing mode (LED flashing)                   │
│      • Verify you're connected to device's WiFi hotspot                      │
│      • Try increasing scan time (use 'r' to rescan)                          │
│                                                                              │
├──────────────────────────────────────────────────────────────────────────────┤
│ r: rescan • m: manual IP • q: quit                                           │
└──────────────────────────────────────────────────────────────────────────────┘
```

**Try these solutions:**

1. **Check the LED** - Is your Smartap's LED flashing? If not, hold the power button for 5-10 seconds to enter pairing mode.

2. **Check your WiFi** - Look at your computer's WiFi settings. You should be connected to a network called `evalve`. If you're connected to your home WiFi instead, switch to `evalve`.

3. **Press `r`** to scan again after making changes.

4. **Press `m`** to enter the device's IP address manually (try `192.168.4.1`).

---

## Step 2: The Configuration Dashboard

Once you select a device, the dashboard appears showing all settings:

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ SMARTAP CONFIGURATION WIZARD v1.0.0                    github.com/muurk/smartap│
├──────────────────────────────────────────────────────────────────────────────┤
│ Device: eValve12345 • 192.168.4.1:80 • FW: 1.2.3                             │
│ ────────────────────────────────────────────────────────────────────────     │
│                                                                              │
│ OUTLETS                                                                      │
│ → First Button Press   [1] Outlet 1                                      ▼  │
│   Second Button Press  [2] Outlet 2                                      ▼  │
│   Third Button Press   [4] Outlet 3                                      ▼  │
│   Third Knob Mode      [ ] Disabled                                      ▼  │
│                                                                              │
│                            [Apply Outlets]                                   │
│                                                                              │
│ WIFI                                                                         │
│   Network              MyHomeWiFi                                        ▼  │
│                                                                              │
│                             [Apply WiFi]                                     │
│                                                                              │
│ SERVER                                                                       │
│   Hostname             evalve.smartap-tech.com                               │
│   Port                 443                                                   │
│                                                                              │
│                            [Apply Server]                                    │
│                                                                              │
├──────────────────────────────────────────────────────────────────────────────┤
│ ↑/↓: navigate • tab: next section • enter: edit • q: quit                    │
└──────────────────────────────────────────────────────────────────────────────┘
```

The dashboard is divided into three sections:

| Section | What It Does |
|---------|--------------|
| **OUTLETS** | Controls which water outlets turn on when you press each button |
| **WIFI** | Changes which WiFi network your device connects to |
| **SERVER** | Advanced setting - where the device connects for smart features |

**Navigation:**

- **↑/↓ arrows** - Move between fields
- **Tab** - Jump to the next section
- **Enter** - Edit the selected field
- **q** - Quit the wizard

---

## Configuring Outlets (Button Assignments)

This is the most common task - changing what happens when you press the buttons on your shower.

### Understanding Your Shower

Your Smartap device has **three outlets** (water outputs):

- **Outlet 1** - Usually the main shower head
- **Outlet 2** - Usually a second fixture (hand shower, rain head, etc.)
- **Outlet 3** - Usually a third fixture (bath spout, body jets, etc.)

Each button press can turn on **one outlet**, **multiple outlets**, or **nothing**.

### Changing a Button Assignment

1. Use **↑/↓** to highlight the button you want to change (e.g., "First Button Press")
2. Press **Enter**

The field expands to show all options:

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ OUTLETS                                                                      │
│ → First Button Press   [1] Outlet 1                                      ▼  │
│         ( ) [0] None                                                         │
│         (•) [1] Outlet 1                                              ←      │
│         ( ) [2] Outlet 2                                                     │
│         ( ) [3] Outlets 1+2                                                  │
│         ( ) [4] Outlet 3                                                     │
│         ( ) [5] Outlets 1+3                                                  │
│         ( ) [6] Outlets 2+3                                                  │
│         ( ) [7] Outlets 1+2+3                                                │
│         ↑/↓ select • Enter confirm • Esc cancel                              │
│                                                                              │
│   Second Button Press  [2] Outlet 2                                      ▼  │
│   Third Button Press   [4] Outlet 3                                      ▼  │
└──────────────────────────────────────────────────────────────────────────────┘
```

3. Use **↑/↓** to highlight your choice
4. Press **Enter** to confirm (or **Esc** to cancel)

### What the Options Mean

| Option | What Turns On |
|--------|---------------|
| `[0] None` | Button does nothing |
| `[1] Outlet 1` | Just outlet 1 |
| `[2] Outlet 2` | Just outlet 2 |
| `[3] Outlets 1+2` | Both outlets 1 and 2 together |
| `[4] Outlet 3` | Just outlet 3 |
| `[5] Outlets 1+3` | Both outlets 1 and 3 together |
| `[6] Outlets 2+3` | Both outlets 2 and 3 together |
| `[7] Outlets 1+2+3` | All three outlets together |

### Example: Setting Up Your Shower

Let's say your shower has:

- **Outlet 1** = Rain shower head
- **Outlet 2** = Hand-held shower
- **Outlet 3** = Bath filler

You want:

- First button = Rain head only
- Second button = Hand shower only
- Third button = Bath filler only

Set each button to match:

1. First Button Press → `[1] Outlet 1`
2. Second Button Press → `[2] Outlet 2`
3. Third Button Press → `[4] Outlet 3`

### Applying Your Changes

After making changes, you'll see the dashboard shows **⚠ Modified** and the Apply button shows changes are pending:

```
│ Device: eValve12345 • 192.168.4.1:80 • FW: 1.2.3                             │
│ ⚠ MODIFIED                                                                   │
│ ────────────────────────────────────────────────────────────────────────     │
│                                                                              │
│ OUTLETS                                                                      │
│   First Button Press   [1] Outlet 1 ⚠                                    ▼  │
│ → Second Button Press  [3] Outlets 1+2 ⚠                                 ▼  │
│   Third Button Press   [4] Outlet 3                                      ▼  │
│   Third Knob Mode      [ ] Disabled                                      ▼  │
│                                                                              │
│                        [Apply Outlets] ⚠ Modified                            │
```

1. Navigate to **[Apply Outlets]**
2. Press **Enter**

The wizard sends your changes to the device:

```
┌────────────────────────────────────────────────────────────────────────────┐
│                                                                            │
│                    ⣾ APPLYING CONFIGURATION                                │
│                                                                            │
│                    ████████████████░░░░░░░░░  65%                          │
│                                                                            │
│                    ✓ Configuration sent to device                          │
│                    ⣾ Verifying changes...                                  │
│                                                                            │
│                    Elapsed: 1.2s                                           │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

When complete, you'll see confirmation:

```
┌────────────────────────────────────────────────────────────────────────────┐
│                                                                            │
│                    ✓ CONFIGURATION APPLIED!                                │
│                                                                            │
│  ✓ Configuration updated successfully!                                     │
│                                                                            │
│  Your changes have been saved to the device                                │
│                                                                            │
│  Verified new configuration:                                               │
│    First Press:  [1] Outlet 1                                              │
│    Second Press: [3] Outlets 1+2                                           │
│    Third Press:  [4] Outlet 3                                              │
│    Third Knob:   Disabled                                                  │
│                                                                            │
│  Configuration verified in 1.5 seconds                                     │
│                                                                            │
│                          [Continue]                                        │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

Press **Enter** to return to the dashboard.

---

## Configuring WiFi

Use this if you need to connect your device to a different WiFi network.

!!! warning "Important"
    After changing WiFi settings, your device will disconnect from its current network. You'll need to reconnect your computer to the same network to continue using the wizard.

### Changing the WiFi Network

1. Navigate to the **Network** field under WIFI
2. Press **Enter**

You'll see available networks and a password field:

```
│ WIFI                                                                         │
│ → Network              MyHomeWiFi                                        ▼  │
│       Select network:                                                        │
│         (•) MyHomeWiFi                                                ←      │
│         ( ) NeighborsWiFi                                                    │
│         ( ) GuestNetwork                                                     │
│                                                                              │
│       Password: ••••••••                                                     │
│         ↑/↓ select • Enter confirm • Esc cancel                              │
│                                                                              │
│                             [Apply WiFi]                                     │
```

3. Use **↑/↓** to select a network
4. Use **↓** to move to the password field
5. Type the WiFi password
6. Press **Enter** to confirm

### Applying WiFi Changes

Navigate to **[Apply WiFi]** and press **Enter**.

You'll see a warning because WiFi changes disconnect the device:

```
┌────────────────────────────────────────────────────────────────────────────┐
│                                                                            │
│                    ⚠ WIFI NETWORK CHANGE WARNING                           │
│                                                                            │
│  After applying this change:                                               │
│  • Device will disconnect from current network                             │
│  • You will lose connection to the device                                  │
│                                                                            │
│  Old Network: evalve                                                       │
│  New Network: MyHomeWiFi                                                   │
│                                                                            │
│  To continue:                                                              │
│  1. Connect to the same network on this computer                           │
│  2. Re-run this application to verify changes                              │
│                                                                            │
│               → [Apply Changes]         [Cancel]                           │
│                                                                            │
│  ←/→: Navigate  •  Enter: Confirm  •  Esc: Back                            │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

Select **[Apply Changes]** and press **Enter** to proceed.

---

## Testing Your Configuration

After making changes:

1. Go to your shower
2. Press each button on the control panel
3. Verify the correct outlets turn on

If something isn't right, run the wizard again and adjust the settings.

---

## Common Configurations

### Standard Three-Outlet Setup

Each button controls one outlet:

- First Button → `[1] Outlet 1`
- Second Button → `[2] Outlet 2`
- Third Button → `[4] Outlet 3`

### Dual Shower Heads on One Button

First button turns on both shower heads:

- First Button → `[3] Outlets 1+2`
- Second Button → `[4] Outlet 3`
- Third Button → `[0] None`

### "All On" Button

First button turns everything on:

- First Button → `[7] Outlets 1+2+3`
- Second Button → `[1] Outlet 1`
- Third Button → `[2] Outlet 2`

---

## Troubleshooting

### Changes Don't Seem to Work

1. Make sure you pressed **[Apply Outlets]** after making changes
2. Check that you saw the "✓ CONFIGURATION APPLIED!" screen
3. Try power-cycling your Smartap device (turn off, wait 10 seconds, turn on)

### Wrong Outlets Activating

Your plumber may have connected fixtures to different outlets than expected. Use trial and error:

1. Set First Button to `[1] Outlet 1` only
2. Press the first button and note which fixture turns on
3. Repeat for outlets 2 and 3
4. Now you know which outlet number controls which fixture

### Can't Find Device After WiFi Change

After changing WiFi:

1. Connect your computer to the **same WiFi network** you configured on the device
2. Wait 30-60 seconds for the device to connect
3. Run the wizard again - it should find your device

---

## What's Next?

Your basic configuration is complete. Your shower should now respond correctly to each button press.

**Optional next steps:**

- [Troubleshooting](troubleshooting.md) - If you're having issues
- [Device Jailbreak Guide](../jailbreak/overview.md) - If you want remote control features

---

[:material-arrow-left: Previous: WiFi Setup](wifi-setup.md){ .md-button }
[:material-help-circle: Troubleshooting](troubleshooting.md){ .md-button }
