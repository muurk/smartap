# =============================================================================
# Read Certificate File from CC3200 Filesystem
# =============================================================================
#
# Part of the Smartap Project
# https://github.com/muurk/smartap
# Documentation: https://muurk.github.io/smartap/
#
# REFERENCE ONLY: This script is provided for educational purposes to show
# how the CC3200 filesystem can be accessed via JTAG/GDB. For actual use,
# the smartap-jtag utility provides a safer and more reliable interface:
#
#   ./bin/smartap-jtag read-cert
#
# =============================================================================
#
# Known working SimpleLink functions:
# sl_FsOpen  = 0x20015c64
# sl_FsRead  = 0x20014b54
# sl_FsClose = 0x2001555c

target remote 172.16.80.207:3333

# Memory locations
set $work_buffer = 0x20030000
set $file_handle_ptr = 0x20031000
set $filename_ptr = 0x20031004
set $token_ptr = 0x20031020

printf "\n=== Reading /cert/129.der ===\n\n"

# Halt device
printf "[1/5] Halting device...\n"
monitor halt
shell sleep 0.5

# Write filename to memory
printf "[2/5] Setting up filename in memory...\n"
set *(char*)($filename_ptr+0)  = 0x2F
set *(char*)($filename_ptr+1)  = 0x63
set *(char*)($filename_ptr+2)  = 0x65
set *(char*)($filename_ptr+3)  = 0x72
set *(char*)($filename_ptr+4)  = 0x74
set *(char*)($filename_ptr+5)  = 0x2F
set *(char*)($filename_ptr+6)  = 0x31
set *(char*)($filename_ptr+7)  = 0x32
set *(char*)($filename_ptr+8)  = 0x39
set *(char*)($filename_ptr+9)  = 0x2E
set *(char*)($filename_ptr+10) = 0x64
set *(char*)($filename_ptr+11) = 0x65
set *(char*)($filename_ptr+12) = 0x72
set *(char*)($filename_ptr+13) = 0x00

printf "Filename: "
x/s $filename_ptr

# Open file
printf "\n[3/5] Opening file...\n"
set $r0 = $filename_ptr
set $r1 = 0
set $r2 = $token_ptr
set $r3 = $file_handle_ptr
set $pc = 0x20015c64
set $lr = 0x20000001
set $sp = 0x20031d00

finish

set $open_result = $r0
set $handle = *(int*)$file_handle_ptr

printf "Open result: %d\n", $open_result
printf "File handle: 0x%08x\n", $handle

# Read file (2048 bytes - known size)
printf "\n[4/5] Reading file contents...\n"
set $r0 = $handle
set $r1 = 0
set $r2 = $work_buffer
set $r3 = 2048
set $pc = 0x20014b54
set $lr = 0x20000001

finish

set $read_result = $r0
printf "Read result: %d bytes\n", $read_result

printf "\nFirst 128 bytes of certificate:\n"
x/128bx $work_buffer

# Close file
printf "\n[5/5] Closing file...\n"
set $r0 = $handle
set $r1 = 0
set $r2 = 0
set $r3 = 0
set $pc = 0x2001555c
set $lr = 0x20000001

finish

printf "Close result: %d\n", $r0

printf "\n=== Summary ===\n"
printf "Open: %d (should be 0 for success)\n", $open_result
printf "Read: %d bytes (should be 2048)\n", $read_result
printf "\nResuming device...\n"
continue &

detach
quit
