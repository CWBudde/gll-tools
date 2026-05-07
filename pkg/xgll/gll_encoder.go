package xgll

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	internalgll "github.com/cwbudde/gll-tools/internal/gll"
	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

type gllEncoder struct {
	// Destination writer for binary output
	w io.Writer
}

func newGLLEncoder(w io.Writer) *gllEncoder {
	// Create encoder with writer
	return &gllEncoder{w: w}
}

func (e *gllEncoder) writeFile(file *gllbin.File) error {
	// Validate file input
	if file == nil {
		return fmt.Errorf("file is nil")
	}

	// Write header first
	if err := e.writeHeader(file.Header); err != nil {
		return err
	}

	// Write GenSystem block (raw if provided)
	if len(file.GenSystem.RawBlock) > 0 {
		if _, err := e.w.Write(file.GenSystem.RawBlock); err != nil {
			return fmt.Errorf("write gensystem raw: %w", err)
		}
	} else {
		genSystem, err := e.encodeGenSystem(file)
		if err != nil {
			return err
		}

		if _, err := e.w.Write(genSystem); err != nil {
			return fmt.Errorf("write gensystem: %w", err)
		}
	}

	// Optional tail bytes
	if len(file.RawTail) > 0 {
		if _, err := e.w.Write(file.RawTail); err != nil {
			return fmt.Errorf("write tail: %w", err)
		}
	}

	// Done writing file
	return nil
}

func (e *gllEncoder) writeHeader(header gllbin.Header) error {
	// Magic header
	if _, err := e.w.Write([]byte(internalgll.MagicEGLL)); err != nil {
		return fmt.Errorf("write magic: %w", err)
	}

	// Reserved field
	if err := binary.Write(e.w, binary.LittleEndian, int32(0)); err != nil {
		return fmt.Errorf("write reserved: %w", err)
	}

	// Format identifier
	if err := writeString(e.w, internalgll.MagicEASEGLL); err != nil {
		return fmt.Errorf("write format id: %w", err)
	}

	// Header versioning
	version := header.FormatVersion
	if version == 0 {
		version = defaultBinaryFormatVersion
	}

	// Format version
	if err := binary.Write(e.w, binary.LittleEndian, version); err != nil {
		return fmt.Errorf("write version: %w", err)
	}

	// Sub-version
	if err := binary.Write(e.w, binary.LittleEndian, header.SubVersion); err != nil {
		return fmt.Errorf("write sub-version: %w", err)
	}

	// Optional checksum
	if version >= 4 {
		if _, err := e.w.Write(header.Checksum[:]); err != nil {
			return fmt.Errorf("write checksum: %w", err)
		}
	}

	// Optional hash
	if version >= 6 {
		hashLen := int32(0)
		if hasHash(header.HashID) {
			hashLen = int32(len(header.HashID))
		}
		if err := binary.Write(e.w, binary.LittleEndian, hashLen); err != nil {
			return fmt.Errorf("write hash length: %w", err)
		}

		if hashLen > 0 {
			if _, err := e.w.Write(header.HashID[:]); err != nil {
				return fmt.Errorf("write hash: %w", err)
			}
		}
	}

	// Header complete
	return nil
}

