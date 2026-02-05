#!/usr/bin/env python3
"""Basic example of parsing a GLL file."""

import sys
from pathlib import Path

from gll import GllFile


def main():
    if len(sys.argv) < 2:
        print("Usage: python basic_parse.py <gll_file>")
        sys.exit(1)

    gll_path = Path(sys.argv[1])
    if not gll_path.exists():
        print(f"Error: File not found: {gll_path}")
        sys.exit(1)

    # Parse the GLL file
    print(f"Parsing: {gll_path}")
    gll = GllFile.parse(gll_path)

    # Print metadata
    print("\n=== Metadata ===")
    print(f"Manufacturer: {gll.metadata.manufacturer}")
    print(f"Product: {gll.metadata.product_name}")
    print(f"Description: {gll.metadata.description}")

    # Print system info
    print("\n=== System Info ===")
    print(f"Label: {gll.gen_system.label}")
    print(f"Type: {gll.gen_system.system_type.name}")
    print(f"Company: {gll.gen_system.company}")

    # Print header info
    print("\n=== Header ===")
    print(f"Format: {gll.header.format_id}")
    print(f"Version: {gll.header.format_version}.{gll.header.sub_version}")

    # Print database summary
    db = gll.database
    print("\n=== Database ===")
    print(f"Box Types: {len(db.box_types)}")
    print(f"Source Definitions: {len(db.source_definitions)}")
    print(f"Frames: {len(db.frames)}")
    print(f"Filter Groups: {len(db.filter_groups)}")
    print(f"Data Files: {len(db.data_files)}")
    print(f"Include Files: {len(db.include_files)}")

    # List box types
    if db.box_types:
        print("\n=== Box Types ===")
        for bt in db.box_types[:5]:  # Limit to first 5
            print(f"  - {bt.label} (weight: {bt.weight}kg)")

    # List sources
    if db.source_definitions:
        print("\n=== Source Definitions ===")
        for sd in db.source_definitions[:5]:  # Limit to first 5
            print(f"  - {sd.label or sd.key}")
            print(f"      Sensitivity: {sd.sensitivity} dB (1W/1m)")
            print(f"      Impedance: {sd.impedance} ohms")

    # List resources
    print("\n=== Resources ===")
    print(f"Total: {len(gll.resources)}")
    for res in gll.resources[:5]:  # Limit to first 5
        print(f"  - {res.name or f'Resource {res.index}'} ({res.type}, {res.size} bytes)")


if __name__ == "__main__":
    main()
