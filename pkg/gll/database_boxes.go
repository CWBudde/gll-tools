package gll

import (
	"fmt"
	"io"

	"github.com/cwbudde/gll-tools/internal/gll"
)

// DataFile represents an embedded file in the GLL
type DataFile struct {
	Key      string `json:"key"`
	Filename string `json:"filename"`
	Size     int32  `json:"size"`
	Offset   int64  `json:"offset"` // Offset in file for lazy loading
	Data     []byte `json:"-"`      // Actual data (loaded on demand)
}

// BoxType represents a speaker cabinet type
type BoxType struct {
	Label                  string        `json:"label"`
	Key                    string        `json:"key"`
	Sources                []string      `json:"sources,omitempty"` // Keys of acoustic sources
	SourcePlacements       []BoxSource   `json:"source_placements,omitempty"`
	CaseGeometry           *CaseGeometry `json:"case_geometry,omitempty"`
	NextPivot              *Vector3D     `json:"next_pivot,omitempty"`
	ReferencePoint         *Vector3D     `json:"reference_point,omitempty"`
	CenterOfMass           *Vector3D     `json:"center_of_mass,omitempty"`
	Weight                 float64       `json:"weight,omitempty"`                   // kg
	VerticalOpeningAngle   float64       `json:"vertical_opening_angle,omitempty"`   // degrees
	HorizontalOpeningAngle float64       `json:"horizontal_opening_angle,omitempty"` // degrees
}

// IncludeFile represents an additional data file embedded in the GLL
type IncludeFile struct {
	Label    string `json:"label"`
	Key      string `json:"key"`
	Filename string `json:"filename"`
	Size     int32  `json:"size"`
	Offset   int64  `json:"offset"` // Offset in file for lazy loading
	Data     []byte `json:"-"`      // Actual data (loaded on demand)
}

// BoxSource represents a source placement inside a box type.
type BoxSource struct {
	Label        string   `json:"label,omitempty"`
	Key          string   `json:"key,omitempty"`
	Position     Vector3D `json:"position"`
	Angles       Vector3D `json:"angles"` // H, V, R rotation angles
	SourceDefKey string   `json:"source_def_key,omitempty"`
}

// CaseGeometry represents 3D cabinet/frame geometry (mesh data)
// Binary format: block_size + vcheck(0) + sver + IsSymmetric(int32) + SymmetryAxis(double) + VertexBuffer + EdgeBuffer + [FaceBuffer if sver>=1]
type CaseGeometry struct {
	IsSymmetric  bool     `json:"is_symmetric"`            // Planar symmetry on X axis
	SymmetryAxis float64  `json:"symmetry_axis,omitempty"` // X coordinate of symmetry plane
	Vertices     []Vertex `json:"vertices,omitempty"`      // 3D vertex positions
	Edges        []Edge   `json:"edges,omitempty"`         // Vertex index pairs
	Faces        []Face   `json:"faces,omitempty"`         // Face definitions (sver >= 1)
	SubVersion   int16    `json:"sub_version"`
}

// Vertex represents a 3D point in the geometry
// Binary format: block_size + vcheck(0) + sver + Color(int32) + X,Y,Z(3 doubles) + Label(string) + HasTwin(byte)
type Vertex struct {
	Color   int32   `json:"color,omitempty"` // RGB color for rendering
	X       float64 `json:"x"`               // X coordinate (mm) - stored as double
	Y       float64 `json:"y"`               // Y coordinate (mm) - stored as double
	Z       float64 `json:"z"`               // Z coordinate (mm) - stored as double
	Label   string  `json:"label,omitempty"` // Optional vertex label
	HasTwin bool    `json:"has_twin"`        // True if vertex has symmetric twin
}

// Edge represents a line segment between two vertices
// Binary format: block_size + vcheck(0) + sver + Color(int32) + V1(int32) + V2(int32) + Label(string) + HasTwin(byte)
// Note: Negative vertex indices refer to mirrored vertices when symmetry is enabled
type Edge struct {
	Color   int32  `json:"color,omitempty"` // RGB color for rendering
	V1      int32  `json:"v1"`              // First vertex index (negative = mirrored)
	V2      int32  `json:"v2"`              // Second vertex index (negative = mirrored)
	Label   string `json:"label,omitempty"` // Optional edge label
	HasTwin bool   `json:"has_twin"`        // True if edge has symmetric twin
}

