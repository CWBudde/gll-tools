"""Acoustic calculation classes for array response computation."""

import json
from dataclasses import dataclass, field
from typing import Any

from . import _ffi
from .exceptions import ConfigurationError, GllError
from .file import GllFile
from .types import FrequencyBalloon, TransferFunction, Vector3D


@dataclass
class AirProperties:
    """Air properties for acoustic calculations."""

    temperature: float = 20.0  # Celsius
    humidity: float = 0.5  # Relative humidity (0-1)
    pressure: float = 101.325  # kPa

    def to_dict(self) -> dict[str, float]:
        """Convert to dictionary for JSON serialization."""
        return {
            "temperature": self.temperature,
            "humidity": self.humidity,
            "pressure": self.pressure,
        }


@dataclass
class ArrayElement:
    """Single element in an array configuration."""

    box_type: str
    position: Vector3D = field(default_factory=Vector3D)
    angles: Vector3D = field(default_factory=Vector3D)
    gain: float = 0.0
    delay: float = 0.0
    muted: bool = False

    def to_dict(self) -> dict[str, Any]:
        """Convert to dictionary for JSON serialization."""
        return {
            "box_type": self.box_type,
            "position": {
                "x": self.position.x,
                "y": self.position.y,
                "z": self.position.z,
            },
            "angles": {"x": self.angles.x, "y": self.angles.y, "z": self.angles.z},
            "gain": self.gain,
            "delay": self.delay,
            "muted": self.muted,
        }


@dataclass
class ArrayConfig:
    """Line array configuration for response calculations.

    Example:
        >>> config = ArrayConfig()
        >>> config.add_element("K2", splay=0.5)
        >>> config.add_element("K2", splay=1.0)
        >>> config.add_element("K2", splay=1.5)
    """

    elements: list[ArrayElement] = field(default_factory=list)

    def add_element(
        self,
        box_type: str,
        *,
        position: Vector3D | None = None,
        splay: float = 0.0,
        tilt: float = 0.0,
        gain: float = 0.0,
        delay: float = 0.0,
        muted: bool = False,
    ) -> "ArrayConfig":
        """Add an element to the array.

        Args:
            box_type: Name or key of the box type to use.
            position: Position in meters. If None, position is calculated
                     based on previous elements.
            splay: Vertical splay angle in degrees.
            tilt: Horizontal tilt angle in degrees.
            gain: Per-element gain in dB.
            delay: Additional delay in seconds.
            muted: Whether this element is muted.

        Returns:
            Self for method chaining.
        """
        if position is None:
            position = Vector3D()

        # Convert degrees to radians for angles
        import math

        angles = Vector3D(
            x=math.radians(tilt),
            y=math.radians(splay),
            z=0.0,
        )

        element = ArrayElement(
            box_type=box_type,
            position=position,
            angles=angles,
            gain=gain,
            delay=delay,
            muted=muted,
        )
        self.elements.append(element)
        return self

    def clear(self) -> "ArrayConfig":
        """Remove all elements from the configuration."""
        self.elements.clear()
        return self

    def to_dict(self) -> dict[str, Any]:
        """Convert to dictionary for JSON serialization."""
        return {
            "elements": [e.to_dict() for e in self.elements],
        }


@dataclass
class ArrayResponse:
    """Result of an array response calculation."""

    transfer_function: TransferFunction | None = None
    combined_balloon: FrequencyBalloon | None = None
    element_contributions: list[TransferFunction] = field(default_factory=list)
    error: str | None = None

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> "ArrayResponse":
        """Create from dictionary."""
        tf = None
        if "transfer_function" in d and d["transfer_function"]:
            tf_data = d["transfer_function"]
            tf = TransferFunction.from_dict(tf_data)
        combined_balloon = None
        if "combined_balloon" in d and d["combined_balloon"]:
            combined_balloon = FrequencyBalloon.from_dict(d["combined_balloon"])
        return cls(
            transfer_function=tf,
            combined_balloon=combined_balloon,
            element_contributions=[
                TransferFunction.from_dict(c)
                for c in d.get("element_contributions", [])
            ],
            error=d.get("error"),
        )

    @property
    def is_valid(self) -> bool:
        """Check if the response is valid (no error)."""
        return self.error is None and self.transfer_function is not None


class ArrayCalculator:
    """Compute array responses for a GLL file.

    Example:
        >>> gll = GllFile.parse("speaker.gll")
        >>> calc = ArrayCalculator(gll)
        >>> config = ArrayConfig()
        >>> config.add_element("K2", splay=0.5)
        >>> config.add_element("K2", splay=1.0)
        >>> response = calc.compute_response(config)
        >>> if response.is_valid:
        ...     print(f"Levels: {response.transfer_function.level[:5]}")
    """

    def __init__(self, gll_file: GllFile) -> None:
        """Initialize the calculator with a GLL file.

        Args:
            gll_file: The GLL file to use for calculations.

        Raises:
            GllError: If the GLL file was parsed from bytes.
        """
        if gll_file.path is None:
            raise GllError(
                "ArrayCalculator requires a GLL file parsed from a file path, "
                "not from bytes."
            )
        self.gll_file = gll_file
        self._default_air = AirProperties()

    def compute_response(
        self,
        config: ArrayConfig,
        *,
        receiver: Vector3D | None = None,
        air: AirProperties | None = None,
        air_attenuation: bool = False,
        frequency: float | None = None,
    ) -> ArrayResponse:
        """Compute the combined array response.

        Args:
            config: Array configuration with elements.
            receiver: Receiver position in meters. Defaults to (0, 10, 0)
                     which is 10m directly in front.
            air: Air properties for calculations. Defaults to 20°C, 50% humidity,
                 and 101.325 kPa.
            air_attenuation: Whether to include air absorption. Default False.
            frequency: Optional single frequency in Hz for combined balloon output.

        Returns:
            ArrayResponse with the combined transfer function.

        Raises:
            ConfigurationError: If the configuration is invalid.
        """
        if not config.elements:
            raise ConfigurationError("Array configuration has no elements")

        if receiver is None:
            receiver = Vector3D(0, 10, 0)

        if air is None:
            air = self._default_air

        request = {
            "gll_path": str(self.gll_file.path),
            "elements": [e.to_dict() for e in config.elements],
            "receiver": {"x": receiver.x, "y": receiver.y, "z": receiver.z},
            "air": air.to_dict(),
            "air_atten": air_attenuation,
            "frequency": frequency,
        }

        result = _ffi.compute_array_response(json.dumps(request))
        return ArrayResponse.from_dict(result)

    def compute_response_grid(
        self,
        config: ArrayConfig,
        receivers: list[Vector3D],
        *,
        air: AirProperties | None = None,
        air_attenuation: bool = False,
    ) -> list[ArrayResponse]:
        """Compute array response at multiple receiver positions.

        This is more efficient than calling compute_response in a loop
        because it reuses parsed data.

        Args:
            config: Array configuration with elements.
            receivers: List of receiver positions in meters.
            air: Air properties for calculations.
            air_attenuation: Whether to include air absorption.

        Returns:
            List of ArrayResponse objects, one for each receiver.
        """
        return [
            self.compute_response(
                config,
                receiver=recv,
                air=air,
                air_attenuation=air_attenuation,
            )
            for recv in receivers
        ]

    @property
    def available_box_types(self) -> list[str]:
        """Get list of available box type names."""
        return [bt.label for bt in self.gll_file.database.box_types]

    @property
    def available_sources(self) -> list[str]:
        """Get list of available source definition names."""
        return [sd.label or sd.key for sd in self.gll_file.database.source_definitions]