func (e *gllEncoder) encodeGenSystem(file *gllbin.File) ([]byte, error) {
	// Serialize GenSystem to block
	sys := file.GenSystem
	var payload bytes.Buffer

	// Basic identifiers
	if err := writeString(&payload, sys.Label); err != nil {
		return nil, err
	}

	if err := binary.Write(&payload, binary.LittleEndian, sys.Version); err != nil {
		return nil, fmt.Errorf("write version: %w", err)
	}

	if err := writeString(&payload, sys.Key); err != nil {
		return nil, err
	}

	if err := binary.Write(&payload, binary.LittleEndian, int32(sys.Type)); err != nil {
		return nil, fmt.Errorf("write type: %w", err)
	}

	if err := writeString(&payload, sys.Company); err != nil {
		return nil, err
	}

	if err := writeString(&payload, sys.InfoText); err != nil {
		return nil, err
	}

	if err := writeString(&payload, sys.CopyrightText); err != nil {
		return nil, err
	}

	if err := writeString(&payload, sys.SupportText); err != nil {
		return nil, err
	}

	if err := writeString(&payload, sys.WebsiteText); err != nil {
		return nil, err
	}

	if err := writeString(&payload, sys.EmailText); err != nil {
		return nil, err
	}

	if err := binary.Write(&payload, binary.LittleEndian, sys.BackgroundColor); err != nil {
		return nil, fmt.Errorf("write background color: %w", err)
	}

	// Encode database block
	db, err := e.encodeDatabase(file.Database)
	if err != nil {
		return nil, err
	}

	// Append database bytes
	if _, err := payload.Write(db); err != nil {
		return nil, fmt.Errorf("write database: %w", err)
	}

	// Optional GenSystem flags
	if sys.FlagsPresent {
		flags := int32(0)
		if sys.AllowUserDefinedClusterSetup {
			flags |= 0x01
		}
		if sys.EnableForSubArrays {
			flags |= 0x02
		}

		if err := binary.Write(&payload, binary.LittleEndian, flags); err != nil {
			return nil, fmt.Errorf("write gensystem flags: %w", err)
		}
	}

	// Wrap in block header
	return encodeBlock(sys.SubVersion, payload.Bytes()), nil
}

func (e *gllEncoder) encodeDatabase(db *gllbin.Database) ([]byte, error) {
	// Use raw block when available
	if db != nil && len(db.RawBlock) > 0 {
		return db.RawBlock, nil
	}

	// Build minimal empty database payload
	var payload bytes.Buffer

	// Placeholder fields
	if err := binary.Write(&payload, binary.LittleEndian, int32(0)); err != nil {
		return nil, fmt.Errorf("write db field1: %w", err)
	}

	if err := binary.Write(&payload, binary.LittleEndian, int32(0)); err != nil {
		return nil, fmt.Errorf("write db field2: %w", err)
	}

	// DataFiles count
	if err := binary.Write(&payload, binary.LittleEndian, int32(0)); err != nil {
		return nil, fmt.Errorf("write datafiles count: %w", err)
	}

	// BoxTypes buffer
	boxTypesBlock, err := e.encodeBoxTypesBuffer(db)
	if err != nil {
		return nil, fmt.Errorf("encode boxtypes: %w", err)
	}
	if _, err := payload.Write(boxTypesBlock); err != nil {
		return nil, fmt.Errorf("write boxtypes block: %w", err)
	}

	// Frames buffer (empty block)
	if err := binary.Write(&payload, binary.LittleEndian, int32(0)); err != nil {
		return nil, fmt.Errorf("write frames block: %w", err)
	}

	// Connectors buffer (empty block)
	if err := binary.Write(&payload, binary.LittleEndian, int32(0)); err != nil {
		return nil, fmt.Errorf("write connectors block: %w", err)
	}

	// Limits buffer (empty block)
	if err := binary.Write(&payload, binary.LittleEndian, int32(0)); err != nil {
		return nil, fmt.Errorf("write limits block: %w", err)
	}

	// SourceDefinitions
	var sourceItems []gllbin.SourceDefinitionItem
	if db != nil {
		sourceItems = db.SourceDefinitions
	}
	sourceDefs, err := e.encodeSourceDefinitionsBuffer(sourceItems)
	if err != nil {
		return nil, fmt.Errorf("encode source definitions: %w", err)
	}
	if _, err := payload.Write(sourceDefs); err != nil {
		return nil, fmt.Errorf("write source definitions: %w", err)
	}

	// Choose sub-version
	subVersion := int16(3)
	if db != nil && db.SubVersion != 0 {
		subVersion = db.SubVersion
	}

	// Encode as block
	return encodeBlock(subVersion, payload.Bytes()), nil
}