// Face represents a polygon face (list of vertex indices)
// Binary format: block_size + vcheck(0) + sver + HasTwin(byte) + Count(int32) + Vertices(int32[]) + Color(int32) + Label(string)
// Note: Negative vertex indices refer to mirrored vertices when symmetry is enabled
type Face struct {
	HasTwin  bool    `json:"has_twin"`        // True if face has symmetric twin
	Vertices []int32 `json:"vertices"`        // Vertex indices forming the face
	Color    int32   `json:"color,omitempty"` // RGB color for rendering
	Label    string  `json:"label,omitempty"` // Optional face label
}

func parseDataFileBuffer(br *gll.ByteReader, maxOffset int64) ([]DataFile, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading data file count: %w", err)
	}

	if count <= 0 {
		return nil, nil
	}

	files := make([]DataFile, 0, count)

	for i := int32(0); i < count; i++ {
		if br.Offset() >= maxOffset {
			break
		}

		df, err := parseDataFile(br)
		if err != nil {
			return files, fmt.Errorf("parsing data file %d: %w", i, err)
		}

		files = append(files, *df)
	}

	return files, nil
}

func parseDataFile(br *gll.ByteReader) (*DataFile, error) {
	df := &DataFile{}

	// Read entry size (total size of this DataFile entry)
	entrySize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading entry size: %w", err)
	}

	startOffset := br.Offset()
	endOffset := startOffset + int64(entrySize) - 4 // -4 because entrySize includes itself

	// Read unknown field (always 0?)
	_, err = br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading unknown field: %w", err)
	}

	// Read Filename (this serves as both key and filename)
	filename, err := br.ReadString()
	if err != nil {
		return nil, fmt.Errorf("reading filename: %w", err)
	}

	df.Filename = filename
	df.Key = filename // Use filename as key

	// Read data size
	size, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading data size: %w", err)
	}

	df.Size = size

	// Record offset for lazy loading
	df.Offset = br.Offset()

	// Skip to end of entry (using entry size for safety)
	_, err = br.Seek(endOffset, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("seeking to entry end: %w", err)
	}

	return df, nil
}

// parseSourceBuffer parses a buffer of box sources (placements).
func parseSourceBuffer(br *gll.ByteReader, maxOffset int64) ([]BoxSource, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	// Buffer has block wrapper
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading buffer block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, nil
	}

	bufferEnd := br.Offset() + int64(blockSize) - 4
	if bufferEnd > maxOffset {
		bufferEnd = maxOffset
	}
	defer func() {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
	}()

	// Read version check and sub-version
	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		return nil, fmt.Errorf("unsupported buffer version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading count: %w", err)
	}

	if count <= 0 {
		return nil, nil
	}

	sources := make([]BoxSource, 0, count)

	for i := int32(0); i < count; i++ {
		if br.Offset() >= bufferEnd {
			break
		}

		source, err := parseBoxSource(br)
		if err != nil {
			break
		}

		sources = append(sources, *source)
	}

	return sources, nil
}

// parseBoxSource parses a single box source placement.
func parseBoxSource(br *gll.ByteReader) (*BoxSource, error) {
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid block size: %d", blockSize)
	}

	endOffset := br.Offset() + int64(blockSize) - 4

	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	source := &BoxSource{}

	source.Label, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading label: %w", err)
	}

	source.Key, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading key: %w", err)
	}

	source.Position.X, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading position X: %w", err)
	}

	source.Position.Y, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading position Y: %w", err)
	}

	source.Position.Z, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading position Z: %w", err)
	}

	source.Angles.X, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading angle H: %w", err)
	}

	source.Angles.Y, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading angle V: %w", err)
	}

	source.Angles.Z, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading angle R: %w", err)
	}

	source.SourceDefKey, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading source definition key: %w", err)
	}

	_, _ = br.Seek(endOffset, io.SeekStart)
	return source, nil
}

func parseBoxTypeBuffer(br *gll.ByteReader, maxOffset int64) ([]BoxType, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	// BoxTypes buffer has a block wrapper
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading buffer block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, nil
	}

	bufferEnd := br.Offset() + int64(blockSize) - 4

	// Read version check and sub-version
	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, fmt.Errorf("unsupported buffer version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	// Now read count
	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading box type count: %w", err)
	}

	if count <= 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, nil
	}

	boxes := make([]BoxType, 0, count)

	for i := int32(0); i < count; i++ {
		if br.Offset() >= bufferEnd {
			break
		}

		box, err := parseBoxType(br, bufferEnd)
		if err != nil {
			// Non-fatal, skip remaining
			break
		}

		boxes = append(boxes, *box)
	}

	// Seek to end of buffer
	_, _ = br.Seek(bufferEnd, io.SeekStart)

	return boxes, nil
}

