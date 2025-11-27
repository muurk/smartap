# How It Works

This section explains the mechanics behind the smartap-jtag tool. If you're wondering "how does this actually work?" or considering contributing to the project, start here.

## The Problem

The Smartap device validates TLS certificates against a CA certificate stored in its flash memory. The original CA was issued by Comodo (now Sectigo), and the server certificates have long since expired. Even if the servers were still online, the device would reject the connection.

We can't simply flash new firmware—there's no source code, and the device likely validates firmware signatures. We can't modify the hardcoded server address in meaningful ways without breaking things.

But we *can* replace the CA certificate. And that changes everything.

## The Insight

The device's firmware is built on Texas Instruments' SimpleLink SDK. This SDK includes a complete filesystem API for reading and writing files to flash storage. The CA certificate is just a file: `/cert/129.der`.

Here's the key insight: **the device already has all the code it needs to replace its own certificate**. We just need a way to call those functions.

That's where JTAG comes in.

## The Approach

JTAG is a hardware debugging interface. With a Raspberry Pi connected to the device's JTAG header, we can:

- Halt the processor
- Read and write memory
- Set register values
- Resume execution at any address

That last point is critical. We can point the processor at any function in memory and say "run this." If we know the addresses of the SimpleLink filesystem functions, we can call them directly.

```
┌─────────────────────────────────────────────────────────────────┐
│                     The Certificate Injection Flow               │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│   Raspberry Pi          JTAG            Smartap Device           │
│   ┌──────────┐         Cable           ┌──────────────┐         │
│   │          │ ───────────────────────▶│              │         │
│   │ OpenOCD  │                         │   CC3200     │         │
│   │   +      │  "Delete /cert/129.der" │   Firmware   │         │
│   │  GDB     │ ───────────────────────▶│              │         │
│   │          │                         │  SimpleLink  │         │
│   │          │  "Write new cert data"  │     SDK      │         │
│   │          │ ───────────────────────▶│              │         │
│   └──────────┘                         └──────────────┘         │
│                                                                   │
│   We use GDB to call the device's own filesystem functions       │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

## The Challenge: Firmware Versions

There's a catch. The SimpleLink functions aren't at the same memory addresses in every firmware version. If we call the wrong address, we could corrupt memory or brick the device.

We solve this with **signature-based detection**. Each firmware version has a unique fingerprint based on the actual bytes at known function locations. Before we do anything dangerous, we verify we're looking at a firmware version we recognise.

This is similar to how antivirus software identifies malware—by matching known byte patterns rather than relying on filenames or metadata.

## What You'll Learn

This section covers:

| Page | What It Explains |
|------|------------------|
| [Firmware Detection](firmware-detection.md) | How we identify firmware versions using signatures |
| [Certificate Injection](certificate-injection.md) | How we call SimpleLink functions via GDB |
| [Adding New Firmware](adding-firmware.md) | How to contribute support for unrecognised devices |

## Why This Matters

Understanding these mechanics isn't just academic. If you encounter an unrecognised firmware version, you'll need to find the function addresses yourself. If something goes wrong during injection, understanding the process helps you diagnose the issue.

And if you want to contribute to this project—whether by adding firmware support, improving the tools, or extending the server—this is the foundation everything else builds on.

---

[:material-arrow-right: Next: Firmware Detection](firmware-detection.md){ .md-button .md-button--primary }
