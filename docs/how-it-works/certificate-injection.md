# Certificate Injection

This page explains how we actually replace the CA certificate on the device. It's a fascinating technique: we use the device's own firmware functions against itself.

## The Technique

The Smartap device runs firmware built on Texas Instruments' SimpleLink SDK. This SDK provides filesystem functions for reading and writing files to flash storage. The CA certificate is stored as `/cert/129.der`.

We can't modify the firmware. But we don't need to. Using JTAG and GDB, we can:

1. Halt the processor mid-execution
2. Set up function arguments in registers and memory
3. Point the program counter at a SimpleLink function
4. Let it run and capture the result

We're essentially making function calls from outside the device, using the device's own code.

## ARM Calling Convention Primer

To call a function, we need to understand how ARM passes arguments. The Cortex-M4 uses the ARM EABI calling convention:

| Register | Purpose |
|----------|---------|
| R0 | First argument / return value |
| R1 | Second argument |
| R2 | Third argument |
| R3 | Fourth argument |
| SP | Stack pointer |
| LR | Link register (return address) |
| PC | Program counter (current instruction) |

To call `sl_FsOpen(filename, mode, token, handle)`:

```gdb
set $r0 = <pointer to filename string>
set $r1 = <access mode flags>
set $r2 = <pointer to token buffer>
set $r3 = <pointer to file handle buffer>
set $lr = <return address>
set $pc = <address of sl_FsOpen>
continue
```

When the function returns, the result is in R0.

## The Injection Sequence

Certificate injection happens in nine steps:

```
┌─────────────────────────────────────────────────────────────────┐
│ [1/9] Halt the device                                           │
│       Stop execution so we can manipulate state                 │
├─────────────────────────────────────────────────────────────────┤
│ [2/9] Set up memory                                             │
│       Write filename string and certificate data to RAM         │
├─────────────────────────────────────────────────────────────────┤
│ [3/9] Load certificate to work buffer                           │
│       Copy the new CA cert into device memory                   │
├─────────────────────────────────────────────────────────────────┤
│ [4/9] Delete old certificate                                    │
│       Call sl_FsDel("/cert/129.der")                            │
├─────────────────────────────────────────────────────────────────┤
│ [4.5/9] Wait for flash erase                                    │
│         Flash operations are asynchronous - must wait           │
├─────────────────────────────────────────────────────────────────┤
│ [5/9] Create new certificate file                               │
│       Call sl_FsOpen with create flags                          │
├─────────────────────────────────────────────────────────────────┤
│ [6/9] Write certificate data                                    │
│       Call sl_FsWrite with certificate bytes                    │
├─────────────────────────────────────────────────────────────────┤
│ [7/9] Close the file                                            │
│       Call sl_FsClose to finalise                               │
├─────────────────────────────────────────────────────────────────┤
│ [8/9] Verify injection                                          │
│       Check bytes written matches certificate size              │
├─────────────────────────────────────────────────────────────────┤
│ [9/9] Resume device                                             │
│       Let the device continue normal operation                  │
└─────────────────────────────────────────────────────────────────┘
```

## Memory Layout During Injection

We need somewhere safe to store temporary data. The CC3200 has 256KB of RAM starting at `0x20000000`. We use a region near the end that's unlikely to conflict with running code:

```
CC3200 RAM Layout During Injection
──────────────────────────────────

0x20000000 ┌─────────────────────────┐
           │                         │
           │   Firmware Code/Data    │
           │   (don't touch!)        │
           │                         │
0x20030000 ├─────────────────────────┤ ◄── work_buffer
           │   Certificate Data      │     (up to 4KB)
           │   (temporarily stored)  │
0x20031000 ├─────────────────────────┤ ◄── file_handle_ptr
           │   File Handle (4 bytes) │
0x20031004 ├─────────────────────────┤ ◄── filename_ptr
           │   Filename String       │     "/cert/129.der\0"
           │   (up to 28 bytes)      │
0x20031020 ├─────────────────────────┤ ◄── token_ptr
           │   Token Buffer          │
0x20031d00 ├─────────────────────────┤ ◄── stack_base
           │   Stack Space           │     (for function calls)
           │                         │
0x20040000 └─────────────────────────┘
```

These addresses come from the firmware database. They're chosen to be safe for each firmware version.

## Step by Step: What the GDB Script Does

### Step 2: Writing the Filename

The filename `/cert/129.der` must be in device memory before we can pass it to `sl_FsDel` or `sl_FsOpen`. We write it byte by byte:

```gdb
# Write "/cert/129.der" to filename_ptr (0x20031004)
set *((unsigned char*)($filename_ptr + 0)) = 47    # '/'
set *((unsigned char*)($filename_ptr + 1)) = 99    # 'c'
set *((unsigned char*)($filename_ptr + 2)) = 101   # 'e'
set *((unsigned char*)($filename_ptr + 3)) = 114   # 'r'
set *((unsigned char*)($filename_ptr + 4)) = 116   # 't'
set *((unsigned char*)($filename_ptr + 5)) = 47    # '/'
set *((unsigned char*)($filename_ptr + 6)) = 49    # '1'
set *((unsigned char*)($filename_ptr + 7)) = 50    # '2'
set *((unsigned char*)($filename_ptr + 8)) = 57    # '9'
set *((unsigned char*)($filename_ptr + 9)) = 46    # '.'
set *((unsigned char*)($filename_ptr + 10)) = 100  # 'd'
set *((unsigned char*)($filename_ptr + 11)) = 101  # 'e'
set *((unsigned char*)($filename_ptr + 12)) = 114  # 'r'
set *((unsigned char*)($filename_ptr + 13)) = 0    # NULL terminator
```

