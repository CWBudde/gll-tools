package clf_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/cwbudde/gll-tools/internal/clf"
	"github.com/cwbudde/gll-tools/pkg/gll"
)

func TestIntegration_ParseGLL_ExportCLF(t *testing.T) {
	entries, err := os.ReadDir("../../testdata/gll/")
	if err != nil {
		t.Skip("testdata not available")
	}

	var gllPath string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".gll") {
			gllPath = "../../testdata/gll/" + e.Name()
			break
		}
	}

	if gllPath == "" {
		t.Skip("no test GLL files found")
	}

	f, err := os.Open(gllPath)
	if err != nil {
		t.Fatalf("failed to open %s: %v", gllPath, err)
	}
	defer f.Close()

	file, err := gll.Parse(f)
	if err != nil {
		t.Fatalf("failed to parse GLL: %v", err)
	}

	if file.Database == nil || len(file.Database.SourceDefinitions) == 0 {
		t.Skip("no source definitions in test file")
	}

	src := file.Database.SourceDefinitions[0]
	if src.Definition == nil || src.Definition.BalloonData == nil {
		t.Skip("first source has no balloon data")
	}

	err = gll.LoadBalloonResponses(f, src.Definition.BalloonData)
	if err != nil {
		t.Fatalf("failed to load responses: %v", err)
	}

	var buf bytes.Buffer
	err = clf.ExportSource(&buf, src.Definition, &file.GenSystem)
	if err != nil {
		t.Fatalf("ExportSource failed: %v", err)
	}

	output := buf.String()

	if !strings.HasPrefix(output, "<CLF") {
		t.Error("output doesn't start with CLF tag")
	}
	if !strings.Contains(output, "END>") {
		t.Error("output doesn't contain END tag")
	}
	if !strings.Contains(output, "<BAND>\t") {
		t.Error("output contains no BAND data")
	}

	bandCount := strings.Count(output, "<BAND>\t")
	if bandCount != 24 && bandCount != 8 {
		t.Errorf("unexpected band count: %d", bandCount)
	}

	t.Logf("Successfully exported %d bytes, %d bands from %s", len(output), bandCount, gllPath)
}
