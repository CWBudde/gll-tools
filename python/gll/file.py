"""GllFile class for parsing and accessing GLL file data."""

from pathlib import Path
from typing import BinaryIO

from . import _ffi
from .exceptions import GllError, ResourceError
from .types import (
    Database,
    DataFile,
    GenSystem,
    Header,
    IncludeFile,
    Metadata,
    Resource,
)


class GllFile:
    """Represents a parsed GLL file.

    Use GllFile.parse() to create an instance from a file path or file-like object.

    Example:
        >>> gll = GllFile.parse("speaker.gll")
        >>> print(gll.metadata.manufacturer)
        >>> print(gll.metadata.product_name)
        >>> for source in gll.database.source_definitions:
        ...     print(f"Source: {source.label}")
    """

    def __init__(
        self,
        data: dict,
        path: Path | None = None,
    ) -> None:
        """Initialize a GllFile from parsed data.

        Args:
            data: Parsed JSON data from the GLL library.
            path: Optional path to the source GLL file.
        """
        self._data = data
        self._path = path
        self._header: Header | None = None
        self._gen_system: GenSystem | None = None
        self._metadata: Metadata | None = None
        self._database: Database | None = None
        self._resources: list[Resource] | None = None

    @classmethod
    def parse(cls, source: str | Path | BinaryIO) -> "GllFile":
        """Parse a GLL file.

        Args:
            source: Either a file path (str or Path) or a file-like object
                   opened in binary mode.

        Returns:
            A GllFile instance with the parsed data.

        Raises:
            ParseError: If the file cannot be parsed.
            FileNotFoundError: If the file path does not exist.
        """
        if isinstance(source, (str, Path)):
            path = Path(source)
            if not path.exists():
                raise FileNotFoundError(f"GLL file not found: {path}")
            data = _ffi.parse_file(path)
            return cls(data, path)
        else:
            # Read from file-like object
            content = source.read()
            data = _ffi.parse_bytes(content)
            return cls(data, None)

    @property
    def path(self) -> Path | None:
        """Get the path to the source GLL file, if available."""
        return self._path

    @property
    def header(self) -> Header:
        """Get the GLL file header."""
        if self._header is None:
            self._header = Header.from_dict(self._data.get("header", {}))
        return self._header

    @property
    def gen_system(self) -> GenSystem:
        """Get the GenSystem container."""
        if self._gen_system is None:
            self._gen_system = GenSystem.from_dict(self._data.get("gen_system"))
        return self._gen_system

    @property
    def metadata(self) -> Metadata:
        """Get the loudspeaker metadata."""
        if self._metadata is None:
            self._metadata = Metadata.from_dict(self._data.get("metadata"))
        return self._metadata

    @property
    def database(self) -> Database:
        """Get the database containing box types, sources, etc."""
        if self._database is None:
            self._database = Database.from_dict(self._data.get("database"))
        return self._database

    @property
    def resources(self) -> list[Resource]:
        """Get the list of embedded resources."""
        if self._resources is None:
            self._resources = [
                Resource.from_dict(r, i)
                for i, r in enumerate(self._data.get("resources", []))
            ]
        return self._resources

    def extract_resource(self, resource: Resource | int) -> bytes:
        """Extract a resource's binary content.

        Args:
            resource: Either a Resource object or a resource index.

        Returns:
            The raw bytes of the resource.

        Raises:
            ResourceError: If the resource cannot be extracted.
            GllError: If the file was parsed from bytes (not a file path).
        """
        if self._path is None:
            raise GllError(
                "Cannot extract resources from GLL files parsed from bytes. "
                "Parse from a file path instead."
            )

        if isinstance(resource, Resource):
            index = resource.index
        else:
            index = resource

        return _ffi.extract_resource(self._path, index)

    def extract_data_file(self, data_file: DataFile | int) -> bytes:
        """Extract a data file's content.

        Args:
            data_file: Either a DataFile object or a data file index.

        Returns:
            The raw bytes of the data file.

        Raises:
            ResourceError: If the data file cannot be extracted.
            GllError: If the file was parsed from bytes.
        """
        if self._path is None:
            raise GllError(
                "Cannot extract data files from GLL files parsed from bytes."
            )

        if isinstance(data_file, DataFile):
            index = data_file.index
        else:
            index = data_file

        return _ffi.extract_data_file(self._path, index)

    def extract_include_file(self, include_file: IncludeFile | int) -> bytes:
        """Extract an include file's content (PDFs, documentation, etc.).

        Args:
            include_file: Either an IncludeFile object or an include file index.

        Returns:
            The raw bytes of the include file.

        Raises:
            ResourceError: If the include file cannot be extracted.
            GllError: If the file was parsed from bytes.
        """
        if self._path is None:
            raise GllError(
                "Cannot extract include files from GLL files parsed from bytes."
            )

        if isinstance(include_file, IncludeFile):
            index = include_file.index
        else:
            index = include_file

        return _ffi.extract_include_file(self._path, index)

    def get_balloon_at_frequency(
        self, source_index: int, frequency_hz: float
    ) -> dict:
        """Get balloon directivity data at a specific frequency.

        Args:
            source_index: Index of the source definition.
            frequency_hz: Frequency in Hz.

        Returns:
            Dictionary with balloon data including:
            - frequency: The actual frequency used
            - meridian_step: Azimuth step in degrees
            - parallel_step: Elevation step in degrees
            - symmetry: Symmetry type
            - data: 2D list of SPL values [meridian][parallel]

        Raises:
            GllError: If the file was parsed from bytes.
            ResourceError: If the source or balloon data cannot be found.
        """
        if self._path is None:
            raise GllError(
                "Cannot get balloon data from GLL files parsed from bytes."
            )

        return _ffi.get_balloon_at_frequency(self._path, source_index, frequency_hz)

    def __repr__(self) -> str:
        """Return a string representation of the GllFile."""
        name = self.metadata.product_name or self.gen_system.label or "Unknown"
        manufacturer = self.metadata.manufacturer or self.gen_system.company or ""
        if manufacturer:
            return f"GllFile({manufacturer!r}, {name!r})"
        return f"GllFile({name!r})"
