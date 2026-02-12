"""Type definitions for the gll package."""

from dataclasses import dataclass, field
from enum import IntEnum
from typing import Any, Optional


class SystemType(IntEnum):
    """Type of loudspeaker system."""

    LINE_ARRAY = 0
    CLUSTER = 1
    LOUDSPEAKER = 2


@dataclass(frozen=True)
class Vector3D:
    """3D coordinate or angle vector."""

    x: float = 0.0
    y: float = 0.0
    z: float = 0.0

    @classmethod
    def from_dict(cls, d: dict[str, Any] | None) -> "Vector3D":
        """Create from dictionary."""
        if d is None:
            return cls()
        return cls(x=d.get("x", 0.0), y=d.get("y", 0.0), z=d.get("z", 0.0))


@dataclass
class Header:
    """GLL file header information."""

    magic: str
    format_id: str
    format_version: int
    sub_version: int

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> "Header":
        """Create from dictionary."""
        return cls(
            magic=d.get("magic", ""),
            format_id=d.get("format_id", ""),
            format_version=d.get("format_version", 0),
            sub_version=d.get("sub_version", 0),
        )


@dataclass
class Metadata:
    """Loudspeaker metadata."""

    product_name: str = ""
    display_name: str = ""
    manufacturer: str = ""
    description: str = ""
    copyright: str = ""
    website: str = ""
    email: str = ""

    @classmethod
    def from_dict(cls, d: dict[str, Any] | None) -> "Metadata":
        """Create from dictionary."""
        if d is None:
            return cls()
        return cls(
            product_name=d.get("product_name", ""),
            display_name=d.get("display_name", ""),
            manufacturer=d.get("manufacturer", ""),
            description=d.get("description", ""),
            copyright=d.get("copyright", ""),
            website=d.get("website", ""),
            email=d.get("email", ""),
        )


@dataclass
class GenSystem:
    """Main GLL system container."""

    label: str = ""
    version: float = 0.0
    key: str = ""
    system_type: SystemType = SystemType.LOUDSPEAKER
    company: str = ""
    info_text: str = ""
    copyright_text: str = ""
    support_text: str = ""
    website_text: str = ""
    email_text: str = ""

    @classmethod
    def from_dict(cls, d: dict[str, Any] | None) -> "GenSystem":
        """Create from dictionary."""
        if d is None:
            return cls()
        return cls(
            label=d.get("label", ""),
            version=d.get("version", 0.0),
            key=d.get("key", ""),
            system_type=SystemType(d.get("type", 2)),
            company=d.get("company", ""),
            info_text=d.get("info_text", ""),
            copyright_text=d.get("copyright_text", ""),
            support_text=d.get("support_text", ""),
            website_text=d.get("website_text", ""),
            email_text=d.get("email_text", ""),
        )


@dataclass
class Resource:
    """Embedded resource in a GLL file."""

    index: int
    type: str
    name: str
    offset: int
    size: int
    decompressed_size: int = 0

    @classmethod
    def from_dict(cls, d: dict[str, Any], index: int) -> "Resource":
        """Create from dictionary."""
        return cls(
            index=index,
            type=d.get("type", "UNKNOWN"),
            name=d.get("name", ""),
            offset=d.get("offset", 0),
            size=d.get("size", 0),
            decompressed_size=d.get("decompressed_size", 0),
        )


@dataclass
class DataFile:
    """Embedded data file (geometry, images, etc.)."""

    index: int
    key: str
    filename: str
    size: int
    offset: int

    @classmethod
    def from_dict(cls, d: dict[str, Any], index: int) -> "DataFile":
        """Create from dictionary."""
        return cls(
            index=index,
            key=d.get("key", ""),
            filename=d.get("filename", ""),
            size=d.get("size", 0),
            offset=d.get("offset", 0),
        )


@dataclass
class IncludeFile:
    """Additional data file (PDFs, documentation, etc.)."""

    index: int
    label: str
    key: str
    filename: str
    size: int
    offset: int

    @classmethod
    def from_dict(cls, d: dict[str, Any], index: int) -> "IncludeFile":
        """Create from dictionary."""
        return cls(
            index=index,
            label=d.get("label", ""),
            key=d.get("key", ""),
            filename=d.get("filename", ""),
            size=d.get("size", 0),
            offset=d.get("offset", 0),
        )


@dataclass
class TransferFunctionDef:
    """Transfer function definition (frequency range)."""

    start_frequency: float = 20.0
    end_frequency: float = 20000.0
    bands_per_octave: int = 24

    @classmethod
    def from_dict(cls, d: dict[str, Any] | None) -> "TransferFunctionDef":
        """Create from dictionary."""
        if d is None:
            return cls()
        return cls(
            start_frequency=d.get("start_frequency", 20.0),
            end_frequency=d.get("end_frequency", 20000.0),
            bands_per_octave=d.get("bands_per_octave", 24),
        )

    def get_frequency(self, index: int) -> float:
        """Get frequency at given index."""
        import math

        octaves = math.log2(self.end_frequency / self.start_frequency)
        total_bands = int(octaves * self.bands_per_octave)
        if index < 0 or index >= total_bands:
            return 0.0
        return self.start_frequency * (2.0 ** (index / self.bands_per_octave))