func (e *gllEncoder) encodeBoxTypesBuffer(db *gllbin.Database) ([]byte, error) {
	// Return empty block if no database or no box types
	if db == nil || len(db.BoxTypes) == 0 {
		var buf bytes.Buffer
		_ = binary.Write(&buf, binary.LittleEndian, int32(0))
		return buf.Bytes(), nil
	}

	// Build BoxTypes buffer content
	var content bytes.Buffer

	// Write count
	// nolint:gosec // BoxTypes count is controlled by database structure
	if err := binary.Write(&content, binary.LittleEndian, int32(len(db.BoxTypes))); err != nil {
		return nil, fmt.Errorf("write count: %w", err)
	}

	// Write each BoxType
	for i, box := range db.BoxTypes {
		boxBlock, err := e.encodeBoxType(&box)
		if err != nil {
			return nil, fmt.Errorf("encode boxtype %d (%s): %w", i, box.Key, err)
		}
		if _, err := content.Write(boxBlock); err != nil {
			return nil, fmt.Errorf("write boxtype %d: %w", i, err)
		}
	}

	// Wrap in buffer block (with version check and sub-version)
	return encodeBlock(0, content.Bytes()), nil
}

func (e *gllEncoder) encodeBoxType(box *gllbin.BoxType) ([]byte, error) {
	var content bytes.Buffer

	// Write Label
	if err := writeString(&content, box.Label); err != nil {
		return nil, fmt.Errorf("write label: %w", err)
	}

	// Write Key
	if err := writeString(&content, box.Key); err != nil {
		return nil, fmt.Errorf("write key: %w", err)
	}

	// Write SourceBuffer (source placements)
	sourceBuffer, err := e.encodeSourceBuffer(box.SourcePlacements)
	if err != nil {
		return nil, fmt.Errorf("encode sources: %w", err)
	}
	if _, err := content.Write(sourceBuffer); err != nil {
		return nil, fmt.Errorf("write sources: %w", err)
	}

	// Write InputConfigBuffer
	inputConfigBuffer, err := e.encodeInputConfigBuffer(box.InputConfig)
	if err != nil {
		return nil, fmt.Errorf("encode input config: %w", err)
	}
	if _, err := content.Write(inputConfigBuffer); err != nil {
		return nil, fmt.Errorf("write input config: %w", err)
	}

	// Write CaseGeometry
	geometryBuffer, err := e.encodeCaseGeometry(box.CaseGeometry)
	if err != nil {
		return nil, fmt.Errorf("encode geometry: %w", err)
	}
	if _, err := content.Write(geometryBuffer); err != nil {
		return nil, fmt.Errorf("write geometry: %w", err)
	}

	// Write NextPivot
	if err := writeVector3D(&content, box.NextPivot); err != nil {
		return nil, fmt.Errorf("write next pivot: %w", err)
	}

	// Write ReferencePoint
	if err := writeVector3D(&content, box.ReferencePoint); err != nil {
		return nil, fmt.Errorf("write reference point: %w", err)
	}

	// Write CenterOfMass
	if err := writeVector3D(&content, box.CenterOfMass); err != nil {
		return nil, fmt.Errorf("write center of mass: %w", err)
	}

	// Write Weight
	if err := binary.Write(&content, binary.LittleEndian, box.Weight); err != nil {
		return nil, fmt.Errorf("write weight: %w", err)
	}

	// Determine sub-version based on optional fields
	subVersion := int16(0)
	if box.VerticalOpeningAngle != 0 || box.HorizontalOpeningAngle != 0 {
		subVersion = 1
		// Write VerticalOpeningAngle
		if err := binary.Write(&content, binary.LittleEndian, box.VerticalOpeningAngle); err != nil {
			return nil, fmt.Errorf("write vertical opening angle: %w", err)
		}
		// Write HorizontalOpeningAngle
		if err := binary.Write(&content, binary.LittleEndian, box.HorizontalOpeningAngle); err != nil {
			return nil, fmt.Errorf("write horizontal opening angle: %w", err)
		}
	}

	// Wrap in block with version check and sub-version
	return encodeBlock(subVersion, content.Bytes()), nil
}

