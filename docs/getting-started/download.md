# Download the Configuration Tool

Get the smartap-cfg tool for your computer.

---

## Choose Your Operating System

=== "Windows"

    ### Windows

    [:material-download: Windows](https://github.com/muurk/smartap/releases/latest/download/smartap-cfg-windows-amd64.exe){ .md-button .md-button--primary }

    **After downloading:**

    1. Find the downloaded file (usually in your Downloads folder)
    2. Double-click `smartap-cfg-windows-amd64.exe` to run it

    !!! warning "Windows Security Warning"
        Windows may show a blue "Windows protected your PC" warning. This is normal for new software.

        **To continue:**

        1. Click **"More info"**
        2. Click **"Run anyway"**

        This only happens the first time you run the tool.

=== "macOS"

    ### macOS

    **Which Mac do you have?**

    - [:material-download: Apple Silicon (M1/M2/M3)](https://github.com/muurk/smartap/releases/latest/download/smartap-cfg-darwin-arm64){ .md-button .md-button--primary }
    - [:material-download: Intel](https://github.com/muurk/smartap/releases/latest/download/smartap-cfg-darwin-amd64){ .md-button }

    !!! tip "Not sure which Mac you have?"
        Click the Apple menu → "About This Mac". If it says "Apple M1" or similar, you have Apple Silicon. If it says "Intel", you have an Intel Mac.

    **After downloading:**

    1. Open **Finder** and go to your **Downloads** folder
    2. Find the downloaded file (`smartap-cfg-darwin-arm64` or `smartap-cfg-darwin-amd64`)
    3. **Right-click** (or Control-click) on the file
    4. Select **"Open"** from the menu
    5. Click **"Open"** in the security dialog

    !!! warning "macOS Security Warning"
        macOS blocks apps downloaded from the internet by default. Right-clicking and selecting "Open" tells macOS you trust this file.

        If you see "cannot be opened because it is from an unidentified developer":

        1. Go to **System Settings** → **Privacy & Security**
        2. Scroll down to see a message about the blocked app
        3. Click **"Open Anyway"**

    **Opening Terminal (first time only):**

    If you've never used Terminal before:

    1. Press **Command + Space** to open Spotlight
    2. Type **Terminal** and press Enter
    3. A window with a command prompt appears

    **Running the tool:**

    In Terminal, type these commands (pressing Enter after each):

    ```
    cd ~/Downloads
    chmod +x smartap-cfg-darwin-arm64
    ./smartap-cfg-darwin-arm64
    ```

    (Replace `arm64` with `amd64` if you have an Intel Mac)

=== "Linux"

    ### Linux

    **Download the version for your system:**

    - [:material-download: Linux x86_64](https://github.com/muurk/smartap/releases/latest/download/smartap-cfg-linux-amd64){ .md-button .md-button--primary }
    - [:material-download: Raspberry Pi 4/5 (ARM64)](https://github.com/muurk/smartap/releases/latest/download/smartap-cfg-linux-arm64){ .md-button }
    - [:material-download: Raspberry Pi 32bit ARMv7](https://github.com/muurk/smartap/releases/latest/download/smartap-cfg-linux-armv7){ .md-button }

    **After downloading:**

    ```bash
    cd ~/Downloads
    chmod +x smartap-cfg-linux-amd64
    ./smartap-cfg-linux-amd64
    ```

---

## Verify It Works

When you run the tool successfully, you'll see the wizard interface start up:

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ SMARTAP CONFIGURATION WIZARD v1.0.0                    github.com/muurk/smartap│
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│                        ⣾ SEARCHING FOR DEVICES                               │
│                                                                              │
```

If you see this, the tool is working. It's looking for your Smartap device on the network.

!!! info "Device Not Found?"
    If the wizard can't find your device, that's expected at this stage. Continue to the next page to learn how to connect to your device.

---

## What's Next?

[:material-arrow-right: Continue to Connection Guide](wifi-setup.md){ .md-button .md-button--primary }

The next page explains how to connect your computer to your Smartap device so the wizard can find it.

---

[:material-arrow-left: Previous: Overview](overview.md){ .md-button }
[:material-arrow-right: Next: Connection Guide](wifi-setup.md){ .md-button .md-button--primary }
