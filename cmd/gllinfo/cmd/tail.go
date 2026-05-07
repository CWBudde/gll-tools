package cmd

import (
	"bytes"
	"compress/zlib"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/cwbudde/gll-tools/pkg/gll"
	"github.com/spf13/cobra"
)

var tailCmd = &cobra.Command{
	Use:   "tail <file.gll>",
	Short: "Inspect trailing bytes after the GenSystem block",
	Long: `Inspect trailing bytes after the GenSystem block and scan for known signatures
(PNG, zlib, PDF, ZIP).`,
	Args: cobra.ExactArgs(1),
	RunE: runTail,
}

func init() {
	rootCmd.AddCommand(tailCmd)
}

func runTail(cmd *cobra.Command, args []string) error {
	filename := args[0]

	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	file, err := gll.Parse(f)
	if err != nil {
		return fmt.Errorf("parsing GLL file: %w", err)
	}

	tail := file.RawTail
	if len(tail) == 0 {
		fmt.Printf("Tail: 0 bytes (no trailing data)\n")
		return nil
	}

	tailStart := info.Size() - int64(len(tail))
	fmt.Printf("Tail: %d bytes (offset %d)\n", len(tail), tailStart)

	printTailStats(tail)
	printHexDump(tail)
	printLengthPrefixedStrings(tail, 4, 200)
	printTailRecords(tail, 4, 200, 64)
	printTailPresets(file)

	matches := scanTailSignatures(tail)
	if len(matches) == 0 {
		fmt.Println("Signatures: none found")
	} else {
		fmt.Println("Signatures:")
		for _, m := range matches {
			fmt.Printf("  %s at +%d (abs %d)\n", m.kind, m.offset, tailStart+int64(m.offset))
			dumpHexWindow(tail, m.offset, 64)
			if isZlibSignature(m.kind) {
				checkZlib(tail[m.offset:], 5<<20)
			}
		}
	}

	return nil
}

type tailMatch struct {
	kind   string
	offset int
}

func scanTailSignatures(data []byte) []tailMatch {
	type sig struct {
		kind string
		data []byte
	}

	sigs := []sig{
		{kind: "PNG", data: []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}},
		{kind: "ZLIB(78 9C)", data: []byte{0x78, 0x9C}},
		{kind: "ZLIB(78 5E)", data: []byte{0x78, 0x5E}},
		{kind: "ZLIB(78 DA)", data: []byte{0x78, 0xDA}},
		{kind: "PDF", data: []byte("%PDF")},
		{kind: "ZIP", data: []byte{'P', 'K', 0x03, 0x04}},
	}

	var out []tailMatch
	for _, s := range sigs {
		offset := 0
		for {
			idx := bytes.Index(data[offset:], s.data)
			if idx < 0 {
				break
			}
			pos := offset + idx
			out = append(out, tailMatch{kind: s.kind, offset: pos})
			offset = pos + 1
		}
	}

	return out
}

func dumpHexWindow(data []byte, offset int, radius int) {
	start := offset - radius
	if start < 0 {
		start = 0
	}
	end := offset + radius
	if end > len(data) {
		end = len(data)
	}

	snippet := data[start:end]
	fmt.Printf("    hex[%d:%d]: %s\n", start, end, hex.EncodeToString(snippet))
}

func isZlibSignature(kind string) bool {
	return kind == "ZLIB(78 9C)" || kind == "ZLIB(78 5E)" || kind == "ZLIB(78 DA)"
}

func checkZlib(data []byte, maxBytes int64) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		fmt.Printf("    zlib: invalid (%v)\n", err)
		return
	}
	defer reader.Close()

	limited := io.LimitReader(reader, maxBytes+1)
	decompressed, err := io.ReadAll(limited)
	if err != nil {
		fmt.Printf("    zlib: read error (%v)\n", err)
		return
	}

	if int64(len(decompressed)) > maxBytes {
		fmt.Printf("    zlib: >%d bytes (truncated)\n", maxBytes)
		return
	}

	preview := decompressed
	if len(preview) > 16 {
		preview = preview[:16]
	}

	fmt.Printf("    zlib: ok (%d bytes) head=%s\n", len(decompressed), hex.EncodeToString(preview))
}

func printTailStats(data []byte) {
	entropy := shannonEntropy(data)
	printable, total := countPrintable(data)
	fmt.Printf("Entropy: %.3f bits/byte\n", entropy)
	if total > 0 {
		fmt.Printf("ASCII printable: %d/%d (%.1f%%)\n", printable, total, 100.0*float64(printable)/float64(total))
	}
	printASCIIRuns(data, 16)
	printRepeats(data, 16)
}