func (e *gllEncoder) encodeSourceBuffer(sources []gllbin.BoxSource) ([]byte, error) {
	// Return empty block if no sources
	if len(sources) == 0 {
		var buf bytes.Buffer
		_ = binary.Write(&buf, binary.LittleEndian, int32(0))
		return buf.Bytes(), nil
	}

	// Build source buffer content
	var content bytes.Buffer

	// Write count
	// nolint:gosec // Source count is controlled by BoxType structure
	if err := binary.Write(&content, binary.LittleEndian, int32(len(sources))); err != nil {
		return nil, fmt.Errorf("write count: %w", err)
	}

	// Write each source
	for i, src := range sources {
		srcBlock, err := e.encodeBoxSource(&src)
		if err != nil {
			return nil, fmt.Errorf("encode source %d (%s): %w", i, src.Key, err)
		}
		if _, err := content.Write(srcBlock); err != nil {
			return nil, fmt.Errorf("write source %d: %w", i, err)
		}
	}

	// Wrap in buffer block with version check and sub-version
	return encodeBlock(0, content.Bytes()), nil
}

func (e *gllEncoder) encodeBoxSource(src *gllbin.BoxSource) ([]byte, error) {
	var content bytes.Buffer

	// Write SourceDefKey
	if err := writeString(&content, src.SourceDefKey); err != nil {
		return nil, fmt.Errorf("write source def key: %w", err)
	}

	// Write Position (Vector3D)
	if err := writeVector3D(&content, &src.Position); err != nil {
		return nil, fmt.Errorf("write position: %w", err)
	}

	// Write Angles (Vector3D)
	if err := writeVector3D(&content, &src.Angles); err != nil {
		return nil, fmt.Errorf("write angles: %w", err)
	}

	// Write Label
	if err := writeString(&content, src.Label); err != nil {
		return nil, fmt.Errorf("write label: %w", err)
	}

	// Write Key
	if err := writeString(&content, src.Key); err != nil {
		return nil, fmt.Errorf("write key: %w", err)
	}

	// Wrap in block with version check and sub-version
	return encodeBlock(0, content.Bytes()), nil
}

func (e *gllEncoder) encodeCaseGeometry(geom *gllbin.CaseGeometry) ([]byte, error) {
	// Return empty block if no geometry
	if geom == nil {
		var buf bytes.Buffer
		_ = binary.Write(&buf, binary.LittleEndian, int32(0))
		return buf.Bytes(), nil
	}

	// Build geometry content
	var content bytes.Buffer

	// Write IsSymmetric flag
	isSymmetric := int32(0)
	if geom.IsSymmetric {
		isSymmetric = 1
	}
	if err := binary.Write(&content, binary.LittleEndian, isSymmetric); err != nil {
		return nil, fmt.Errorf("write is_symmetric: %w", err)
	}

	// Write SymmetryAxis
	if err := binary.Write(&content, binary.LittleEndian, geom.SymmetryAxis); err != nil {
		return nil, fmt.Errorf("write symmetry_axis: %w", err)
	}

	// Write VertexBuffer
	vertexBuffer, err := e.encodeVertexBuffer(geom.Vertices)
	if err != nil {
		return nil, fmt.Errorf("encode vertices: %w", err)
	}
	if _, err := content.Write(vertexBuffer); err != nil {
		return nil, fmt.Errorf("write vertices: %w", err)
	}

	// Write EdgeBuffer
	edgeBuffer, err := e.encodeEdgeBuffer(geom.Edges)
	if err != nil {
		return nil, fmt.Errorf("encode edges: %w", err)
	}
	if _, err := content.Write(edgeBuffer); err != nil {
		return nil, fmt.Errorf("write edges: %w", err)
	}

	// Write FaceBuffer (only if sub-version >= 1)
	if geom.SubVersion >= 1 {
		faceBuffer, err := e.encodeFaceBuffer(geom.Faces)
		if err != nil {
			return nil, fmt.Errorf("encode faces: %w", err)
		}
		if _, err := content.Write(faceBuffer); err != nil {
			return nil, fmt.Errorf("write faces: %w", err)
		}
	}

	// Wrap in block with version check and sub-version
	return encodeBlock(geom.SubVersion, content.Bytes()), nil
}

