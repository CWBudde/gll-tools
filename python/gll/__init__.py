"""Python bindings for EASE GLL (Generic Loudspeaker Library) file parsing.

This package provides native Python access to GLL files, which are used by
AFMG's EASE acoustic simulation software to describe loudspeaker systems.

Basic usage:
    >>> from gll import GllFile
    >>> gll = GllFile.parse("speaker.gll")
    >>> print(gll.metadata.manufacturer)
    >>> print(gll.metadata.product_name)

Array response calculation:
    >>> from gll import GllFile, ArrayCalculator, ArrayConfig
    >>> gll = GllFile.parse("speaker.gll")
    >>> calc = ArrayCalculator(gll)
    >>> config = ArrayConfig()
    >>> config.add_element("K2", splay=0.5)
    >>> config.add_element("K2", splay=1.0)
    >>> response = calc.compute_response(config)

Resource extraction:
    >>> gll = GllFile.parse("speaker.gll")
    >>> for resource in gll.resources:
    ...     data = gll.extract_resource(resource)
    ...     with open(resource.name, 'wb') as f:
    ...         f.write(data)
"""

from .acoustics import (
    AirProperties,
    ArrayCalculator,
    ArrayConfig,
    ArrayElement,
    ArrayResponse,
)
from .exceptions import (
    ConfigurationError,
    GllError,
    ParseError,
    ResourceError,
)
from .file import GllFile
from .types import (
    AngularResolution,
    BalloonData,
    BoxSource,
    BoxType,
    CaseGeometry,
    Database,
    DataFile,
    FilterGroup,
    Frame,
    GenSystem,
    Header,
    IncludeFile,
    Limit,
    Metadata,
    Resource,
    SourceDefinition,
    SystemType,
    TransferFunction,
    TransferFunctionDef,
    Vector3D,
    Warning,
)

# Version from the shared library
try:
    from ._ffi import get_version

    __version__ = get_version()
except Exception:
    __version__ = "0.1.0"

__all__ = [
    # Main classes
    "GllFile",
    "ArrayCalculator",
    "ArrayConfig",
    "ArrayElement",
    "ArrayResponse",
    "AirProperties",
    # Exceptions
    "GllError",
    "ParseError",
    "ResourceError",
    "ConfigurationError",
    # Types
    "Header",
    "Metadata",
    "GenSystem",
    "Database",
    "Resource",
    "DataFile",
    "IncludeFile",
    "BoxType",
    "BoxSource",
    "CaseGeometry",
    "SourceDefinition",
    "BalloonData",
    "AngularResolution",
    "TransferFunction",
    "TransferFunctionDef",
    "Frame",
    "FilterGroup",
    "Limit",
    "Warning",
    "Vector3D",
    "SystemType",
    # Version
    "__version__",
]
