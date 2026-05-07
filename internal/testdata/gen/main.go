//go:build ignore

// gen writes testdata/gll/example-vis.gll — a small synthetic GLL with real
// balloon data, used by the web-demo smoke tests.
//
// Run from the repo root:
//
//	go run ./internal/testdata/gen/
package main

import (
	"log"
	"os"

	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
	"github.com/cwbudde/gll-tools/pkg/xgll"
)

func main() {
	out := "testdata/gll/example-vis.gll"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}

	f, err := os.Create(out)
	if err != nil {
		log.Fatalf("create %s: %v", out, err)
	}
	defer f.Close()

	if err := xgll.EncodeFile(buildFile(), f); err != nil {
		log.Fatalf("encode: %v", err)
	}
	log.Printf("wrote %s", out)
}

func buildFile() *gllbin.File {
	src := xgll.SyntheticSource("Full Range", "sdFullRange", 90.0)

	boxType := gllbin.BoxType{
		Label:   "Test Cabinet",
		Key:     "bxTest",
		Sources: []string{src.Key},
		SourcePlacements: []gllbin.BoxSource{
			{
				Label:        "Main Source",
				Key:          "srcMain",
				SourceDefKey: src.Key,
			},
		},
	}

	file := &gllbin.File{}
	file.Header.Magic = "EGLL"
	file.Header.FormatID = "EASE_GLL"
	file.Header.FormatVersion = 4
	file.GenSystem.Label = "Example Visualisation"
	file.GenSystem.Key = "sysExampleVis"
	file.GenSystem.Type = gllbin.SystemTypeLoudspeaker
	file.GenSystem.Company = "ExampleCo"
	file.GenSystem.InfoText = "Synthetic test fixture for web-demo smoke tests."
	file.Database = &gllbin.Database{
		SubVersion:        3,
		BoxTypes:          []gllbin.BoxType{boxType},
		SourceDefinitions: []gllbin.SourceDefinitionItem{src},
	}
	return file
}