func parseBoxType(br *gll.ByteReader, maxOffset int64) (*BoxType, error) {
	if br.Offset() >= maxOffset {
		return nil, fmt.Errorf("offset %d exceeds maxOffset %d", br.Offset(), maxOffset)
	}

	// Read block size
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid block size: %d", blockSize)
	}

	startOffset := br.Offset()
	endOffset := startOffset + int64(blockSize) - 4

	// Ensure we don't read beyond maxOffset
	if endOffset > maxOffset {
		endOffset = maxOffset
	}

	// Read version check
	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported version: %d", versionCheck)
	}

	// Read sub-version
	subVersion, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	box := &BoxType{}

	// Read Label
	box.Label, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return box, nil
	}

	// Read Key
	box.Key, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return box, nil
	}

	// Parse SourceBuffer (source placements)
	sourcePlacements, err := parseSourceBuffer(br, endOffset)
	if err == nil && len(sourcePlacements) > 0 {
		box.SourcePlacements = sourcePlacements
		box.Sources = make([]string, 0, len(sourcePlacements))
		for _, src := range sourcePlacements {
			if src.Key != "" {
				box.Sources = append(box.Sources, src.Key)
			}
		}
	}

	// Skip InputConfigBuffer
	if err := skipBlockBuffer(br, endOffset); err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return box, nil
	}

	// Parse CaseGeometry (3D mesh data)
	caseGeometry, err := parseCaseGeometry(br, endOffset)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return box, nil
	}
	box.CaseGeometry = caseGeometry

	// Read NextPivot (Vector3D)
	nextPivot, err := parseVector3D(br)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return box, nil
	}
	box.NextPivot = nextPivot

	// Read ReferencePoint (Vector3D)
	refPoint, err := parseVector3D(br)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return box, nil
	}
	box.ReferencePoint = refPoint

	// Read CenterOfMass (Vector3D)
	centerOfMass, err := parseVector3D(br)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return box, nil
	}
	box.CenterOfMass = centerOfMass

	// Read Weight
	box.Weight, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return box, nil
	}

	if subVersion >= 1 {
		// Read VerticalOpeningAngle
		box.VerticalOpeningAngle, err = br.ReadDouble()
		if err != nil {
			_, _ = br.Seek(endOffset, io.SeekStart)
			return box, nil
		}

		// Read HorizontalOpeningAngle
		box.HorizontalOpeningAngle, err = br.ReadDouble()
		if err != nil {
			_, _ = br.Seek(endOffset, io.SeekStart)
			return box, nil
		}
	}

	_, _ = br.Seek(endOffset, io.SeekStart)

	return box, nil
}

// parseIncludeFileBuffer parses the IncludeFiles buffer
func parseIncludeFileBuffer(br *gll.ByteReader, maxOffset int64) ([]IncludeFile, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	// Buffer has block wrapper
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading buffer block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, nil
	}

	bufferEnd := br.Offset() + int64(blockSize) - 4
	if bufferEnd > maxOffset {
		bufferEnd = maxOffset
	}

	// Read version check and sub-version
	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, fmt.Errorf("unsupported buffer version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	// Read count
	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading count: %w", err)
	}

	if count <= 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, nil
	}

	files := make([]IncludeFile, 0, count)

	for i := int32(0); i < count; i++ {
		if br.Offset() >= bufferEnd {
			break
		}

		file, err := parseIncludeFile(br)
		if err != nil {
			break
		}

		files = append(files, *file)
	}

	_, _ = br.Seek(bufferEnd, io.SeekStart)

	return files, nil
}

func parseIncludeFile(br *gll.ByteReader) (*IncludeFile, error) {
	// Read block size
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid block size: %d", blockSize)
	}

	endOffset := br.Offset() + int64(blockSize) - 4

	// Read version check
	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported version: %d", versionCheck)
	}

	// Read sub-version
	_, err = br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	file := &IncludeFile{}

	// Read Label
	file.Label, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading label: %w", err)
	}

	// Read Key
	file.Key, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading key: %w", err)
	}

	// Read Filename
	file.Filename, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading filename: %w", err)
	}

	// Read data size
	size, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading data size: %w", err)
	}

	file.Size = size

	// Record offset for lazy loading
	file.Offset = br.Offset()

	_, _ = br.Seek(endOffset, io.SeekStart)

	return file, nil
}