### Step 3: Loading the Certificate

GDB's `restore` command efficiently copies binary data into memory:

```gdb
restore /tmp/smartap-cert-XXXXX.bin binary $work_buffer
```

This is much faster than writing byte-by-byte for a 1KB+ certificate.

### Step 4: Deleting the Old Certificate

Now we call `sl_FsDel`:

```gdb
# sl_FsDel(filename, token)
set $r0 = $filename_ptr      # "/cert/129.der"
set $r1 = 0                  # token (not used)
set $pc = 0x20016ea8         # sl_FsDel address (firmware-specific)
set $lr = 0x20000001         # return address (thumb mode)
set $sp = $stack_base        # safe stack location
finish                       # run until function returns
set $delete_result = $r0     # capture result
```

The `finish` command runs until the current function returns. The result (success/error code) ends up in R0.

### Step 4.5: The 5-Second Wait

!!! warning "Flash Erase is Asynchronous"
    The CC3200's flash filesystem doesn't complete erase operations immediately. If we try to create the new file too quickly, `sl_FsOpen` may fail because the old file hasn't fully been erased.

```gdb
shell sleep 5
```

This wait is crucial. Without it, injection fails intermittently.

### Step 5: Creating the New File

The `sl_FsOpen` function has complex mode flags:

```gdb
# Calculate mode flags for file creation
set $gran_size = 256
set $size_blocks = (($cert_size + 255) / 256)  # Round up
set $access_mode = 0x1                          # Write access
set $flags = 0x3                                # Create + Commit
set $gran_idx = 5                               # Granularity index

# Pack into mode parameter
set $mode = (($access_mode << 12) | ($gran_idx << 8) | $size_blocks | ($flags << 16))

# sl_FsOpen(filename, mode, token_ptr, file_handle_ptr)
set $r0 = $filename_ptr
set $r1 = $mode
set $r2 = $token_ptr
set $r3 = $file_handle_ptr
set $pc = 0x20015c64         # sl_FsOpen address
set $lr = 0x20000001
set $sp = $stack_base
finish
set $file_handle = *(int*)$file_handle_ptr
```

The mode calculation handles TI's granular file allocation system.

### Step 6: Writing the Certificate

```gdb
# sl_FsWrite(handle, offset, data, length)
set $r0 = $file_handle
set $r1 = 0                  # offset (start of file)
set $r2 = $work_buffer       # certificate data
set $r3 = $cert_size         # number of bytes
set $pc = 0x20014bf8         # sl_FsWrite address
set $lr = 0x20000001
set $sp = $stack_base
finish
set $bytes_written = $r0
```

### Step 7: Closing the File

```gdb
# sl_FsClose(handle, cert_name, signature, sig_len)
set $r0 = $file_handle
set $r1 = 0                  # cert_name (NULL)
set $r2 = 0                  # signature (NULL)
set $r3 = 0                  # sig_len (0)
set $pc = 0x2001555c         # sl_FsClose address
set $lr = 0x20000001
set $sp = $stack_base
finish
set $close_result = $r0
```

### Step 8: Verification

We verify that:

1. `$bytes_written == $cert_size` — all bytes were written
2. `$close_result == 0` — file closed successfully

If either check fails, the injection failed and the device may be in an inconsistent state.

## What Can Go Wrong

### Wrong Firmware Detection

If signatures matched incorrectly (shouldn't happen with 7/7 requirement), function addresses will be wrong. Best case: immediate crash. Worst case: silent memory corruption.

### Flash Wear

Flash memory has limited write cycles. Normal use is fine, but repeated injection during development could eventually wear out the certificate storage sector.

### Interrupted Injection

If power is lost or JTAG disconnects mid-injection, the certificate file may be corrupted or missing. The device will fail to establish TLS connections until injection is completed successfully.

### Stack Corruption

If the stack pointer is set incorrectly, function calls may corrupt memory. The memory layout addresses are chosen carefully to avoid this.

## The Result

After successful injection:

```
✓ Certificate injection complete

  Firmware:        Smartap 0x355
  Certificate:     Embedded Root CA
  Bytes Written:   1234
  Status:          Verified

The device will now trust certificates signed by the injected CA.
```

The device's trust anchor has been replaced. It will now accept TLS connections from servers presenting certificates signed by your CA, rather than the original (expired) Comodo certificate chain.

---

[:material-arrow-left: Previous: Firmware Detection](firmware-detection.md){ .md-button }
[:material-arrow-right: Next: Adding New Firmware](adding-firmware.md){ .md-button .md-button--primary }
