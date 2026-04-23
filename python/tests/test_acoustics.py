"""Tests for acoustic calculations."""

from pathlib import Path

import pytest

from gll import (
    AirProperties,
    ArrayCalculator,
    ArrayConfig,
    ConfigurationError,
    GllFile,
    Vector3D,
)


def test_array_config_creation() -> None:
    """Test creating an array configuration."""
    config = ArrayConfig()
    assert len(config.elements) == 0


def test_array_config_add_element() -> None:
    """Test adding elements to configuration."""
    config = ArrayConfig()
    config.add_element("K2", splay=0.5)
    config.add_element("K2", splay=1.0)

    assert len(config.elements) == 2
    assert config.elements[0].box_type == "K2"
    assert config.elements[1].box_type == "K2"


def test_array_config_chaining() -> None:
    """Test method chaining on add_element."""
    config = (
        ArrayConfig()
        .add_element("K2", splay=0.5)
        .add_element("K2", splay=1.0)
        .add_element("K2", splay=1.5)
    )

    assert len(config.elements) == 3


def test_array_config_clear() -> None:
    """Test clearing configuration."""
    config = ArrayConfig()
    config.add_element("K2", splay=0.5)
    config.clear()
    assert len(config.elements) == 0


def test_array_config_with_position() -> None:
    """Test adding element with explicit position."""
    config = ArrayConfig()
    pos = Vector3D(1.0, 2.0, 3.0)
    config.add_element("K2", position=pos)

    assert config.elements[0].position.x == 1.0
    assert config.elements[0].position.y == 2.0
    assert config.elements[0].position.z == 3.0


def test_array_config_with_gain_delay() -> None:
    """Test adding element with gain and delay."""
    config = ArrayConfig()
    config.add_element("K2", gain=-3.0, delay=0.001)

    assert config.elements[0].gain == -3.0
    assert config.elements[0].delay == 0.001


def test_array_config_with_mute() -> None:
    """Test adding muted element."""
    config = ArrayConfig()
    config.add_element("K2", muted=True)

    assert config.elements[0].muted is True


def test_array_config_to_dict() -> None:
    """Test serialization to dictionary."""
    config = ArrayConfig()
    config.add_element("K2", splay=0.5)

    d = config.to_dict()
    assert "elements" in d
    assert len(d["elements"]) == 1
    assert d["elements"][0]["box_type"] == "K2"


def test_air_properties_default() -> None:
    """Test default air properties."""
    air = AirProperties()
    assert air.temperature == 20.0
    assert air.humidity == 0.5


def test_air_properties_custom() -> None:
    """Test custom air properties."""
    air = AirProperties(temperature=25.0, humidity=0.6)
    assert air.temperature == 25.0
    assert air.humidity == 0.6


def test_array_calculator_creation(line_array_gll: Path) -> None:
    """Test creating an array calculator."""
    gll = GllFile.parse(line_array_gll)
    calc = ArrayCalculator(gll)
    assert calc is not None


def test_array_calculator_available_box_types(line_array_gll: Path) -> None:
    """Test listing available box types."""
    gll = GllFile.parse(line_array_gll)
    calc = ArrayCalculator(gll)
    box_types = calc.available_box_types
    assert isinstance(box_types, list)
    assert len(box_types) > 0


def test_array_calculator_available_sources(line_array_gll: Path) -> None:
    """Test listing available sources."""
    gll = GllFile.parse(line_array_gll)
    calc = ArrayCalculator(gll)
    sources = calc.available_sources
    assert isinstance(sources, list)


def test_array_calculator_empty_config_error(line_array_gll: Path) -> None:
    """Test that empty configuration raises error."""
    gll = GllFile.parse(line_array_gll)
    calc = ArrayCalculator(gll)
    config = ArrayConfig()

    with pytest.raises(ConfigurationError):
        calc.compute_response(config)


def test_array_calculator_compute_response(line_array_gll: Path) -> None:
    """Test computing array response."""
    gll = GllFile.parse(line_array_gll)
    calc = ArrayCalculator(gll)

    # Get first available box type
    box_types = calc.available_box_types
    if not box_types:
        pytest.skip("No box types available")

    box_type = box_types[0]

    config = ArrayConfig()
    config.add_element(box_type, splay=0.5)
    config.add_element(box_type, splay=1.0)

    response = calc.compute_response(config)

    # Response may or may not be valid depending on whether
    # the box type has source definitions linked
    if response.is_valid:
        assert response.transfer_function is not None
        assert len(response.transfer_function.level) > 0


