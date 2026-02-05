# C API Documentation

The gll-tools shared library (`libgll.so` / `libgll.dll` / `libgll.dylib`) provides a C-compatible API for parsing GLL files from any programming language that supports C FFI.

## Building the Library

```bash
# Linux/macOS
CGO_ENABLED=1 go build -buildmode=c-shared -o libgll.so ./cmd/gllpy

# Windows (requires MinGW)
CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 \
    go build -buildmode=c-shared -o libgll.dll ./cmd/gllpy
```

The build produces two files:
- `libgll.so` (or `.dll`/`.dylib`) - The shared library
- `libgll.h` - C header file with type definitions and function declarations

## Data Types

### GLL_Result

Used for functions returning JSON strings:

```c
typedef struct {
    char* data;      // JSON string (NULL on error)
    char* error;     // Error message (NULL on success)
    int64_t length;  // Length of data
} GLL_Result;
```

### GLL_ByteResult

Used for functions returning binary data:

```c
typedef struct {
    void* data;      // Binary data (NULL on error)
    int64_t length;  // Length of data
    char* error;     // Error message (NULL on success)
} GLL_ByteResult;
```

## API Functions

### GLL_Version

```c
char* GLL_Version(void);
```

Returns the library version string. Caller must free with `GLL_FreeString()`.

### GLL_ParseFile

```c
GLL_Result GLL_ParseFile(char* path);
```

Parse a GLL file and return its contents as JSON.

**Parameters:**
- `path`: Path to the GLL file (UTF-8 encoded)

**Returns:** `GLL_Result` with JSON data containing:
- `header`: File format information
- `metadata`: Product metadata
- `gen_system`: System container data
- `database`: Box types, sources, filters, etc.
- `resources`: List of embedded resources

**Example JSON output:**
```json
{
  "header": {
    "magic": "EGLL",
    "format_id": "EASE_GLL",
    "format_version": 6
  },
  "metadata": {
    "product_name": "K2",
    "manufacturer": "L-Acoustics"
  },
  "database": {
    "box_types": [...],
    "source_definitions": [...]
  }
}
```

### GLL_ParseBytes

```c
GLL_Result GLL_ParseBytes(char* data, int64_t length);
```

Parse GLL data from memory buffer.

### GLL_ExtractResource

```c
GLL_ByteResult GLL_ExtractResource(char* path, int32_t resourceIndex);
```

Extract an embedded resource (image, PDF, etc.) by index.

### GLL_ExtractDataFile

```c
GLL_ByteResult GLL_ExtractDataFile(char* path, int32_t dataFileIndex);
```

Extract a data file (3D geometry, etc.) by index.

### GLL_ExtractIncludeFile

```c
GLL_ByteResult GLL_ExtractIncludeFile(char* path, int32_t includeFileIndex);
```

Extract an include file (documentation PDF, etc.) by index.

### GLL_ComputeArrayResponse

```c
GLL_Result GLL_ComputeArrayResponse(char* configJSON);
```

Compute combined array response from a JSON configuration.

**Input JSON format:**
```json
{
  "gll_path": "/path/to/file.gll",
  "elements": [
    {"box_type": "K2", "angles": {"x": 0, "y": 0.0087, "z": 0}, "gain": 0},
    {"box_type": "K2", "angles": {"x": 0, "y": 0.0175, "z": 0}, "gain": 0}
  ],
  "receiver": {"x": 0, "y": 20, "z": 0},
  "air": {"temperature": 20, "humidity": 0.5},
  "air_atten": false
}
```

**Output JSON format:**
```json
{
  "transfer_function": {
    "level": [85.2, 86.1, ...],
    "phase": [0.12, 0.15, ...],
    "delay": 0.058
  }
}
```

### GLL_GetBalloonAtFrequency

```c
GLL_Result GLL_GetBalloonAtFrequency(char* path, int32_t sourceIndex, double frequencyHz);
```

Get directivity balloon data at a specific frequency.

**Output JSON format:**
```json
{
  "frequency": 1000,
  "meridian_step": 5,
  "parallel_step": 5,
  "symmetry": 0,
  "data": [[...], [...], ...]
}
```

### Memory Management

Always free returned data to prevent memory leaks:

```c
void GLL_FreeResult(GLL_Result result);
void GLL_FreeByteResult(GLL_ByteResult result);
void GLL_FreeString(char* s);
```

## Language Examples

### C

```c
#include "libgll.h"
#include <stdio.h>

int main() {
    GLL_Result result = GLL_ParseFile("speaker.gll");

    if (result.error != NULL) {
        printf("Error: %s\n", result.error);
        GLL_FreeResult(result);
        return 1;
    }

    printf("JSON: %s\n", result.data);
    GLL_FreeResult(result);
    return 0;
}
```