@dataclass
class TransferFunction:
    """Frequency/phase response data."""

    definition: TransferFunctionDef = field(default_factory=TransferFunctionDef)
    level: list[float] = field(default_factory=list)
    phase: list[float] = field(default_factory=list)
    delay: float = 0.0

    @classmethod
    def from_dict(cls, d: dict[str, Any] | None) -> "TransferFunction":
        """Create from dictionary."""
        if d is None:
            return cls()
        return cls(
            definition=TransferFunctionDef.from_dict(d.get("definition")),
            level=d.get("level", []),
            phase=d.get("phase", []),
            delay=d.get("delay", 0.0),
        )

    @property
    def frequencies(self) -> list[float]:
        """Get list of frequencies for each band."""
        return [self.definition.get_frequency(i) for i in range(len(self.level))]


@dataclass
class AngularResolution:
    """Balloon angular resolution parameters."""

    meridian_step: float = 5.0
    parallel_step: float = 5.0
    symmetry: int = 0
    front_half_only: bool = False

    @classmethod
    def from_dict(cls, d: dict[str, Any] | None) -> "AngularResolution":
        """Create from dictionary."""
        if d is None:
            return cls()
        return cls(
            meridian_step=d.get("meridian_step", 5.0),
            parallel_step=d.get("parallel_step", 5.0),
            symmetry=d.get("symmetry", 0),
            front_half_only=d.get("front_half_only", False),
        )


@dataclass
class BalloonData:
    """Directivity balloon data."""

    angular_resolution: AngularResolution = field(default_factory=AngularResolution)
    responses: list[TransferFunction] = field(default_factory=list)

    @classmethod
    def from_dict(cls, d: dict[str, Any] | None) -> "BalloonData":
        """Create from dictionary."""
        if d is None:
            return cls()
        return cls(
            angular_resolution=AngularResolution.from_dict(d.get("angular_resolution")),
            responses=[
                TransferFunction.from_dict(r) for r in d.get("responses", [])
            ],
        )


@dataclass
class SourceDefinition:
    """Acoustic source definition with balloon data."""

    key: str = ""
    label: str = ""
    sensitivity: float = 0.0
    impedance: float = 8.0
    max_power: float = 0.0
    balloon_data: Optional[BalloonData] = None
    frequency_response: Optional[TransferFunction] = None

    @classmethod
    def from_dict(cls, d: dict[str, Any] | None) -> "SourceDefinition":
        """Create from dictionary."""
        if d is None:
            return cls()

        definition = d.get("definition", {})
        balloon = None
        if definition and "balloon_data" in definition:
            balloon = BalloonData.from_dict(definition.get("balloon_data"))

        freq_response = None
        if definition and "frequency_response" in definition:
            freq_response = TransferFunction.from_dict(
                definition.get("frequency_response")
            )

        return cls(
            key=d.get("key", ""),
            label=definition.get("label", "") if definition else "",
            sensitivity=definition.get("sensitivity_1w_1m", 0.0) if definition else 0.0,
            impedance=definition.get("impedance", 8.0) if definition else 8.0,
            max_power=definition.get("max_power", 0.0) if definition else 0.0,
            balloon_data=balloon,
            frequency_response=freq_response,
        )


@dataclass
class BoxSource:
    """Source placement inside a box type."""

    label: str = ""
    key: str = ""
    position: Vector3D = field(default_factory=Vector3D)
    angles: Vector3D = field(default_factory=Vector3D)
    source_def_key: str = ""

    @classmethod
    def from_dict(cls, d: dict[str, Any] | None) -> "BoxSource":
        """Create from dictionary."""
        if d is None:
            return cls()
        return cls(
            label=d.get("label", ""),
            key=d.get("key", ""),
            position=Vector3D.from_dict(d.get("position")),
            angles=Vector3D.from_dict(d.get("angles")),
            source_def_key=d.get("source_def_key", ""),
        )


@dataclass
class CaseGeometry:
    """3D cabinet/frame geometry."""

    is_symmetric: bool = False
    symmetry_axis: float = 0.0
    vertex_count: int = 0
    edge_count: int = 0
    face_count: int = 0

    @classmethod
    def from_dict(cls, d: dict[str, Any] | None) -> "CaseGeometry":
        """Create from dictionary."""
        if d is None:
            return cls()
        return cls(
            is_symmetric=d.get("is_symmetric", False),
            symmetry_axis=d.get("symmetry_axis", 0.0),
            vertex_count=len(d.get("vertices", [])),
            edge_count=len(d.get("edges", [])),
            face_count=len(d.get("faces", [])),
        )


