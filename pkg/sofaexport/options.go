package sofaexport

// Options controls how a GLL is exported to one or more SOFA files.
type Options struct {
	// Relative emits raw balloon transfer functions (no on-axis combine).
	// When false (default), each direction's TF is multiplied by
	// SourceDefinition.OnAxisSpectrum and scaled by OnAxisLevel.
	Relative bool

	// OutputDir is the directory where .sofa files are written. Defaults to ".".
	OutputDir string

	// FilenamePattern is a template containing {gll}, {source}, and {usecase}
	// placeholders. Defaults to "{gll}__{source}__{usecase}.sofa".
	FilenamePattern string

	// SourceFilter, when non-empty, restricts export to a single source whose
	// key or label matches.
	SourceFilter string

	// UseCaseFilter, when non-empty, restricts export to use cases matching
	// the given label.
	UseCaseFilter string

	// Overwrite, if true, replaces existing output files. When false the
	// exporter errors out instead of clobbering.
	Overwrite bool
}

// withDefaults returns a copy of opts with empty fields populated.
func (opts Options) withDefaults() Options {
	if opts.OutputDir == "" {
		opts.OutputDir = "."
	}
	if opts.FilenamePattern == "" {
		opts.FilenamePattern = "{gll}__{source}__{usecase}.sofa"
	}
	return opts
}