Compile with:
```bash
gcc -o example example.c -L. -lgll -Wl,-rpath,.
```

### LabVIEW

Use the **Call Library Function Node** to call the C functions:

1. **Configure the node:**
   - Library Name: `libgll.so` (or full path)
   - Function Name: `GLL_ParseFile`
   - Calling Convention: C
   - Thread: Run in any thread

2. **Parameter configuration for GLL_ParseFile:**
   - `path`: String (C String Pointer)
   - Return: Cluster of (data: String Pointer, error: String Pointer, length: I64)

3. **Memory cleanup:**
   - Call `GLL_FreeResult` after processing the result
   - Use another Call Library Function Node

4. **JSON parsing:**
   - Use LabVIEW's JSON parsing VIs to extract data from the returned JSON string

**LabVIEW Tips:**
- Use `Flatten To String` for cluster handling
- The returned strings are null-terminated C strings
- Always call the Free functions to prevent memory leaks
- Consider using the `Unflatten From JSON` VI for parsing results

### MATLAB

```matlab
% Load the library
if ~libisloaded('gll')
    loadlibrary('libgll.so', 'libgll.h');
end

% Parse a file
result = calllib('gll', 'GLL_ParseFile', 'speaker.gll');

if isempty(result.error)
    json_data = result.data;
    data = jsondecode(json_data);
    disp(data.metadata.product_name);
else
    error(result.error);
end

% Free memory
calllib('gll', 'GLL_FreeResult', result);

% Unload when done
unloadlibrary('gll');
```

### C# / .NET

```csharp
using System;
using System.Runtime.InteropServices;

public class GllLibrary
{
    [StructLayout(LayoutKind.Sequential)]
    public struct GLL_Result
    {
        public IntPtr data;
        public IntPtr error;
        public long length;
    }

    [DllImport("libgll.so", CallingConvention = CallingConvention.Cdecl)]
    public static extern GLL_Result GLL_ParseFile(string path);

    [DllImport("libgll.so", CallingConvention = CallingConvention.Cdecl)]
    public static extern void GLL_FreeResult(GLL_Result result);

    public static string ParseFile(string path)
    {
        var result = GLL_ParseFile(path);
        try
        {
            if (result.error != IntPtr.Zero)
            {
                throw new Exception(Marshal.PtrToStringAnsi(result.error));
            }
            return Marshal.PtrToStringAnsi(result.data);
        }
        finally
        {
            GLL_FreeResult(result);
        }
    }
}

// Usage
var json = GllLibrary.ParseFile("speaker.gll");
Console.WriteLine(json);
```

### Rust

```rust
use std::ffi::{CStr, CString};
use std::os::raw::{c_char, c_int};

#[repr(C)]
pub struct GLL_Result {
    pub data: *mut c_char,
    pub error: *mut c_char,
    pub length: i64,
}

#[link(name = "gll")]
extern "C" {
    fn GLL_ParseFile(path: *const c_char) -> GLL_Result;
    fn GLL_FreeResult(result: GLL_Result);
}

fn parse_gll(path: &str) -> Result<String, String> {
    let c_path = CString::new(path).unwrap();

    unsafe {
        let result = GLL_ParseFile(c_path.as_ptr());

        let output = if !result.error.is_null() {
            Err(CStr::from_ptr(result.error).to_string_lossy().into_owned())
        } else {
            Ok(CStr::from_ptr(result.data).to_string_lossy().into_owned())
        };

        GLL_FreeResult(result);
        output
    }
}
```

## Error Handling

All functions follow the same error pattern:
- On success: `error` is NULL, `data` contains the result
- On failure: `error` contains the error message, `data` is NULL

Always check the `error` field before accessing `data`.

## Thread Safety

The library is thread-safe for:
- Concurrent parsing of different files
- Concurrent resource extraction from the same file

Not thread-safe for:
- Concurrent writes to the same output buffer

## Performance Considerations

- **File Parsing:** ~10-100ms for typical files (1-20MB)
- **Resource Extraction:** Depends on resource size, zlib decompression adds overhead
- **Array Response:** ~1-10ms per receiver position

For batch operations, parse the file once and reuse the handle when possible (Python bindings do this automatically).

## Platform Notes

### Linux
- Requires glibc 2.17+ (RHEL/CentOS 7+, Ubuntu 14.04+)
- Set `LD_LIBRARY_PATH` or use `-rpath` when linking

### macOS
- Requires macOS 10.13+ (High Sierra)
- Set `DYLD_LIBRARY_PATH` or use `@rpath`

### Windows
- Requires Windows 10+
- Place DLL in same directory as executable or in PATH
- May need Visual C++ Redistributable
