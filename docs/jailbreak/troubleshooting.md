# Jailbreak Troubleshooting

Solutions to common issues during the jailbreak process.

## OpenOCD Issues

### "JTAG scan chain interrogation failed"

**Symptoms:** OpenOCD can't detect the device

**Solutions:**

1. **Check wiring** - Verify all 5 connections match the pin mapping
2. **Check device power** - Smartap must be powered on
3. **Check cable quality** - Try shorter or better quality jumper wires
4. **Reduce speed** - Add to sysfsgpio-smartap.cfg:
   ```
   adapter speed 50
   ```
5. **Check GPIO access** - Ensure OpenOCD can access GPIO:
   ```bash
   sudo usermod -aG gpio $USER
   # Then log out and back in
   ```

### "Permission denied" errors

**Symptoms:** OpenOCD fails to access GPIO pins

**Solutions:**

```bash
# Option 1: Add user to gpio group (preferred)
sudo usermod -aG gpio $USER
newgrp gpio  # Apply without logout

# Option 2: Run OpenOCD as root (not recommended)
sudo openocd -f ...
```

### OpenOCD loses connection

**Symptoms:** OpenOCD was working, then stops responding

**Causes:**
- Device watchdog timer reset the device
- Power interruption
- JTAG cable came loose

**Solutions:**

1. Power cycle the Smartap device
2. Restart OpenOCD immediately after
3. Run commands quickly before watchdog kicks in

### "Error: cc32xx.cpu: target not halted"

**Symptoms:** GDB operations fail with "target not halted"

**Solutions:**

```bash
# Restart OpenOCD
# The cc3200-complete.cfg should halt the device automatically
# If not, manually halt via telnet:
telnet localhost 4444
> halt
> exit
```

---

## GDB Issues

### "Connection refused" to OpenOCD

**Symptoms:** smartap-jtag can't connect to OpenOCD

**Solutions:**

1. **Check OpenOCD is running:**
   ```bash
   ps aux | grep openocd
   ```

2. **Check binding:** For remote connections, OpenOCD needs:
   ```bash
   openocd ... -c "bindto 0.0.0.0"
   ```

3. **Check firewall:**
   ```bash
   # On Raspberry Pi
   sudo ufw allow 3333/tcp
   ```

4. **Check correct host:**
   ```bash
   smartap-jtag verify-setup --openocd-host <pi-ip-address>
   ```

### "arm-none-eabi-gdb: command not found"

**Symptoms:** GDB binary isn't found

**Solutions:**

```bash
# Option 1: Install ARM toolchain
sudo apt install gcc-arm-none-eabi

# Option 2: Use gdb-multiarch
sudo apt install gdb-multiarch
smartap-jtag --gdb-path gdb-multiarch verify-setup
```

### GDB timeout errors

**Symptoms:** Operations fail with "timeout" or "no response"

**Solutions:**

1. **Increase timeout:**
   ```bash
   smartap-jtag --timeout 10m inject-certs
   ```

2. **Check device hasn't reset** - Restart OpenOCD and try again

3. **Run with verbose:**
   ```bash
   smartap-jtag --verbose inject-certs
   ```

---

## Firmware Detection Issues

### "Firmware unknown" / Low confidence

**Symptoms:** detect-firmware shows <100% confidence

**This is expected for unrecognized firmware versions.**

**Solutions:**

1. **Submit a memory dump** - See [Unrecognized Firmware](unrecognized-firmware.md)
2. **Check for newer smartap-jtag release** - Your version may have been added
3. **Analyze the firmware yourself** - See the [Firmware Analysis Guide](../contributing/firmware-analysis.md) if you're comfortable with reverse engineering

### Detection succeeds but inject-certs fails

**Symptoms:** Firmware detected at 100% but injection fails

**Solutions:**

1. **Fresh device boot** - Power cycle and try immediately
2. **Check OpenOCD still connected** - May need to restart
3. **Run with verbose:**
   ```bash
   smartap-jtag --verbose inject-certs
   ```

---