func (e *gllEncoder) encodeVertexBuffer(vertices []gllbin.Vertex) ([]byte, error) {
	// Return empty block if no vertices
	if len(vertices) == 0 {
		var buf bytes.Buffer
		_ = binary.Write(&buf, binary.LittleEndian, int32(0))
		return buf.Bytes(), nil
	}

	// Build vertex buffer content
	var content bytes.Buffer

	// Write count
	// nolint:gosec // Vertex count is controlled by geometry structure
	if err := binary.Write(&content, binary.LittleEndian, int32(len(vertices))); err != nil {
		return nil, fmt.Errorf("write count: %w", err)
	}

	// Write each vertex
	for i, vertex := range vertices {
		vertexBlock, err := e.encodeVertex(&vertex)
		if err != nil {
			return nil, fmt.Errorf("encode vertex %d: %w", i, err)
		}
		if _, err := content.Write(vertexBlock); err != nil {
			return nil, fmt.Errorf("write vertex %d: %w", i, err)
		}
	}

	// Wrap in buffer block with version check and sub-version
	return encodeBlock(0, content.Bytes()), nil
}

func (e *gllEncoder) encodeVertex(vertex *gllbin.Vertex) ([]byte, error) {
	var content bytes.Buffer

	// Write Color
	if err := binary.Write(&content, binary.LittleEndian, vertex.Color); err != nil {
		return nil, fmt.Errorf("write color: %w", err)
	}

	// Write X coordinate
	if err := binary.Write(&content, binary.LittleEndian, vertex.X); err != nil {
		return nil, fmt.Errorf("write x: %w", err)
	}

	// Write Y coordinate
	if err := binary.Write(&content, binary.LittleEndian, vertex.Y); err != nil {
		return nil, fmt.Errorf("write y: %w", err)
	}

	// Write Z coordinate
	if err := binary.Write(&content, binary.LittleEndian, vertex.Z); err != nil {
		return nil, fmt.Errorf("write z: %w", err)
	}

	// Write Label
	if err := writeString(&content, vertex.Label); err != nil {
		return nil, fmt.Errorf("write label: %w", err)
	}

	// Write HasTwin flag
	hasTwin := byte(0)
	if vertex.HasTwin {
		hasTwin = 1
	}
	if err := binary.Write(&content, binary.LittleEndian, hasTwin); err != nil {
		return nil, fmt.Errorf("write has_twin: %w", err)
	}

	// Wrap in block with version check and sub-version
	return encodeBlock(0, content.Bytes()), nil
}

func (e *gllEncoder) encodeEdgeBuffer(edges []gllbin.Edge) ([]byte, error) {
	// Return empty block if no edges
	if len(edges) == 0 {
		var buf bytes.Buffer
		_ = binary.Write(&buf, binary.LittleEndian, int32(0))
		return buf.Bytes(), nil
	}

	// Build edge buffer content
	var content bytes.Buffer

	// Write count
	// nolint:gosec // Edge count is controlled by geometry structure
	if err := binary.Write(&content, binary.LittleEndian, int32(len(edges))); err != nil {
		return nil, fmt.Errorf("write count: %w", err)
	}

	// Write each edge
	for i, edge := range edges {
		edgeBlock, err := e.encodeEdge(&edge)
		if err != nil {
			return nil, fmt.Errorf("encode edge %d: %w", i, err)
		}
		if _, err := content.Write(edgeBlock); err != nil {
			return nil, fmt.Errorf("write edge %d: %w", i, err)
		}
	}

	// Wrap in buffer block with version check and sub-version
	return encodeBlock(0, content.Bytes()), nil
}

func (e *gllEncoder) encodeEdge(edge *gllbin.Edge) ([]byte, error) {
	var content bytes.Buffer

	// Write Color
	if err := binary.Write(&content, binary.LittleEndian, edge.Color); err != nil {
		return nil, fmt.Errorf("write color: %w", err)
	}

	// Write V1 (first vertex index)
	if err := binary.Write(&content, binary.LittleEndian, edge.V1); err != nil {
		return nil, fmt.Errorf("write v1: %w", err)
	}

	// Write V2 (second vertex index)
	if err := binary.Write(&content, binary.LittleEndian, edge.V2); err != nil {
		return nil, fmt.Errorf("write v2: %w", err)
	}

	// Write Label
	if err := writeString(&content, edge.Label); err != nil {
		return nil, fmt.Errorf("write label: %w", err)
	}

	// Write HasTwin flag
	hasTwin := byte(0)
	if edge.HasTwin {
		hasTwin = 1
	}
	if err := binary.Write(&content, binary.LittleEndian, hasTwin); err != nil {
		return nil, fmt.Errorf("write has_twin: %w", err)
	}

	// Wrap in block with version check and sub-version
	return encodeBlock(0, content.Bytes()), nil
}