// parseDataFileBufferWithWrapper parses a DataFile buffer that has a block wrapper
// This is used for AuthorFiles which has the same format as DataFileBuffer
func parseDataFileBufferWithWrapper(br *gll.ByteReader, maxOffset int64) ([]DataFile, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	// Buffer has block wrapper
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading buffer block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, nil
	}

	bufferEnd := br.Offset() + int64(blockSize) - 4
	if bufferEnd > maxOffset {
		bufferEnd = maxOffset
	}

	// Read version check and sub-version
	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, fmt.Errorf("unsupported buffer version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	// Read count
	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading count: %w", err)
	}

	if count <= 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, nil
	}

	files := make([]DataFile, 0, count)

	for i := int32(0); i < count; i++ {
		if br.Offset() >= bufferEnd {
			break
		}

		df, err := parseDataFile(br)
		if err != nil {
			break
		}

		files = append(files, *df)
	}

	_, _ = br.Seek(bufferEnd, io.SeekStart)

	return files, nil
}

// parseCaseGeometry parses a CaseGeometry block (3D mesh data)
// Structure: block_size + vcheck(0) + sver + IsSymmetric(int32) + [SymmetryAxis(double)] + VertexBuffer + EdgeBuffer + [FaceBuffer if sver>=1]
func parseCaseGeometry(br *gll.ByteReader, maxOffset int64) (*CaseGeometry, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading geometry block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid geometry block size: %d", blockSize)
	}

	startOffset := br.Offset()
	endOffset := startOffset + int64(blockSize) - 4
	if endOffset > maxOffset {
		endOffset = maxOffset
	}

	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading geometry version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported geometry version: %d", versionCheck)
	}

	subVersion, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading geometry sub-version: %w", err)
	}

	geometry := &CaseGeometry{SubVersion: subVersion}

	isSymmetric, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading geometry symmetry flag: %w", err)
	}
	geometry.IsSymmetric = isSymmetric != 0

	geometry.SymmetryAxis, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading geometry symmetry axis: %w", err)
	}

	vertices, err := parseVertexBuffer(br, endOffset)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, err
	}
	geometry.Vertices = vertices

	edges, err := parseEdgeBuffer(br, endOffset)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, err
	}
	geometry.Edges = edges

	if subVersion >= 1 {
		faces, err := parseFaceBuffer(br, endOffset)
		if err != nil {
			_, _ = br.Seek(endOffset, io.SeekStart)
			return nil, err
		}
		geometry.Faces = faces
	}

	_, _ = br.Seek(endOffset, io.SeekStart)

	return geometry, nil
}

func parseVertexBuffer(br *gll.ByteReader, maxOffset int64) ([]Vertex, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading vertex buffer size: %w", err)
	}

	if blockSize <= 0 {
		return nil, nil
	}

	bufferEnd := br.Offset() + int64(blockSize) - 4
	if bufferEnd > maxOffset {
		bufferEnd = maxOffset
	}

	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading vertex buffer version check: %w", err)
	}
	if versionCheck != 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, fmt.Errorf("unsupported vertex buffer version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading vertex buffer sub-version: %w", err)
	}

	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading vertex count: %w", err)
	}

	if count <= 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, nil
	}

	vertices := make([]Vertex, 0, count)
	for i := int32(0); i < count; i++ {
		if br.Offset() >= bufferEnd {
			break
		}

		vertex, err := parseVertex(br)
		if err != nil {
			break
		}

		vertices = append(vertices, *vertex)
	}

	_, _ = br.Seek(bufferEnd, io.SeekStart)

	return vertices, nil
}

func parseVertex(br *gll.ByteReader) (*Vertex, error) {
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading vertex block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid vertex block size: %d", blockSize)
	}

	endOffset := br.Offset() + int64(blockSize) - 4

	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading vertex version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported vertex version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading vertex sub-version: %w", err)
	}

	vertex := &Vertex{}

	vertex.Color, err = br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading vertex color: %w", err)
	}

	vec, err := parseVector3D(br)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading vertex coordinates: %w", err)
	}
	vertex.X, vertex.Y, vertex.Z = vec.X, vec.Y, vec.Z

	vertex.Label, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading vertex label: %w", err)
	}

	hasTwin, err := br.ReadByte()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading vertex twin flag: %w", err)
	}
	vertex.HasTwin = hasTwin != 0

	_, _ = br.Seek(endOffset, io.SeekStart)

	return vertex, nil
}

