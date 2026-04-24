#!/usr/bin/env python3
"""Example of computing array response from a GLL file."""

import sys
from pathlib import Path

from gll import (
    AirProperties,
    ArrayCalculator,
    ArrayConfig,
    GllFile,
    Vector3D,
)


def main():
    if len(sys.argv) < 2:
        print("Usage: python array_response.py <gll_file>")
        sys.exit(1)

    gll_path = Path(sys.argv[1])

    if not gll_path.exists():
        print(f"Error: File not found: {gll_path}")
        sys.exit(1)

    # Parse the GLL file
    print(f"Parsing: {gll_path}")
    gll = GllFile.parse(gll_path)

    # Create calculator
    calc = ArrayCalculator(gll)

    # Show available box types
    box_types = calc.available_box_types
    print(f"\n=== Available Box Types ({len(box_types)}) ===")
    for bt in box_types:
        print(f"  - {bt}")

    if not box_types:
        print("No box types available for array configuration")
        sys.exit(0)

    # Configure array with first available box type
    box_type = box_types[0]
    print(f"\n=== Configuring Array with '{box_type}' ===")

    config = (
        ArrayConfig()
        .add_element(box_type, splay=0.0)   # First box, no splay
        .add_element(box_type, splay=0.5)   # Second box, 0.5° splay
        .add_element(box_type, splay=1.0)   # Third box, 1.0° splay
        .add_element(box_type, splay=1.5)   # Fourth box, 1.5° splay
    )

    print(f"Array configured with {len(config.elements)} elements")

    # Compute response at different positions
    print("\n=== Computing Responses ===")

    receivers = [
        Vector3D(0, 10, 0),    # 10m on-axis
        Vector3D(0, 20, 0),    # 20m on-axis
        Vector3D(0, 20, -5),   # 20m away, 5m below
        Vector3D(5, 20, 0),    # 20m away, 5m to the side
    ]

    air = AirProperties(temperature=20.0, humidity=0.5, pressure=101.325)

    for i, recv in enumerate(receivers):
        print(f"\nReceiver {i+1}: ({recv.x}, {recv.y}, {recv.z}) meters")

        response = calc.compute_response(
            config,
            receiver=recv,
            air=air,
            air_attenuation=False,
        )

        if response.is_valid:
            tf = response.transfer_function
            print(f"  Bands: {len(tf.level)}")
            print(f"  Delay: {tf.delay*1000:.2f} ms")

            # Find approximate level at 1kHz (mid-band)
            mid_idx = len(tf.level) // 2
            print(f"  Level at mid-band: {tf.level[mid_idx]:.1f} dB")
        else:
            print(f"  Error: {response.error}")

    print("\nDone!")


if __name__ == "__main__":
    main()
