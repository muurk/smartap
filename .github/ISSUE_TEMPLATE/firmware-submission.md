---
name: Firmware Submission
about: Submit a memory dump for unrecognized firmware support
title: '[FIRMWARE] Device memory dump for analysis'
labels: firmware
assignees: ''
---

## ⚠️ IMPORTANT: WiFi Credentials Warning

**Your memory dump WILL contain your WiFi SSID and password in plaintext.**

Before submitting:
1. Connect your Smartap device to a **temporary WiFi hotspot** (e.g., phone hotspot)
2. Use credentials **you don't mind sharing publicly**
3. Perform the memory dump while connected to this temporary network
4. You can change your device back to your real WiFi after the dump

**Do NOT submit a dump that contains your real home WiFi credentials!**

---

## Device Information

- **Device Model** (if known):
- **Where purchased**:
- **Approximate purchase date**:

## Firmware Detection Output

Paste the complete output from `smartap-jtag detect-firmware`:

```
(paste output here)
```

## Memory Dump

Attach your memory dump file (firmware.bin) to this issue.

**How to create the dump:**
```bash
# Make sure you're connected to a temporary WiFi first!
smartap-jtag dump-memory --output firmware.bin
```

⚠️ **Before attaching**, confirm:
- [ ] I have connected my device to a **temporary/disposable WiFi network**
- [ ] The credentials in this dump are **not my real home WiFi**
- [ ] I understand this dump will be publicly visible on GitHub

## Additional Notes

(Any other relevant information about your device or setup)

---

### What happens next?

1. A maintainer will analyze your memory dump
2. They will identify function addresses for certificate operations
3. Support will be added to `smartap-jtag` for your firmware version
4. You'll be notified when you can run `smartap-jtag inject-certs`

### Need help creating the dump?

- [Unrecognized Firmware Guide](https://muurk.github.io/smartap/jailbreak/unrecognized-firmware/) - Step-by-step instructions
- [Hardware Setup Guide](https://muurk.github.io/smartap/jailbreak/hardware-setup/) - JTAG connection setup