func (e *gllEncoder) encodeFaceBuffer(faces []gllbin.Face) ([]byte, error) {
	// Return empty block if no faces
	if len(faces) == 0 {
		var buf bytes.Buffer
		_ = binary.Write(&buf, binary.LittleEndian, int32(0))
		return buf.Bytes(), nil
	}

	// Build face buffer content
	var content bytes.Buffer

	// Write count
	// nolint:gosec // Face count is controlled by geometry structure
	if err := binary.Write(&content, binary.LittleEndian, int32(len(faces))); err != nil {
		return nil, fmt.Errorf("write count: %w", err)
	}

	// Write each face
	for i, face := range faces {
		faceBlock, err := e.encodeFace(&face)
		if err != nil {
			return nil, fmt.Errorf("encode face %d: %w", i, err)
		}
		if _, err := content.Write(faceBlock); err != nil {
			return nil, fmt.Errorf("write face %d: %w", i, err)
		}
	}

	// Wrap in buffer block with version check and sub-version
	return encodeBlock(0, content.Bytes()), nil
}

func (e *gllEncoder) encodeFace(face *gllbin.Face) ([]byte, error) {
	var content bytes.Buffer

	// Write HasTwin flag
	hasTwin := byte(0)
	if face.HasTwin {
		hasTwin = 1
	}
	if err := binary.Write(&content, binary.LittleEndian, hasTwin); err != nil {
		return nil, fmt.Errorf("write has_twin: %w", err)
	}

	// Write VertexCount
	// nolint:gosec // Vertex count is controlled by face structure
	vertexCount := int32(len(face.Vertices))
	if err := binary.Write(&content, binary.LittleEndian, vertexCount); err != nil {
		return nil, fmt.Errorf("write vertex_count: %w", err)
	}

	// Write Vertex indices
	for i, idx := range face.Vertices {
		if err := binary.Write(&content, binary.LittleEndian, idx); err != nil {
			return nil, fmt.Errorf("write vertex[%d]: %w", i, err)
		}
	}

	// Write Color
	if err := binary.Write(&content, binary.LittleEndian, face.Color); err != nil {
		return nil, fmt.Errorf("write color: %w", err)
	}

	// Write Label
	if err := writeString(&content, face.Label); err != nil {
		return nil, fmt.Errorf("write label: %w", err)
	}

	// Wrap in block with version check and sub-version
	return encodeBlock(0, content.Bytes()), nil
}

func (e *gllEncoder) encodeInputConfigBuffer(inputConfig *gllbin.BoxInputConfig) ([]byte, error) {
	// Return empty block if no input config
	if inputConfig == nil {
		var buf bytes.Buffer
		_ = binary.Write(&buf, binary.LittleEndian, int32(0))
		return buf.Bytes(), nil
	}

	// Encode the BoxInputConfig
	configBlock, err := e.encodeBoxInputConfig(inputConfig)
	if err != nil {
		return nil, fmt.Errorf("encode box input config: %w", err)
	}

	// The buffer contains the single BoxInputConfig block directly
	// (not wrapped in a count+items structure like other buffers)
	return configBlock, nil
}

func (e *gllEncoder) encodeBoxInputConfig(config *gllbin.BoxInputConfig) ([]byte, error) {
	var content bytes.Buffer

	// Write Label
	if err := writeString(&content, config.Label); err != nil {
		return nil, fmt.Errorf("write label: %w", err)
	}

	// Write Key
	if err := writeString(&content, config.Key); err != nil {
		return nil, fmt.Errorf("write key: %w", err)
	}

	// Write BoxInputBuffer (inputs)
	inputBuffer, err := e.encodeBoxInputBuffer(config.Inputs)
	if err != nil {
		return nil, fmt.Errorf("encode inputs: %w", err)
	}
	if _, err := content.Write(inputBuffer); err != nil {
		return nil, fmt.Errorf("write inputs: %w", err)
	}

	// Wrap in block with version check and sub-version
	return encodeBlock(0, content.Bytes()), nil
}

