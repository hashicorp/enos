package flightplan

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"

	"github.com/hashicorp/enos/internal/flightplan/funcs"
	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// FlightPlanFileNamePattern is what file names match valid enos configuration files
var (
	FlightPlanFileNamePattern = regexp.MustCompile(`^enos[-\w]*?\.hcl$`)
	VariablesNamePattern      = regexp.MustCompile(`^enos[-\w]*?\.vars\.hcl$`)
)

// DecoderOpt is a functional option for a new flight plan
type DecoderOpt func(*Decoder) error

// NewDecoder takes functional options and returns a new flight plan
func NewDecoder(opts ...DecoderOpt) (*Decoder, error) {
	d := &Decoder{
		FPParser:   hclparse.NewParser(),
		VarsParser: hclparse.NewParser(),
	}

	for _, opt := range opts {
		err := opt(d)
		if err != nil {
			return d, err
		}
	}

	return d, nil
}

// WithDecoderBaseDir sets the flight plan base directory
func WithDecoderBaseDir(path string) DecoderOpt {
	return func(fp *Decoder) error {
		var err error
		fp.dir, err = filepath.Abs(path)
		return err
	}
}

// Decoder is our Enos flight plan, or, our representation of the HCL file(s)
// an author has composed.
type Decoder struct {
	FPParser   *hclparse.Parser
	VarsParser *hclparse.Parser
	dir        string
}

// Parse locates enos configuration files and parses them.
func (d *Decoder) Parse() hcl.Diagnostics {
	// Parse the given directory, eventually we'll need to also look in the user
	// configuration directory as well.
	return d.parseDir(d.dir)
}

// parseDir walks the directory and parses any Enos HCL or variables files.
func (d *Decoder) parseDir(path string) hcl.Diagnostics {
	var diags hcl.Diagnostics

	// We can ignore the error returned from Walk() because we're aggregating
	// all errors and warnings into diags, which we'll handle afterwards.
	_ = filepath.Walk(d.dir, func(path string, info fs.FileInfo, err error) error {
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
			if path != d.dir {
				return filepath.SkipDir
			}
		}

		if VariablesNamePattern.MatchString(info.Name()) {
			_, pDiags := d.VarsParser.ParseHCLFile(path)
			diags = diags.Extend(pDiags)
			if pDiags.HasErrors() {
				return pDiags
			}

			return nil
		}

		if FlightPlanFileNamePattern.MatchString(info.Name()) {
			_, pDiags := d.FPParser.ParseHCLFile(path)
			diags = diags.Extend(pDiags)
			if pDiags.HasErrors() {
				return pDiags
			}

			return nil
		}

		return nil
	})

	return diags
}

// ParserFiles returns combined parser files. These files can be used to add
// context to diagnostics.
func (d *Decoder) ParserFiles() map[string]*hcl.File {
	files := d.FPParser.Files()
	for name, file := range d.VarsParser.Files() {
		files[name] = file
	}

	return files
}

