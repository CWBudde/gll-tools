package sofaexport

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cwbudde/gll-tools/pkg/gll"
)

// ErrSkipped is returned by ExportSourceBalloon when the source/use case did
// not match the supplied filters.
var ErrSkipped = errors.New("skipped by filter")

// ExportSourceBalloon builds and writes a single SOFA file for a (source,
// balloon) pair. Returns the output path on success.
//
// The caller is responsible for ensuring the balloon's responses have been
// loaded (gll.LoadBalloonResponses).
func ExportSourceBalloon(
	src *gll.SourceDefinition,
	balloon *gll.BalloonData,
	ctx BuildContext,
	gllStem string,
	sourceKey string,
	opts Options,
) (string, error) {
	opts = opts.withDefaults()

	if !filterMatches(opts.SourceFilter, sourceKey, src.Label) {
		return "", ErrSkipped
	}
	if !filterMatches(opts.UseCaseFilter, ctx.UseCase) {
		return "", ErrSkipped
	}

	f, err := BuildSOFAFile(src, balloon, ctx, opts)
	if err != nil {
		return "", err
	}

	useCase := ctx.UseCase
	if useCase == "" {
		useCase = defaultUseCase
	}
	name := renderPattern(opts.FilenamePattern, gllStem, sourceKey, useCase, src.Label)
	out := filepath.Join(opts.OutputDir, name)

	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", opts.OutputDir, err)
	}
	if !opts.Overwrite {
		if _, err := os.Stat(out); err == nil {
			return "", fmt.Errorf("output file %s exists (use --overwrite)", out)
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}

	if err := f.Save(out); err != nil {
		return "", fmt.Errorf("save %s: %w", out, err)
	}
	return out, nil
}

// ExportFile parses a GLL file and writes one .sofa file per BalloonData per
// SourceDefinition. Returns the list of written paths. Skips sources/cases
// excluded by Options.SourceFilter / Options.UseCaseFilter.
func ExportFile(gllPath string, opts Options) ([]string, error) {
	opts = opts.withDefaults()

	fh, err := os.Open(gllPath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", gllPath, err)
	}
	defer fh.Close()

	parsed, err := gll.Parse(fh)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", gllPath, err)
	}
	if parsed.Database == nil {
		return nil, fmt.Errorf("no database section found in %s", gllPath)
	}

	manufacturer := parsed.GenSystem.Company
	if manufacturer == "" {
		manufacturer = parsed.Metadata.Manufacturer
	}
	model := parsed.GenSystem.Label
	if model == "" {
		model = parsed.Metadata.ProductName
	}

	stem := filenameStem(gllPath)

	var out []string
	for _, item := range parsed.Database.SourceDefinitions {
		src := item.Definition
		if src == nil || src.BalloonData == nil {
			continue
		}
		balloon := src.BalloonData

		// Lazy-load responses if not yet populated.
		if len(balloon.Responses) == 0 && balloon.ResponseCount > 0 {
			if _, err := fh.Seek(0, io.SeekStart); err != nil {
				return out, fmt.Errorf("seek for balloon load: %w", err)
			}
			if err := gll.LoadBalloonResponses(fh, balloon); err != nil {
				return out, fmt.Errorf("load balloon for source %q: %w", item.Key, err)
			}
		}

		ctx := BuildContext{
			Manufacturer: manufacturer,
			Model:        model,
			UseCase:      defaultUseCase,
		}
		path, err := ExportSourceBalloon(src, balloon, ctx, stem, item.Key, opts)
		if errors.Is(err, ErrSkipped) {
			continue
		}
		if err != nil {
			return out, fmt.Errorf("source %q: %w", item.Key, err)
		}
		out = append(out, path)
	}
	return out, nil
}

func filterMatches(filter string, candidates ...string) bool {
	if filter == "" {
		return true
	}
	for _, c := range candidates {
		if c == filter {
			return true
		}
	}
	return false
}

var unsafeFilenameChars = regexp.MustCompile(`[^A-Za-z0-9_.-]`)

func sanitizeFilenamePart(s string) string {
	if s == "" {
		return defaultUseCase
	}
	return unsafeFilenameChars.ReplaceAllString(s, "_")
}

func filenameStem(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func renderPattern(pattern, gllStem, sourceKey, useCase, sourceLabel string) string {
	src := sourceKey
	if src == "" {
		src = sourceLabel
	}
	r := strings.NewReplacer(
		"{gll}", sanitizeFilenamePart(gllStem),
		"{source}", sanitizeFilenamePart(src),
		"{usecase}", sanitizeFilenamePart(useCase),
	)
	name := r.Replace(pattern)
	if !strings.HasSuffix(strings.ToLower(name), ".sofa") {
		name += ".sofa"
	}
	return name
}
