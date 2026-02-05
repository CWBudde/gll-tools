# gll - Python Bindings for EASE GLL Files

Python bindings for parsing EASE GLL (Generic Loudspeaker Library) files.

GLL is a proprietary binary format used by AFMG's EASE acoustic simulation software to describe loudspeaker systems, including directivity measurements, frequency responses, and metadata.

## Installation

```bash
pip install gll
```

Or install from source:

```bash
# Clone the repository
git clone https://github.com/cwbudde/gll-tools
cd gll-tools

# Build the shared library
CGO_ENABLED=1 go build -buildmode=c-shared -o python/gll/_libgll.so ./cmd/gllpy

# Install the Python package
pip install -e ./python
```

## Quick Start

### Parse a GLL File

```python
from gll import GllFile

# Parse a GLL file
gll = GllFile.parse("speaker.gll")

# Access metadata
print(f"Manufacturer: {gll.metadata.manufacturer}")
print(f"Product: {gll.metadata.product_name}")
print(f"Description: {gll.metadata.description}")

# Access system info
print(f"System Type: {gll.gen_system.system_type.name}")
```

### Access Database Components

```python
from gll import GllFile

gll = GllFile.parse("speaker.gll")

# List box types (speaker cabinets)
for box in gll.database.box_types:
    print(f"Box: {box.label}")
    print(f"  Weight: {box.weight} kg")
    print(f"  Sources: {len(box.source_placements)}")

# List source definitions (acoustic sources)
for source in gll.database.source_definitions:
    print(f"Source: {source.label}")
    print(f"  Sensitivity: {source.sensitivity} dB (1W/1m)")
    print(f"  Impedance: {source.impedance} ohms")
```

### Extract Embedded Resources

```python
from gll import GllFile

gll = GllFile.parse("speaker.gll")

# List resources (images, PDFs, etc.)
for resource in gll.resources:
    print(f"Resource: {resource.name} ({resource.type}, {resource.size} bytes)")

# Extract a resource
if gll.resources:
    data = gll.extract_resource(gll.resources[0])
    with open(gll.resources[0].name, 'wb') as f:
        f.write(data)

# Extract include files (documentation)
for inc in gll.database.include_files:
    data = gll.extract_include_file(inc)
    with open(inc.filename, 'wb') as f:
        f.write(data)
```

### Compute Array Response

```python
from gll import GllFile, ArrayCalculator, ArrayConfig, Vector3D

gll = GllFile.parse("line_array.gll")

# Create calculator
calc = ArrayCalculator(gll)

# See available box types
print(f"Available box types: {calc.available_box_types}")

# Configure array
config = ArrayConfig()
config.add_element("K2", splay=0.5)  # First box, 0.5° splay
config.add_element("K2", splay=1.0)  # Second box, 1.0° splay
config.add_element("K2", splay=1.5)  # Third box, 1.5° splay

# Compute response at receiver position
receiver = Vector3D(0, 20, -5)  # 20m away, 5m below
response = calc.compute_response(config, receiver=receiver)

if response.is_valid:
    tf = response.transfer_function
    print(f"Level at 1kHz: {tf.level[len(tf.level)//2]:.1f} dB")
```

### Get Balloon Directivity Data

```python
from gll import GllFile

gll = GllFile.parse("speaker.gll")

# Get balloon data at 1kHz for first source
balloon = gll.get_balloon_at_frequency(source_index=0, frequency_hz=1000)

print(f"Frequency: {balloon['frequency']} Hz")
print(f"Angular resolution: {balloon['meridian_step']}° x {balloon['parallel_step']}°")

# balloon['data'] is a 2D array [azimuth][elevation] of SPL values
data = balloon['data']
on_axis = data[0][len(data[0])//2]  # On-axis (0° azimuth, 0° elevation)
print(f"On-axis SPL: {on_axis:.1f} dB")
```

## API Reference

### GllFile

The main class for parsing GLL files.

- `GllFile.parse(source)` - Parse from file path or file-like object
- `gll.header` - File header information
- `gll.metadata` - Loudspeaker metadata
- `gll.gen_system` - System container information
- `gll.database` - Database with box types, sources, etc.
- `gll.resources` - List of embedded resources
- `gll.extract_resource(resource)` - Extract resource bytes
- `gll.extract_data_file(data_file)` - Extract data file bytes
- `gll.extract_include_file(include_file)` - Extract include file bytes
- `gll.get_balloon_at_frequency(source_index, frequency_hz)` - Get directivity data

### ArrayCalculator

Compute combined array responses.

- `ArrayCalculator(gll_file)` - Create calculator
- `calc.compute_response(config, receiver=None, air=None)` - Compute response
- `calc.available_box_types` - List of box type names
- `calc.available_sources` - List of source names

### ArrayConfig

Configure array elements.

- `ArrayConfig()` - Create empty configuration
- `config.add_element(box_type, splay=0, gain=0, ...)` - Add element
- `config.clear()` - Remove all elements

### Types

- `Vector3D(x, y, z)` - 3D coordinate
- `Metadata` - Product metadata
- `Database` - Database container
- `BoxType` - Speaker cabinet type
- `SourceDefinition` - Acoustic source with balloon data
- `TransferFunction` - Frequency/phase response
- `Resource` - Embedded resource
- `DataFile` - Embedded data file
- `IncludeFile` - Embedded documentation file

## Requirements

- Python 3.10+
- Go 1.21+ (for building from source)

## License

MIT License - See LICENSE file for details.

This project is for educational and interoperability purposes. GLL is a proprietary format owned by AFMG.
