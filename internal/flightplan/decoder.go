package flightplan

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// FileNamePattern is what file names match valid enos configuration files
var FileNamePattern = regexp.MustCompile(`^enos[-\w]*?\.hcl$`)

// DecoderOpt is a functional option for a new flight plan
type DecoderOpt func(*Decoder) *Decoder

// NewDecoder takes functional options and returns a new flight plan
func NewDecoder(opts ...DecoderOpt) *Decoder {
	fp := &Decoder{
		parser: hclparse.NewParser(),
	}

	for _, opt := range opts {
		fp = opt(fp)
	}

	return fp
}

// WithDecoderDirectory sets the flight plan directory
func WithDecoderDirectory(path string) DecoderOpt {
	return func(fp *Decoder) *Decoder {
		fp.dir = path
		return fp
	}
}

// Decoder is our Enos flight plan, or, our representation of the HCL file(s)
// an author has composed.
type Decoder struct {
	parser *hclparse.Parser
	dir    string
}

// Parse locates enos configuration files and parses them.
func (f *Decoder) Parse() hcl.Diagnostics {
	// Parse the given directory, eventually we'll need to also look in the user
	// configuration directory as well.
	return f.parseDir(f.dir)
}

func (f *Decoder) parseDir(path string) hcl.Diagnostics {
	var diags hcl.Diagnostics

	// We can ignore the error returned from Walk() because we're aggregating
	// all errors and warnings into diags, which we'll handle afterwards.
	_ = filepath.Walk(f.dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			d := hcl.Diagnostics{
				{
					Severity: hcl.DiagError,
					Summary:  "Failed to read directory",
					Detail:   fmt.Sprintf("Failed to load file information for %s: %s", path, err.Error()),
				},
			}
			diags = diags.Extend(d)

			return d
		}

		// We're only going a single level deep for now so we can ingnore directories
		if info.IsDir() {
			// Always skip the directory unless it's the root we're walking
			if path != f.dir {
				return filepath.SkipDir
			}
		}

		if !FileNamePattern.MatchString(info.Name()) {
			return nil
		}

		_, pDiags := f.parser.ParseHCLFile(path)
		diags = diags.Extend(pDiags)
		if pDiags.HasErrors() {
			return pDiags
		}

		return nil
	})

	return diags
}

func (f *Decoder) parseHCL(src []byte, fname string) hcl.Diagnostics {
	_, diags := f.parser.ParseHCL(src, fname)

	return diags
}

func (f *Decoder) mergedBody() hcl.Body {
	files := []*hcl.File{}

	for _, file := range f.parser.Files() {
		files = append(files, file)
	}

	return hcl.MergeFiles(files)
}

// Decode decodes the HCL into a flight plan. It is done in several passes.
func (f *Decoder) Decode() (*FlightPlan, hcl.Diagnostics) {
	fp := NewFlightPlan()

	diags := gohcl.DecodeBody(f.mergedBody(), nil, fp)

	return fp, diags
}

// NewFlightPlan returns a new instance of a FlightPlan
func NewFlightPlan() *FlightPlan {
	return &FlightPlan{
		Scenarios: []*Scenario{},
	}
}

// FlightPlan represents out flight plan. Due to the complexity of the Enos DSL
// we have to decode in several different passes.
type FlightPlan struct {
	Scenarios []*Scenario `hcl:"scenario,block"`
	Remain    hcl.Body    `hcl:",remain"`
}

// Scenario represents a scenario
type Scenario struct {
	Name string `hcl:"name,label"`
}
