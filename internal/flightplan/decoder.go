package flightplan

import (
	"fmt"
	"path/filepath"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"

	"github.com/hashicorp/enos/internal/flightplan/funcs"
	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/tryfunc"
	"github.com/hashicorp/hcl/v2/hclparse"
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

// WithDecoderFPFiles sets the flight plan contents as raw bytes
func WithDecoderFPFiles(files RawFiles) DecoderOpt {
	return func(fp *Decoder) error {
		fp.fpFiles = files
		return nil
	}
}

// WithDecoderVarFiles sets the flight plan variable contents as raw bytes
func WithDecoderVarFiles(files RawFiles) DecoderOpt {
	return func(fp *Decoder) error {
		fp.varFiles = files
		return nil
	}
}

// WithDecoderEnv sets flight plan variables from env vars
func WithDecoderEnv(vars []string) DecoderOpt {
	return func(fp *Decoder) error {
		fp.varEnvVars = vars
		return nil
	}
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
	fpFiles    RawFiles
	varFiles   RawFiles
	varEnvVars []string
	dir        string
}

// Parse locates enos configuration files and parses them.
func (d *Decoder) Parse() hcl.Diagnostics {
	var diags hcl.Diagnostics

	// Parse and raw configuration bytes we've been configured with
	return diags.Extend(d.parseRawFiles())
}

func (d *Decoder) parseRawFiles() hcl.Diagnostics {
	var diags hcl.Diagnostics

	for path, bytes := range d.fpFiles {
		_, moreDiags := d.FPParser.ParseHCL(bytes, path)
		diags = diags.Extend(moreDiags)
	}

	for path, bytes := range d.varFiles {
		_, moreDiags := d.VarsParser.ParseHCL(bytes, path)
		diags = diags.Extend(moreDiags)
	}

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
			"abspath":                funcs.AbsPathFunc(d.dir),
			"add":                    stdlib.AddFunc,
			"and":                    stdlib.AndFunc,
			"byteslen":               stdlib.BytesLenFunc,
			"bytessclice":            stdlib.BytesSliceFunc,
			"can":                    tryfunc.CanFunc,
			"ceil":                   stdlib.CeilFunc,
			"chomp":                  stdlib.ChompFunc,
			"chunklist":              stdlib.ChunklistFunc,
			"coalesce":               stdlib.CoalesceFunc,
			"coalescelist":           stdlib.CoalesceListFunc,
			"compact":                stdlib.CompactFunc,
			"concat":                 stdlib.ConcatFunc,
			"contains":               stdlib.ContainsFunc,
			"csvdecode":              stdlib.CSVDecodeFunc,
			"distinct":               stdlib.DistinctFunc,
			"divide":                 stdlib.DivideFunc,
			"element":                stdlib.ElementFunc,
			"equal":                  stdlib.EqualFunc,
			"file":                   funcs.FileFunc,
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
			"semverconstraint":       funcs.SemverConstraint,
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
			"timeadd":                stdlib.TimeAddFunc,
			"title":                  stdlib.TitleFunc,
			"trim":                   stdlib.TrimFunc,
			"trimprefix":             stdlib.TrimPrefixFunc,
			"trimspace":              stdlib.TrimSpaceFunc,
			"trimsuffix":             stdlib.TrimSuffixFunc,
			"try":                    tryfunc.TryFunc,
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

	return fp, diags.Extend(
		fp.Decode(
			d.baseEvalContext(),
			d.FPParser.Files(),
			d.VarsParser.Files(),
			d.varEnvVars,
		),
	)
}