func (e *gllEncoder) encodeBoxInputBuffer(inputs []gllbin.BoxInput) ([]byte, error) {
	// Return empty block if no inputs
	if len(inputs) == 0 {
		var buf bytes.Buffer
		_ = binary.Write(&buf, binary.LittleEndian, int32(0))
		return buf.Bytes(), nil
	}

	// Build input buffer content
	var content bytes.Buffer

	// Write count
	// nolint:gosec // Input count is controlled by BoxInputConfig structure
	if err := binary.Write(&content, binary.LittleEndian, int32(len(inputs))); err != nil {
		return nil, fmt.Errorf("write count: %w", err)
	}

	// Write each input
	for i, input := range inputs {
		inputBlock, err := e.encodeBoxInput(&input)
		if err != nil {
			return nil, fmt.Errorf("encode input %d (%s): %w", i, input.Label, err)
		}
		if _, err := content.Write(inputBlock); err != nil {
			return nil, fmt.Errorf("write input %d: %w", i, err)
		}
	}

	// Wrap in buffer block with version check and sub-version
	return encodeBlock(0, content.Bytes()), nil
}

func (e *gllEncoder) encodeBoxInput(input *gllbin.BoxInput) ([]byte, error) {
	var content bytes.Buffer

	// Write Label
	if err := writeString(&content, input.Label); err != nil {
		return nil, fmt.Errorf("write label: %w", err)
	}

	// Write LinkCount
	// nolint:gosec // Link count is controlled by BoxInput structure
	linkCount := int32(len(input.SourceLinks))
	if err := binary.Write(&content, binary.LittleEndian, linkCount); err != nil {
		return nil, fmt.Errorf("write link count: %w", err)
	}

	// Write SourceFilterLinks (inline, no block wrappers)
	for i, link := range input.SourceLinks {
		if err := e.encodeSourceFilterLink(&content, &link); err != nil {
			return nil, fmt.Errorf("write link %d: %w", i, err)
		}
	}

	// Determine sub-version based on optional fields
	subVersion := int16(0)
	if input.RatedImpedance != 0 {
		subVersion = 1
		// Write RatedImpedance
		if err := binary.Write(&content, binary.LittleEndian, input.RatedImpedance); err != nil {
			return nil, fmt.Errorf("write rated impedance: %w", err)
		}
	}

	// Wrap in block with version check and sub-version
	return encodeBlock(subVersion, content.Bytes()), nil
}

func (e *gllEncoder) encodeSourceFilterLink(w io.Writer, link *gllbin.SourceFilterLink) error {
	// Write SourceKey
	if err := writeString(w, link.SourceKey); err != nil {
		return fmt.Errorf("write source key: %w", err)
	}

	// Write FilterGrpKey
	if err := writeString(w, link.FilterGrpKey); err != nil {
		return fmt.Errorf("write filter grp key: %w", err)
	}

	return nil
}

func writeVector3D(w io.Writer, vec *gllbin.Vector3D) error {
	// Write zero vector if nil
	if vec == nil {
		vec = &gllbin.Vector3D{}
	}

	if err := binary.Write(w, binary.LittleEndian, vec.X); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, vec.Y); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, vec.Z); err != nil {
		return err
	}

	return nil
}

func encodeBlock(subVersion int16, content []byte) []byte {
	// Build block with size and version headers
	var buf bytes.Buffer

	// Size includes header fields
	blockSize := int32(len(content) + 4 + 2 + 2) // nolint:gosec
	_ = binary.Write(&buf, binary.LittleEndian, blockSize)
	_ = binary.Write(&buf, binary.LittleEndian, int16(0))
	_ = binary.Write(&buf, binary.LittleEndian, subVersion)
	_, _ = buf.Write(content)

	// Return encoded block
	return buf.Bytes()
}

func writeString(w io.Writer, value string) error {
	// Encode length-prefixed string
	data := []byte(value)
	if len(data) > 0xFFFF {
		return fmt.Errorf("string too long: %d", len(data))
	}

	if err := binary.Write(w, binary.LittleEndian, uint16(len(data))); err != nil { // nolint:gosec // checked above
		return err
	}

	if len(data) == 0 {
		return nil
	}

	// Write raw bytes
	_, err := w.Write(data)

	return err
}
