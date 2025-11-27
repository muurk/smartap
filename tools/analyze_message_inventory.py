#!/usr/bin/env python3
"""
Analyze all captured messages to create a comprehensive inventory.
This will help us understand what message types we're actually seeing.
"""

import json
import glob
from collections import defaultdict, Counter

def main():
    # Collect all messages
    all_messages = []

    files = sorted(glob.glob('smartap-server/analysis/messages/capture-*.jsonl'))
    print(f"Analyzing {len(files)} capture files...\n")

    for filepath in files:
        try:
            with open(filepath, 'r') as f:
                for line in f:
                    if not line.strip():
                        continue
                    msg = json.loads(line)
                    all_messages.append(msg)
        except Exception as e:
            print(f"Warning: Failed to read {filepath}: {e}")

    print(f"Total messages collected: {len(all_messages)}\n")

    # Group by payload length
    by_length = defaultdict(list)
    for msg in all_messages:
        length = msg['payload_length']
        by_length[length].append(msg)

    print("=" * 80)
    print("MESSAGE LENGTH DISTRIBUTION")
    print("=" * 80)
    for length in sorted(by_length.keys()):
        count = len(by_length[length])
        print(f"  {length:3d} bytes: {count:4d} messages")

    print("\n" + "=" * 80)
    print("ANALYZING EACH MESSAGE LENGTH")
    print("=" * 80)

    for length in sorted(by_length.keys()):
        messages = by_length[length]
        print(f"\n--- {length} BYTE MESSAGES ({len(messages)} total) ---")

        # Get unique hex patterns
        hex_patterns = Counter([msg['payload_hex'] for msg in messages])
        unique_count = len(hex_patterns)

        print(f"Unique patterns: {unique_count}")

        # Show first byte (message type)
        first_bytes = Counter()
        for msg in messages:
            hex_data = msg['payload_hex']
            if len(hex_data) >= 2:
                first_byte = hex_data[0:2]
                first_bytes[first_byte] += 1

        print(f"First byte distribution:")
        for byte_val, count in first_bytes.most_common():
            print(f"  0x{byte_val}: {count} messages")

        # Show most common patterns (up to 5)
        if unique_count <= 10:
            print(f"\nAll unique hex patterns:")
            for idx, (hex_pattern, count) in enumerate(hex_patterns.most_common(), 1):
                print(f"  Pattern {idx} ({count} occurrences):")
                print(f"    {hex_pattern}")
                # Try to identify message type
                if len(hex_pattern) >= 2:
                    first_byte = int(hex_pattern[0:2], 16)
                    print(f"    First byte: 0x{first_byte:02x}")
        else:
            print(f"\nTop 5 most common patterns:")
            for idx, (hex_pattern, count) in enumerate(hex_patterns.most_common(5), 1):
                print(f"  Pattern {idx} ({count} occurrences):")
                print(f"    {hex_pattern[:80]}{'...' if len(hex_pattern) > 80 else ''}")

    print("\n" + "=" * 80)
    print("SPECIAL INTEREST: 77-BYTE MESSAGES")
    print("=" * 80)

    if 77 in by_length:
        msgs_77 = by_length[77]
        print(f"\nFound {len(msgs_77)} messages of 77 bytes")

        # Check if they're all identical
        hex_patterns_77 = set([msg['payload_hex'] for msg in msgs_77])
        print(f"Unique patterns: {len(hex_patterns_77)}")

        if len(hex_patterns_77) == 1:
            print("\n✓ All 77-byte messages are IDENTICAL")
            print("\nHex:")
            print(f"  {list(hex_patterns_77)[0]}")

            # Check message numbers
            msg_nums = [msg['message_num'] for msg in msgs_77]
            print(f"\nMessage numbers: {msg_nums}")
            if all(n == 1 for n in msg_nums):
                print("✓ All are message #1 (first message of connection)")
        else:
            print(f"\n✗ Found {len(hex_patterns_77)} different 77-byte patterns")
            for idx, pattern in enumerate(hex_patterns_77, 1):
                print(f"\nPattern {idx}:")
                print(f"  {pattern}")

    print("\n" + "=" * 80)
    print("ENVELOPE MESSAGES (TYPE 0x01)")
    print("=" * 80)

    # Find messages starting with 7e03 (sync + version) followed by envelope type 0x01
    envelope_messages = []
    for msg in all_messages:
        hex_data = msg['payload_hex']
        # Check if starts with 7e03 and has 0x01 at payload offset 0 (byte 8)
        if len(hex_data) >= 20:  # At least 10 bytes
            if hex_data.startswith('7e03'):
                # Payload starts at byte 8 (offset 16 in hex string)
                payload_start = 16
                if len(hex_data) > payload_start:
                    msg_type = hex_data[payload_start:payload_start+2]
                    if msg_type == '01':
                        envelope_messages.append(msg)

    print(f"\nFound {len(envelope_messages)} envelope messages (type 0x01)")

    if envelope_messages:
        # Group by length
        env_by_length = defaultdict(list)
        for msg in envelope_messages:
            env_by_length[msg['payload_length']].append(msg)

        print("\nEnvelope message lengths:")
        for length in sorted(env_by_length.keys()):
            print(f"  {length} bytes: {len(env_by_length[length])} messages")

        # Show some examples
        print("\nExample envelope messages:")
        for length in sorted(env_by_length.keys())[:3]:
            print(f"\n  {length}-byte envelope (first occurrence):")
            msg = env_by_length[length][0]
            hex_data = msg['payload_hex']
            print(f"    Full: {hex_data}")
            # Show bytes 8-20 (payload offset 0-12)
            if len(hex_data) >= 40:
                print(f"    Payload bytes 0-12: {hex_data[16:40]}")
                print(f"    Payload byte 10: 0x{hex_data[36:38]} (suspected nested message type)")

if __name__ == '__main__':
    main()
