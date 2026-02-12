"""Tests for type definitions."""

import pytest

from gll import (
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


def test_vector3d_creation() -> None:
    """Test Vector3D creation."""
    v = Vector3D(1.0, 2.0, 3.0)
    assert v.x == 1.0
    assert v.y == 2.0
    assert v.z == 3.0


def test_vector3d_default() -> None:
    """Test Vector3D default values."""
    v = Vector3D()
    assert v.x == 0.0
    assert v.y == 0.0
    assert v.z == 0.0


def test_vector3d_from_dict() -> None:
    """Test Vector3D from dictionary."""
    v = Vector3D.from_dict({"x": 1.0, "y": 2.0, "z": 3.0})
    assert v.x == 1.0
    assert v.y == 2.0
    assert v.z == 3.0


def test_vector3d_from_dict_none() -> None:
    """Test Vector3D from None."""
    v = Vector3D.from_dict(None)
    assert v.x == 0.0


def test_vector3d_immutable() -> None:
    """Test Vector3D is immutable (frozen dataclass)."""
    v = Vector3D(1.0, 2.0, 3.0)
    with pytest.raises(AttributeError):
        v.x = 5.0  # type: ignore


def test_system_type_values() -> None:
    """Test SystemType enum values."""
    assert SystemType.LINE_ARRAY == 0
    assert SystemType.CLUSTER == 1
    assert SystemType.LOUDSPEAKER == 2


def test_header_from_dict() -> None:
    """Test Header from dictionary."""
    h = Header.from_dict({
        "magic": "EGLL",
        "format_id": "EASEGLL",
        "format_version": 6,
        "sub_version": 0,
    })
    assert h.magic == "EGLL"
    assert h.format_id == "EASEGLL"
    assert h.format_version == 6


def test_metadata_from_dict() -> None:
    """Test Metadata from dictionary."""
    m = Metadata.from_dict({
        "product_name": "Test Speaker",
        "manufacturer": "Test Co",
    })
    assert m.product_name == "Test Speaker"
    assert m.manufacturer == "Test Co"


def test_metadata_from_none() -> None:
    """Test Metadata from None."""
    m = Metadata.from_dict(None)
    assert m.product_name == ""


def test_gen_system_from_dict() -> None:
    """Test GenSystem from dictionary."""
    gs = GenSystem.from_dict({
        "label": "Test System",
        "company": "Test Co",
        "type": 0,
    })
    assert gs.label == "Test System"
    assert gs.company == "Test Co"
    assert gs.system_type == SystemType.LINE_ARRAY


def test_resource_from_dict() -> None:
    """Test Resource from dictionary."""
    r = Resource.from_dict({
        "type": "PNG",
        "name": "image.png",
        "offset": 1000,
        "size": 500,
    }, index=0)
    assert r.index == 0
    assert r.type == "PNG"
    assert r.name == "image.png"


def test_data_file_from_dict() -> None:
    """Test DataFile from dictionary."""
    df = DataFile.from_dict({
        "key": "model.xed",
        "filename": "model.xed",
        "size": 1000,
        "offset": 500,
    }, index=0)
    assert df.key == "model.xed"
    assert df.size == 1000


def test_include_file_from_dict() -> None:
    """Test IncludeFile from dictionary."""
    inc = IncludeFile.from_dict({
        "label": "Manual",
        "key": "manual.pdf",
        "filename": "manual.pdf",
        "size": 5000,
        "offset": 1000,
    }, index=0)
    assert inc.label == "Manual"
    assert inc.filename == "manual.pdf"


def test_transfer_function_def() -> None:
    """Test TransferFunctionDef."""
    tfd = TransferFunctionDef(
        start_frequency=20.0,
        end_frequency=20000.0,
        bands_per_octave=24,
    )
    assert tfd.start_frequency == 20.0
    assert tfd.get_frequency(0) == 20.0


def test_transfer_function_frequencies() -> None:
    """Test TransferFunction frequencies property."""
    tf = TransferFunction(
        level=[0.0] * 10,
        phase=[0.0] * 10,
    )
    freqs = tf.frequencies
    assert len(freqs) == 10


def test_angular_resolution_from_dict() -> None:
    """Test AngularResolution from dictionary."""
    ar = AngularResolution.from_dict({
        "meridian_step": 5.0,
        "parallel_step": 5.0,
        "symmetry": 0,
    })
    assert ar.meridian_step == 5.0
    assert ar.parallel_step == 5.0


def test_balloon_data_from_dict() -> None:
    """Test BalloonData from dictionary."""
    bd = BalloonData.from_dict({
        "angular_resolution": {
            "meridian_step": 5.0,
            "parallel_step": 5.0,
        },
        "responses": [],
    })
    assert bd.angular_resolution.meridian_step == 5.0
    assert len(bd.responses) == 0


def test_source_definition_from_dict() -> None:
    """Test SourceDefinition from dictionary."""
    sd = SourceDefinition.from_dict({
        "key": "source1",
        "definition": {
            "label": "Test Source",
            "sensitivity_1w_1m": 95.0,
            "impedance": 8.0,
        },
    })
    assert sd.key == "source1"
    assert sd.label == "Test Source"
    assert sd.sensitivity == 95.0


def test_box_source_from_dict() -> None:
    """Test BoxSource from dictionary."""
    bs = BoxSource.from_dict({
        "label": "HF",
        "key": "hf1",
        "position": {"x": 0, "y": 0, "z": 100},
        "source_def_key": "source1",
    })
    assert bs.label == "HF"
    assert bs.position.z == 100


def test_case_geometry_from_dict() -> None:
    """Test CaseGeometry from dictionary."""
    cg = CaseGeometry.from_dict({
        "is_symmetric": True,
        "symmetry_axis": 0.0,
        "vertices": [{"x": 0, "y": 0, "z": 0}],
        "edges": [{"v1": 0, "v2": 1}],
    })
    assert cg.is_symmetric is True
    assert cg.vertex_count == 1


def test_box_type_from_dict() -> None:
    """Test BoxType from dictionary."""
    bt = BoxType.from_dict({
        "label": "K2",
        "key": "k2",
        "weight": 25.5,
        "sources": ["source1"],
    })
    assert bt.label == "K2"
    assert bt.weight == 25.5
    assert "source1" in bt.sources


def test_frame_from_dict() -> None:
    """Test Frame from dictionary."""
    f = Frame.from_dict({
        "label": "Bumper",
        "key": "bumper1",
        "weight": 15.0,
    })
    assert f.label == "Bumper"
    assert f.weight == 15.0


def test_filter_group_from_dict() -> None:
    """Test FilterGroup from dictionary."""
    fg = FilterGroup.from_dict({
        "label": "HPF 80Hz",
        "key": "hpf80",
    })
    assert fg.label == "HPF 80Hz"


def test_limit_from_dict() -> None:
    """Test Limit from dictionary."""
    lim = Limit.from_dict({
        "label": "Max Power",
        "key": "maxpower",
    })
    assert lim.label == "Max Power"


def test_warning_from_dict() -> None:
    """Test Warning from dictionary."""
    w = Warning.from_dict({
        "label": "Overheat",
        "key": "overheat",
        "text": "Temperature too high",
    })
    assert w.label == "Overheat"
    assert w.text == "Temperature too high"


def test_database_from_dict() -> None:
    """Test Database from dictionary."""
    db = Database.from_dict({
        "box_types": [{"label": "K2", "key": "k2"}],
        "source_definitions": [{"key": "source1"}],
        "frames": [],
        "filter_groups": [],
    })
    assert len(db.box_types) == 1
    assert db.box_types[0].label == "K2"


def test_database_from_none() -> None:
    """Test Database from None."""
    db = Database.from_dict(None)
    assert len(db.box_types) == 0