func parseEdgeBuffer(br *gll.ByteReader, maxOffset int64) ([]Edge, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading edge buffer size: %w", err)
	}

	if blockSize <= 0 {
		return nil, nil
	}

	bufferEnd := br.Offset() + int64(blockSize) - 4
	if bufferEnd > maxOffset {
		bufferEnd = maxOffset
	}

	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading edge buffer version check: %w", err)
	}
	if versionCheck != 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, fmt.Errorf("unsupported edge buffer version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading edge buffer sub-version: %w", err)
	}

	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading edge count: %w", err)
	}

	if count <= 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, nil
	}

	edges := make([]Edge, 0, count)
	for i := int32(0); i < count; i++ {
		if br.Offset() >= bufferEnd {
			break
		}

		edge, err := parseEdge(br)
		if err != nil {
			break
		}

		edges = append(edges, *edge)
	}

	_, _ = br.Seek(bufferEnd, io.SeekStart)

	return edges, nil
}

func parseEdge(br *gll.ByteReader) (*Edge, error) {
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading edge block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid edge block size: %d", blockSize)
	}

	endOffset := br.Offset() + int64(blockSize) - 4

	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading edge version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported edge version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading edge sub-version: %w", err)
	}

	edge := &Edge{}

	edge.Color, err = br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading edge color: %w", err)
	}

	edge.V1, err = br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading edge vertex 1: %w", err)
	}

	edge.V2, err = br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading edge vertex 2: %w", err)
	}

	edge.Label, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading edge label: %w", err)
	}

	hasTwin, err := br.ReadByte()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading edge twin flag: %w", err)
	}
	edge.HasTwin = hasTwin != 0

	_, _ = br.Seek(endOffset, io.SeekStart)

	return edge, nil
}

func parseFaceBuffer(br *gll.ByteReader, maxOffset int64) ([]Face, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading face buffer size: %w", err)
	}

	if blockSize <= 0 {
		return nil, nil
	}

	bufferEnd := br.Offset() + int64(blockSize) - 4
	if bufferEnd > maxOffset {
		bufferEnd = maxOffset
	}

	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading face buffer version check: %w", err)
	}
	if versionCheck != 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, fmt.Errorf("unsupported face buffer version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading face buffer sub-version: %w", err)
	}

	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading face count: %w", err)
	}

	if count <= 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, nil
	}

	faces := make([]Face, 0, count)
	for i := int32(0); i < count; i++ {
		if br.Offset() >= bufferEnd {
			break
		}

		face, err := parseFace(br)
		if err != nil {
			break
		}

		faces = append(faces, *face)
	}

	_, _ = br.Seek(bufferEnd, io.SeekStart)

	return faces, nil
}

func parseFace(br *gll.ByteReader) (*Face, error) {
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading face block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid face block size: %d", blockSize)
	}

	endOffset := br.Offset() + int64(blockSize) - 4

	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading face version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported face version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading face sub-version: %w", err)
	}

	face := &Face{}

	hasTwin, err := br.ReadByte()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading face twin flag: %w", err)
	}
	face.HasTwin = hasTwin != 0

	count, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading face vertex count: %w", err)
	}
	if count < 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("invalid face vertex count: %d", count)
	}

	face.Vertices = make([]int32, count)
	for i := int32(0); i < count; i++ {
		v, err := br.ReadInt32()
		if err != nil {
			_, _ = br.Seek(endOffset, io.SeekStart)
			return nil, fmt.Errorf("reading face vertex index %d: %w", i, err)
		}
		face.Vertices[i] = v
	}

	face.Color, err = br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading face color: %w", err)
	}

	face.Label, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading face label: %w", err)
	}

	_, _ = br.Seek(endOffset, io.SeekStart)

	return face, nil
}

// skipBlockBuffer skips a buffer by reading its block size and seeking past it
func skipBlockBuffer(br *gll.ByteReader, maxOffset int64) error {
	if br.Offset() >= maxOffset {
		return nil
	}

	blockSize, err := br.ReadInt32()
	if err != nil {
		return err
	}

	if blockSize <= 4 {
		return nil
	}

	endOffset := br.Offset() + int64(blockSize) - 4
	if endOffset > maxOffset {
		endOffset = maxOffset
	}

	_, err = br.Seek(endOffset, io.SeekStart)
	return err
}
