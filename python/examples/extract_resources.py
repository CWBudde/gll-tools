#!/usr/bin/env python3
"""Example of extracting resources from a GLL file."""

import sys
from pathlib import Path

from gll import GllFile


def main():
    if len(sys.argv) < 2:
        print("Usage: python extract_resources.py <gll_file> [output_dir]")
        sys.exit(1)

    gll_path = Path(sys.argv[1])
    output_dir = Path(sys.argv[2]) if len(sys.argv) > 2 else Path("extracted")

    if not gll_path.exists():
        print(f"Error: File not found: {gll_path}")
        sys.exit(1)

    output_dir.mkdir(parents=True, exist_ok=True)

    # Parse the GLL file
    print(f"Parsing: {gll_path}")
    gll = GllFile.parse(gll_path)

    # Extract embedded resources (images, etc.)
    print(f"\n=== Extracting Resources to {output_dir} ===")
    for res in gll.resources:
        name = res.name or f"resource_{res.index}.{res.type.lower()}"
        output_path = output_dir / name

        print(f"  Extracting: {name} ({res.type}, {res.size} bytes)")
        try:
            data = gll.extract_resource(res)
            output_path.write_bytes(data)
            print(f"    -> {output_path}")
        except Exception as e:
            print(f"    Error: {e}")

    # Extract data files (geometry, etc.)
    print("\n=== Extracting Data Files ===")
    for df in gll.database.data_files:
        output_path = output_dir / df.filename

        print(f"  Extracting: {df.filename} ({df.size} bytes)")
        try:
            data = gll.extract_data_file(df)
            output_path.write_bytes(data)
            print(f"    -> {output_path}")
        except Exception as e:
            print(f"    Error: {e}")

    # Extract include files (PDFs, documentation)
    print("\n=== Extracting Include Files ===")
    for inc in gll.database.include_files:
        output_path = output_dir / inc.filename

        print(f"  Extracting: {inc.filename} ({inc.size} bytes)")
        try:
            data = gll.extract_include_file(inc)
            output_path.write_bytes(data)
            print(f"    -> {output_path}")
        except Exception as e:
            print(f"    Error: {e}")

    print(f"\nDone! Files extracted to: {output_dir}")


if __name__ == "__main__":
    main()