func printHexDump(data []byte) {
	fmt.Println("Hex dump:")
	const cols = 16
	for i := 0; i < len(data); i += cols {
		end := i + cols
		if end > len(data) {
			end = len(data)
		}
		line := data[i:end]
		fmt.Printf("  %04x  ", i)
		for j := 0; j < cols; j++ {
			if i+j < len(data) {
				fmt.Printf("%02x ", data[i+j])
			} else {
				fmt.Print("   ")
			}
		}
		fmt.Print(" ")
		for _, b := range line {
			if b >= 0x20 && b <= 0x7E {
				fmt.Printf("%c", b)
			} else {
				fmt.Print(".")
			}
		}
		fmt.Println()
	}
}

func printLengthPrefixedStrings(data []byte, minLen, maxLen int) {
	hits := gll.ParseTailStrings(data, minLen, maxLen)
	if len(hits) == 0 {
		fmt.Println("Length-prefixed strings: none")
		return
	}
	fmt.Println("Length-prefixed strings:")
	for _, h := range hits {
		fmt.Printf("  +%d: %q\n", h.Offset, h.Value)
	}
}

func printTailRecords(data []byte, minLen, maxLen, gap int) {
	stringsFound := gll.ParseTailStrings(data, minLen, maxLen)
	records := gll.GroupTailStrings(stringsFound, gap)
	if len(records) == 0 {
		fmt.Println("Tail records: none")
		return
	}
	fmt.Println("Tail records:")
	for i, rec := range records {
		label := rec.Label
		if label == "" {
			label = "unknown"
		}
		fmt.Printf("  #%d +%d..+%d (%s)\n", i+1, rec.Start, rec.End, label)
		for _, s := range rec.Strings {
			fmt.Printf("    +%d: %q\n", s.Offset, s.Value)
		}
	}
}

func printTailPresets(file *gll.File) {
	if file.TailData == nil || len(file.TailData.Presets) == 0 {
		fmt.Println("Tail presets: none")
		return
	}

	fmt.Println("Tail presets:")
	for i, p := range file.TailData.Presets {
		fmt.Printf("  #%d (%s)\n", i+1, p.Kind)
		if p.InputLabel != "" {
			fmt.Printf("    input: %s\n", p.InputLabel)
		}
		if p.InputKey != "" {
			fmt.Printf("    input_key: %s\n", p.InputKey)
		}
		if p.PresetLabel != "" {
			fmt.Printf("    preset: %s\n", p.PresetLabel)
		}
		if p.PresetKey != "" {
			fmt.Printf("    preset_key: %s\n", p.PresetKey)
		}
	}
}

func shannonEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}
	var freq [256]int
	for _, b := range data {
		freq[b]++
	}
	var ent float64
	inv := 1.0 / float64(len(data))
	for _, n := range freq {
		if n == 0 {
			continue
		}
		p := float64(n) * inv
		ent -= p * (log2(p))
	}
	return ent
}

func log2(x float64) float64 {
	const ln2 = 0.6931471805599453
	return math.Log(x) / ln2
}

func countPrintable(data []byte) (int, int) {
	count := 0
	for _, b := range data {
		if b >= 0x20 && b <= 0x7E {
			count++
		}
	}
	return count, len(data)
}

func printASCIIRuns(data []byte, minLen int) {
	type run struct {
		start int
		text  string
	}
	var runs []run
	start := -1
	for i, b := range data {
		if b >= 0x20 && b <= 0x7E {
			if start == -1 {
				start = i
			}
		} else if start != -1 {
			if i-start >= minLen {
				runs = append(runs, run{start: start, text: string(data[start:i])})
			}
			start = -1
		}
	}
	if start != -1 && len(data)-start >= minLen {
		runs = append(runs, run{start: start, text: string(data[start:])})
	}
	if len(runs) == 0 {
		fmt.Printf("ASCII runs (>= %d): none\n", minLen)
		return
	}
	fmt.Printf("ASCII runs (>= %d):\n", minLen)
	for _, r := range runs {
		fmt.Printf("  +%d: %q\n", r.start, r.text)
	}
}

//nolint:unparam // window kept configurable for tests covering different sizes
func printRepeats(data []byte, window int) {
	if len(data) < window*2 {
		fmt.Printf("Repeats (window %d): none\n", window)
		return
	}
	type hit struct {
		off1 int
		off2 int
	}
	seen := make(map[string]int)
	var hits []hit
	for i := 0; i+window <= len(data); i++ {
		chunk := string(data[i : i+window])
		if prev, ok := seen[chunk]; ok {
			if i-prev >= window {
				hits = append(hits, hit{off1: prev, off2: i})
			}
		} else {
			seen[chunk] = i
		}
		if len(hits) >= 5 {
			break
		}
	}
	if len(hits) == 0 {
		fmt.Printf("Repeats (window %d): none\n", window)
		return
	}
	fmt.Printf("Repeats (window %d):\n", window)
	for _, h := range hits {
		fmt.Printf("  +%d == +%d\n", h.off1, h.off2)
	}
}
