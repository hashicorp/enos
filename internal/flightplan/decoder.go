package flightplan

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"

	hcl "github.com/hashicorp/hcl/v2"
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

func (f *Decoder) baseEvalContext() *hcl.EvalContext {
	return &hcl.EvalContext{
		Variables: map[string]cty.Value{},
		Functions: map[string]function.Function{
			"absolute":               stdlib.AbsoluteFunc,
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
func (f *Decoder) Decode() (*FlightPlan, hcl.Diagnostics) {
	fp := NewFlightPlan()
	diags := fp.Decode(f.baseEvalContext(), f.mergedBody())

	return fp, diags
}