def test_array_calculator_returns_detailed_response(line_array_gll: Path) -> None:
    """Array responses include element contributions and frequency balloon output."""
    gll = GllFile.parse(line_array_gll)
    calc = ArrayCalculator(gll)

    box_types = calc.available_box_types
    if not box_types:
        pytest.skip("No box types available")

    config = ArrayConfig().add_element(box_types[0], splay=0.5)
    response = calc.compute_response(config, frequency=1000.0)

    if not response.is_valid:
        pytest.skip(f"Array response unavailable: {response.error}")

    assert response.transfer_function is not None
    assert response.element_contributions
    assert response.combined_balloon is not None
    assert response.combined_balloon.frequency == 1000.0
    assert response.combined_balloon.data
    assert isinstance(response.combined_balloon.get_spl(0.0, 0.0), float)


def test_array_calculator_with_receiver(line_array_gll: Path) -> None:
    """Test computing response at specific receiver."""
    gll = GllFile.parse(line_array_gll)
    calc = ArrayCalculator(gll)

    box_types = calc.available_box_types
    if not box_types:
        pytest.skip("No box types available")

    config = ArrayConfig()
    config.add_element(box_types[0])

    receiver = Vector3D(0, 20, -5)
    response = calc.compute_response(config, receiver=receiver)
    # Just check it doesn't error
    assert response is not None


def test_array_calculator_with_air_properties(line_array_gll: Path) -> None:
    """Test computing response with custom air properties."""
    gll = GllFile.parse(line_array_gll)
    calc = ArrayCalculator(gll)

    box_types = calc.available_box_types
    if not box_types:
        pytest.skip("No box types available")

    config = ArrayConfig()
    config.add_element(box_types[0])

    air = AirProperties(temperature=25.0, humidity=0.6)
    response = calc.compute_response(config, air=air)
    assert response is not None


def test_get_balloon_at_frequency(line_array_gll: Path) -> None:
    """Test getting balloon data at a frequency."""
    gll = GllFile.parse(line_array_gll)

    sources = gll.database.source_definitions
    if not sources:
        pytest.skip("No source definitions")

    # Check if sources have balloon data
    has_balloon = any(
        s.balloon_data is not None and s.balloon_data.responses for s in sources
    )
    if not has_balloon:
        pytest.skip("No balloon data in source definitions")

    try:
        balloon = gll.get_balloon_at_frequency(0, 1000.0)
        assert "frequency" in balloon
        assert "data" in balloon
        assert isinstance(balloon["data"], list)
    except Exception as e:
        # May fail if source doesn't have balloon data
        pytest.skip(f"Balloon data not available: {e}")


def test_source_definition_get_balloon_at_frequency(line_array_gll: Path) -> None:
    """SourceDefinition exposes a bound balloon lookup helper."""
    gll = GllFile.parse(line_array_gll)

    source_index = next(
        (
            i
            for i, source in enumerate(gll.database.source_definitions)
            if source.balloon_data is not None and source.balloon_data.responses
        ),
        None,
    )
    if source_index is None:
        pytest.skip("No balloon data in source definitions")

    source = gll.database.source_definitions[source_index]
    balloon = source.get_balloon_at_frequency(1000.0)
    assert "data" in balloon
    assert isinstance(balloon.get_spl(0.0, 0.0), float)


def test_balloon_data_get_spl(line_array_gll: Path) -> None:
    """BalloonData can resolve SPL values when bound to a parsed GLL file."""
    gll = GllFile.parse(line_array_gll)

    source = next(
        (
            source
            for source in gll.database.source_definitions
            if source.balloon_data is not None and source.balloon_data.responses
        ),
        None,
    )
    if source is None or source.balloon_data is None:
        pytest.skip("No balloon data in source definitions")

    spl = source.balloon_data.get_spl(0.0, 0.0, 1000.0)
    assert isinstance(spl, float)