@dataclass
class BoxType:
    """Speaker cabinet type."""

    label: str = ""
    key: str = ""
    sources: list[str] = field(default_factory=list)
    source_placements: list[BoxSource] = field(default_factory=list)
    case_geometry: Optional[CaseGeometry] = None
    next_pivot: Optional[Vector3D] = None
    reference_point: Optional[Vector3D] = None
    center_of_mass: Optional[Vector3D] = None
    weight: float = 0.0
    vertical_opening_angle: float = 0.0
    horizontal_opening_angle: float = 0.0

    @classmethod
    def from_dict(cls, d: dict[str, Any] | None) -> "BoxType":
        """Create from dictionary."""
        if d is None:
            return cls()
        return cls(
            label=d.get("label", ""),
            key=d.get("key", ""),
            sources=d.get("sources", []),
            source_placements=[
                BoxSource.from_dict(s) for s in d.get("source_placements", [])
            ],
            case_geometry=CaseGeometry.from_dict(d.get("case_geometry")),
            next_pivot=Vector3D.from_dict(d.get("next_pivot")),
            reference_point=Vector3D.from_dict(d.get("reference_point")),
            center_of_mass=Vector3D.from_dict(d.get("center_of_mass")),
            weight=d.get("weight", 0.0),
            vertical_opening_angle=d.get("vertical_opening_angle", 0.0),
            horizontal_opening_angle=d.get("horizontal_opening_angle", 0.0),
        )


@dataclass
class Frame:
    """Rigging frame for line arrays."""

    label: str = ""
    key: str = ""
    weight: float = 0.0

    @classmethod
    def from_dict(cls, d: dict[str, Any] | None) -> "Frame":
        """Create from dictionary."""
        if d is None:
            return cls()
        return cls(
            label=d.get("label", ""),
            key=d.get("key", ""),
            weight=d.get("weight", 0.0),
        )


@dataclass
class FilterGroup:
    """Filter group for signal processing."""

    label: str = ""
    key: str = ""

    @classmethod
    def from_dict(cls, d: dict[str, Any] | None) -> "FilterGroup":
        """Create from dictionary."""
        if d is None:
            return cls()
        return cls(
            label=d.get("label", ""),
            key=d.get("key", ""),
        )


@dataclass
class Limit:
    """Operating limit definition."""

    label: str = ""
    key: str = ""

    @classmethod
    def from_dict(cls, d: dict[str, Any] | None) -> "Limit":
        """Create from dictionary."""
        if d is None:
            return cls()
        return cls(
            label=d.get("label", ""),
            key=d.get("key", ""),
        )


@dataclass
class Warning:
    """Warning message definition."""

    label: str = ""
    key: str = ""
    text: str = ""

    @classmethod
    def from_dict(cls, d: dict[str, Any] | None) -> "Warning":
        """Create from dictionary."""
        if d is None:
            return cls()
        return cls(
            label=d.get("label", ""),
            key=d.get("key", ""),
            text=d.get("text", ""),
        )


@dataclass
class Database:
    """Container for all GLL database components."""

    data_files: list[DataFile] = field(default_factory=list)
    box_types: list[BoxType] = field(default_factory=list)
    frames: list[Frame] = field(default_factory=list)
    source_definitions: list[SourceDefinition] = field(default_factory=list)
    filter_groups: list[FilterGroup] = field(default_factory=list)
    limits: list[Limit] = field(default_factory=list)
    warnings: list[Warning] = field(default_factory=list)
    include_files: list[IncludeFile] = field(default_factory=list)

    @classmethod
    def from_dict(cls, d: dict[str, Any] | None) -> "Database":
        """Create from dictionary."""
        if d is None:
            return cls()
        return cls(
            data_files=[
                DataFile.from_dict(df, i)
                for i, df in enumerate(d.get("data_files", []))
            ],
            box_types=[BoxType.from_dict(bt) for bt in d.get("box_types", [])],
            frames=[Frame.from_dict(f) for f in d.get("frames", [])],
            source_definitions=[
                SourceDefinition.from_dict(sd)
                for sd in d.get("source_definitions", [])
            ],
            filter_groups=[
                FilterGroup.from_dict(fg) for fg in d.get("filter_groups", [])
            ],
            limits=[Limit.from_dict(lim) for lim in d.get("limits", [])],
            warnings=[Warning.from_dict(w) for w in d.get("warnings", [])],
            include_files=[
                IncludeFile.from_dict(inc, i)
                for i, inc in enumerate(d.get("include_files", []))
            ],
        )
