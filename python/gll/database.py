"""Database-related wrapper classes for the gll package.

This module exists as a stable import location for database container types.
The concrete dataclass implementations still live in ``gll.types`` so the
existing API remains backward-compatible.
"""

from .types import (
    BoxSource,
    BoxType,
    CaseGeometry,
    DataFile,
    Database,
    FilterGroup,
    Frame,
    IncludeFile,
    Limit,
    Warning,
)

__all__ = [
    "Database",
    "DataFile",
    "IncludeFile",
    "BoxType",
    "BoxSource",
    "CaseGeometry",
    "Frame",
    "FilterGroup",
    "Limit",
    "Warning",
]