## Certificate Injection Issues

### "Failed to delete old certificate"

**Symptoms:** Injection fails at "Deleting old certificate" step

**This might be OK** - If the certificate doesn't exist yet, deletion will fail but injection can continue.

**If injection still fails:**

1. Verify firmware detection showed 100% confidence
2. Power cycle device and try again
3. Check verbose output for specific error

### "Failed to create new file"

**Symptoms:** Injection fails at "Creating new file" step

**Possible causes:**

- File system corruption
- Wrong function addresses (firmware mismatch)
- Device reset during operation

**Solutions:**

1. Verify firmware detection was 100% confident
2. Try reading a file first to test filesystem:
   ```bash
   smartap-jtag read-file --remote-file /cert/129.der --output test.der
   ```
3. Power cycle and retry

### "Bytes written mismatch"

**Symptoms:** Injection reports wrong number of bytes written

**Solutions:**

1. Certificate may have written partially
2. Power cycle and retry injection
3. The certificate file may be corrupted - re-download smartap-jtag

### Device unresponsive after injection

**Symptoms:** Device LEDs off or stuck after injection

**Don't panic!** The injection process resumes the device, but:

1. **Wait 30 seconds** - Device may be rebooting
2. **Power cycle** - Turn off and on
3. **Check certificate** - Use read-file to verify:
   ```bash
   smartap-jtag read-file --remote-file /cert/129.der --output check.der
   ls -la check.der  # Should be ~1500 bytes
   ```

---

## Device Issues

### Device resets during operation

**Symptoms:** Operations start but device resets mid-way

**Cause:** CC3200 watchdog timer

**Solutions:**

1. **Work quickly** - Start OpenOCD right after device power-on
2. **Fresh boot** - Power cycle before important operations
3. **Use the right config** - cc3200-complete.cfg has watchdog handling

### Device won't boot after modification

**Symptoms:** Device doesn't start normally after injection

**Solutions:**

1. **Wait** - Give it 30-60 seconds
2. **Power cycle** - Full off/on
3. **Re-inject** - The certificate can be overwritten:
   ```bash
   smartap-jtag inject-certs
   ```
4. **Read the certificate** - Verify what was written:
   ```bash
   smartap-jtag read-file --remote-file /cert/129.der --output check.der
   file check.der  # Should show DER certificate
   ```

### LED behavior is wrong

**Symptoms:** LEDs don't match expected states

**LED states to know:**

- **Yellow flashing** = Pairing mode
- **Normal operation** = Device working
- **No LEDs** = No power or crashed

After injection, device should return to normal operation.

---

## Network Issues

### Can't connect to Pi from laptop

**Symptoms:** smartap-jtag can't reach OpenOCD on remote Pi

**Solutions:**

1. **Check Pi IP:** `hostname -I` on the Pi
2. **Check port open:** `nc -zv <pi-ip> 3333`
3. **Check OpenOCD binding:** Must have `-c "bindto 0.0.0.0"`
4. **Check firewall:** Both on Pi and your laptop

### Memory dump incomplete

**Symptoms:** Dump file is smaller than 256KB

**Solutions:**

1. Device may have reset during dump
2. Power cycle and retry immediately
3. Increase timeout:
   ```bash
   smartap-jtag --timeout 10m dump-memory --output firmware.bin
   ```

---

## Still Stuck?

If these solutions don't help:

1. **Check GitHub Issues** - Someone may have had the same problem:
   [github.com/muurk/smartap/issues](https://github.com/muurk/smartap/issues)

2. **Run with verbose mode** and include the output:
   ```bash
   smartap-jtag --verbose <command> 2>&1 | tee debug.log
   ```

3. **Open a new issue** with:
   - What you tried
   - Complete command output
   - Device model (if known)
   - Raspberry Pi model and OS version

---

[:material-arrow-left: Previous: Unrecognized Firmware](unrecognized-firmware.md){ .md-button }
[:material-home: Back to Jailbreak Overview](overview.md){ .md-button .md-button--primary }