// baseEvalContext is the root eval context that we'll use during flight plan
// decoding.
func (d *Decoder) baseEvalContext() *hcl.EvalContext {
	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"path": cty.ObjectVal(map[string]cty.Value{
				"root": cty.StringVal(d.dir),
			}),
		},
		Functions: map[string]function.Function{
			"absolute":               stdlib.AbsoluteFunc,
			"abspath":                funcs.AbsPathFunc,
			"add":                    stdlib.AddFunc,
			"and":                    stdlib.AndFunc,
			"byteslen":               stdlib.BytesLenFunc,
			"bytessclice":            stdlib.BytesSliceFunc,
			"csvdecode":              stdlib.CSVDecodeFunc,
			"ceil":                   stdlib.CeilFunc,
			"chomp":                  stdlib.ChompFunc,
			"chunklist":              stdlib.ChunklistFunc,
			"coalesce":               stdlib.CoalesceFunc,
			"coalescelist":           stdlib.CoalesceListFunc,
			"compact":                stdlib.CompactFunc,
			"concat":                 stdlib.ConcatFunc,
			"distinct":               stdlib.DistinctFunc,
			"divide":                 stdlib.DivideFunc,
			"element":                stdlib.ElementFunc,
			"equal":                  stdlib.EqualFunc,
			"flatten":                stdlib.FlattenFunc,
			"floor":                  stdlib.FloorFunc,
			"format":                 stdlib.FormatFunc,
			"formatdate":             stdlib.FormatDateFunc,
			"formatlist":             stdlib.FormatListFunc,
			"greaterthan":            stdlib.GreaterThanFunc,
			"greaterthanorequalto":   stdlib.GreaterThanOrEqualToFunc,
			"hasindex":               stdlib.HasIndexFunc,
			"indent":                 stdlib.IndentFunc,
			"index":                  stdlib.IndexFunc,
			"int":                    stdlib.IntFunc,
			"jsondecode":             stdlib.JSONDecodeFunc,
			"jsonencode":             stdlib.JSONEncodeFunc,
			"join":                   stdlib.JoinFunc,
			"joinpath":               funcs.JoinPathFunc,
			"keys":                   stdlib.KeysFunc,
			"length":                 stdlib.LengthFunc,
			"lessthan":               stdlib.LessThanFunc,
			"lessthanorequalto":      stdlib.LessThanOrEqualToFunc,
			"log":                    stdlib.LogFunc,
			"lookup":                 stdlib.LookupFunc,
			"lower":                  stdlib.LowerFunc,
			"max":                    stdlib.MaxFunc,
			"merge":                  stdlib.MergeFunc,
			"min":                    stdlib.MinFunc,
			"modulo":                 stdlib.ModuloFunc,
			"multiply":               stdlib.MultiplyFunc,
			"negate":                 stdlib.NegateFunc,
			"not":                    stdlib.NotFunc,
			"notequal":               stdlib.NotEqualFunc,
			"or":                     stdlib.OrFunc,
			"parseint":               stdlib.ParseIntFunc,
			"pow":                    stdlib.PowFunc,
			"range":                  stdlib.RangeFunc,
			"regex":                  stdlib.RegexFunc,
			"regexall":               stdlib.RegexAllFunc,
			"regexreplace":           stdlib.RegexReplaceFunc,
			"replace":                stdlib.ReplaceFunc,
			"reverse":                stdlib.ReverseFunc,
			"reverselist":            stdlib.ReverseListFunc,
			"sethaselement":          stdlib.SetHasElementFunc,
			"setintersection":        stdlib.SetIntersectionFunc,
			"setproduct":             stdlib.SetProductFunc,
			"setsubtract":            stdlib.SetSubtractFunc,
			"setsymmetricdifference": stdlib.SetSymmetricDifferenceFunc,
			"setunion":               stdlib.SetUnionFunc,
			"signum":                 stdlib.SignumFunc,
			"slice":                  stdlib.SliceFunc,
			"sort":                   stdlib.SortFunc,
			"split":                  stdlib.SplitFunc,
			"strlen":                 stdlib.StrlenFunc,
			"substr":                 stdlib.SubstrFunc,
			"subtract":               stdlib.SubtractFunc,
			"title":                  stdlib.TitleFunc,
			"trim":                   stdlib.TrimFunc,
			"trimprefix":             stdlib.TrimPrefixFunc,
			"trimspace":              stdlib.TrimSpaceFunc,
			"trimsuffix":             stdlib.TrimSuffixFunc,
			"upper":                  stdlib.UpperFunc,
			"values":                 stdlib.ValuesFunc,
			"zipmap":                 stdlib.ZipmapFunc,
		},
	}
}

// Decode decodes the HCL into a flight plan.
func (d *Decoder) Decode() (*FlightPlan, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	fp, err := NewFlightPlan(WithFlightPlanBaseDirectory(d.dir))
	if err != nil {
		return fp, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "unable to create new flight plan",
			Detail:   fmt.Sprintf("unable to create new flight plan: %s", err.Error()),
		})
	}

	return fp, diags.Extend(fp.Decode(d.baseEvalContext(), d.FPParser.Files(), d.VarsParser.Files()))
}
