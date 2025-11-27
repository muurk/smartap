# =============================================================================
# Replace Certificate on CC3200 Filesystem
# =============================================================================
#
# Part of the Smartap Project
# https://github.com/muurk/smartap
# Documentation: https://muurk.github.io/smartap/
#
# REFERENCE ONLY: This script is provided for educational purposes to show
# how certificates can be replaced on the CC3200 via JTAG/GDB. For actual use,
# the smartap-jtag utility provides a safer and more reliable interface with
# proper error handling and validation:
#
#   ./bin/smartap-jtag inject-certs
#
# =============================================================================
#
# Known working SimpleLink functions:
# sl_FsOpen  = 0x20015c64
# sl_FsWrite = 0x20014bf8
# sl_FsClose = 0x2001555c
# sl_FsDel   = 0x20016ea8

target remote 172.16.80.207:3333

set pagination off
set $work_buffer = 0x20030000
set $file_handle_ptr = 0x20031000
set $filename_ptr = 0x20031004
set $token_ptr = 0x20031020
set $cert_size =     1501

printf "\n=== Certificate Replacement ===\n\n"

# Halt device
printf "[1/6] Halting device...\n"
monitor halt
shell sleep 0.5

# Setup filename: /cert/129.der
printf "[2/6] Setting up filename...\n"
set *(char*)($filename_ptr+0)  = 0x2f
set *(char*)($filename_ptr+1)  = 0x63
set *(char*)($filename_ptr+2)  = 0x65
set *(char*)($filename_ptr+3)  = 0x72
set *(char*)($filename_ptr+4)  = 0x74
set *(char*)($filename_ptr+5)  = 0x2f
set *(char*)($filename_ptr+6)  = 0x31
set *(char*)($filename_ptr+7)  = 0x32
set *(char*)($filename_ptr+8)  = 0x39
set *(char*)($filename_ptr+9)  = 0x2e
set *(char*)($filename_ptr+10) = 0x64
set *(char*)($filename_ptr+11) = 0x65
set *(char*)($filename_ptr+12) = 0x72
set *(char*)($filename_ptr+13) = 0x00

# Load certificate data
printf "[3/6] Loading certificate to memory...\n"
restore custom-certs/ca-root-cert.der binary ($work_buffer)
printf "Loaded %d bytes\n", $cert_size

# Delete old certificate (if exists)
printf "\n[4/6] Deleting old certificate...\n"
set $r0 = $filename_ptr
set $r1 = 0
set $pc = 0x20016ea8
set $lr = 0x20000001
set $sp = 0x20031d00
finish
printf "Delete result: %d (0=success, -11=not found)\n", $r0

# Create new file
printf "\n[5/6] Creating new certificate file...\n"
set $mode = 0x53006
printf "Using mode: 0x%x\n", $mode
set $r0 = $filename_ptr
set $r1 = $mode
set $r2 = $token_ptr
set $r3 = $file_handle_ptr
set $pc = 0x20015c64
set $lr = 0x20000001
set $sp = 0x20031d00
finish

if ($r0 != 0)
    printf "ERROR: Failed to create file (result: %d)\n", $r0
    quit
end

set $handle = *(int*)$file_handle_ptr
printf "File created! Handle: 0x%08x\n", $handle

# Write certificate data
printf "\n[6/6] Writing certificate data...\n"
set $r0 = $handle
set $r1 = 0
set $r2 = $work_buffer
set $r3 = $cert_size
set $pc = 0x20014bf8
set $lr = 0x20000001
finish

if ($r0 != $cert_size)
    printf "ERROR: Write failed (wrote %d, expected %d)\n", $r0, $cert_size
end

printf "Wrote %d bytes\n", $r0

# Close file
set $r0 = $handle
set $r1 = 0
set $r2 = 0
set $r3 = 0
set $pc = 0x2001555c
set $lr = 0x20000001
finish
printf "Close result: %d\n", $r0

printf "\n=== Certificate Replacement Complete ===\n"
printf "Resuming device...\n"
continue &

detach
quit
