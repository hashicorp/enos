package flightplan

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan/funcs"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/tryfunc"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// DecodeTarget determines the depth of flight plan decoding so we only ever decode, expand
// and validate information relevant to our operation.
type DecodeTarget int

const (
	DecodeTargetUnset = iota
	DecodeTargetVariables
	DecodeTargetGlobals
	DecodeTargetSamples
	DecodeTargetScenariosNamesNoVariants
	DecodeTargetScenariosMatrixOnly
	DecodeTargetScenariosNamesExpandVariants
	DecodeTargetTerraformSettings
	DecodeTargetTerraformCLIs
	DecodeTargetProviders
	DecodeTargetModules
	DecodeTargetScenariosComplete
	DecodeTargetAll // Make sure this is always the latest decoding target
)

// DecoderOpt is a functional option for a new flight plan.
type DecoderOpt func(*Decoder) error

// NewDecoder takes functional options and returns a new flight plan.
func NewDecoder(opts ...DecoderOpt) (*Decoder, error) {
	d := &Decoder{
		FPParser:   hclparse.NewParser(),
		VarsParser: hclparse.NewParser(),
		target:     DecodeTargetAll,
	}

	for _, opt := range opts {
		err := opt(d)
		if err != nil {
			return d, err
		}
	}

	return d, nil
}

// WithDecoderFPFiles sets the flight plan contents as raw bytes.
func WithDecoderFPFiles(files RawFiles) DecoderOpt {
	return func(fp *Decoder) error {
		fp.fpFiles = files

		return nil
	}
}

// WithDecoderVarFiles sets the flight plan variable contents as raw bytes.
func WithDecoderVarFiles(files RawFiles) DecoderOpt {
	return func(fp *Decoder) error {
		fp.varFiles = files

		return nil
	}
}

// WithDecoderEnv sets flight plan variables from env vars.
func WithDecoderEnv(vars []string) DecoderOpt {
	return func(fp *Decoder) error {
		fp.varEnvVars = vars

		return nil
	}
}

// WithDecoderBaseDir sets the flight plan base directory.
func WithDecoderBaseDir(path string) DecoderOpt {
	return func(fp *Decoder) error {
		var err error
		fp.dir, err = filepath.Abs(path)

		return err
	}
}

// WithDecoderDecodeTarget sets the decoding mode.
func WithDecoderDecodeTarget(mode DecodeTarget) DecoderOpt {
	return func(fp *Decoder) error {
		fp.target = mode

		return nil
	}
}

// WithDecoderScenarioFilter sets the scenario decoding filter.
func WithDecoderScenarioFilter(filter *ScenarioFilter) DecoderOpt {
	return func(fp *Decoder) error {
		fp.filter = filter

		return nil
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
	target     DecodeTarget
	filter     *ScenarioFilter
}

// Parse locates enos configuration files and parses them.
func (d *Decoder) Parse() hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	// Parse and raw configuration bytes we've been configured with
	return diags.Extend(d.parseRawFiles())
}

func (d *Decoder) parseRawFiles() hcl.Diagnostics {
	diags := hcl.Diagnostics{}

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
func (d *Decoder) Decode(ctx context.Context) (*FlightPlan, hcl.Diagnostics) {
	diags := hcl.Diagnostics{}

	fp, err := NewFlightPlan(WithFlightPlanBaseDirectory(d.dir))
	if err != nil {
		return fp, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "unable to create new flight plan",
			Detail:   fmt.Sprintf("unable to create new flight plan: %s", err.Error()),
		})
	}

	if fp.BaseDir == "" {
		return fp, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "unable to decode flight plan without a base directory",
		})
	}

	evalCtx := d.baseEvalContext()
	if evalCtx == nil {
		evalCtx = &hcl.EvalContext{
			Variables: map[string]cty.Value{},
			Functions: map[string]function.Function{},
		}
	}

	fpFiles := d.FPParser.Files()
	varsFiles := d.VarsParser.Files()

	// Create a "unified" body of all flightplan files to use for decoding
	files := []*hcl.File{}
	for _, file := range fpFiles {
		files = append(files, file)
	}
	body := hcl.MergeFiles(files)

	// Decode our top-level schema
	var moreDiags hcl.Diagnostics
	fp.BodyContent, moreDiags = body.Content(flightPlanSchema)
	diags = diags.Extend(moreDiags)
	if diags.HasErrors() {
		return fp, diags
	}

	// Decode to our desired target level. Start with the lowest level and continue until we've
	// reached our desired target. Each target level includes more blocks. Where appropriate, each
	// decoder is responsible for extending the eval context and/or falling through to the next
	// level.
	decodeToLevel := func() hcl.Diagnostics {
		diags := hcl.Diagnostics{}

		if d.target < DecodeTargetUnset {
			return diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unsupported flight plan decode target level",
				Detail:   fmt.Sprintf("The configured target decode level was less than minimum allowed. Expected a level >= 1, Received level: %d", d.target),
				Subject:  body.MissingItemRange().Ptr(),
			})
		}

		if d.target == DecodeTargetUnset {
			return diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Flight plan decode target level must be configured",
				Subject:  body.MissingItemRange().Ptr(),
			})
		}

		if d.target >= DecodeTargetVariables {
			// Decode and validate our variables and add them to the eval context.
			diags = diags.Extend(fp.decodeVariables(evalCtx, varsFiles, d.varEnvVars))
		}

		if d.target >= DecodeTargetGlobals {
			// Decode our globals and add them to the eval context.
			diags = diags.Extend(fp.decodeGlobals(evalCtx))
		}

		if d.target >= DecodeTargetSamples {
			// Decode to only our samples but does not verify correctness or an intersection with scenarios.
			diags = diags.Extend(fp.decodeSamples(evalCtx))
		}

		if d.target == DecodeTargetScenariosNamesNoVariants {
			// Decode to only our scenario names. Useful for only decoding the scenario names for
			// listing but not validating their internal references or expanding their variants.
			return diags.Extend(fp.decodeScenarios(ctx, evalCtx, d.target, d.filter))
		}

		if d.target == DecodeTargetScenariosMatrixOnly {
			// Decode scenarios to name and matrix only. Useful for shallow decoding scenarios
			// and building sample frames.
			return diags.Extend(fp.decodeScenarios(ctx, evalCtx, d.target, d.filter))
		}

		if d.target == DecodeTargetScenariosNamesExpandVariants {
			// Decode to only our scenario names and variants. Useful for listing all scenarios
			// and variant combinations.
			return diags.Extend(fp.decodeScenarios(ctx, evalCtx, d.target, d.filter))
		}

		if d.target >= DecodeTargetTerraformSettings {
			diags = diags.Extend(fp.decodeTerraformSettings(evalCtx))
		}

		if d.target >= DecodeTargetTerraformCLIs {
			diags = diags.Extend(fp.decodeTerraformCLIs(evalCtx))
		}

		if d.target >= DecodeTargetProviders {
			diags = diags.Extend(fp.decodeProviders(evalCtx))
		}

		if d.target >= DecodeTargetModules {
			diags = diags.Extend(fp.decodeModules(evalCtx))
		}

		if d.target >= DecodeTargetScenariosComplete {
			// Decode scenarios and fully validate them.
			diags = diags.Extend(fp.decodeScenarios(ctx, evalCtx, d.target, d.filter))
		}

		if d.target > DecodeTargetAll {
			return diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unsupported flight plan decode target level",
				Detail:   fmt.Sprintf("The configured target decode level was greater than maximum allowed. Expected a level <= %d, Received level: %d", DecodeTargetAll, d.target),
				Subject:  body.MissingItemRange().Ptr(),
			})
		}

		return diags
	}

	diags = diags.Extend(decodeToLevel())

	return fp, diags
}

// DecodeProto takes a wire request of a FlightPlan and returns a new flight plan and a wire encodable
// decode response.
func DecodeProto(
	ctx context.Context,
	pfp *pb.FlightPlan,
	target DecodeTarget,
	f *pb.Scenario_Filter,
) (*FlightPlan, *pb.DecodeResponse) {
	res := &pb.DecodeResponse{
		Diagnostics: []*pb.Diagnostic{},
	}

	opts := []DecoderOpt{
		WithDecoderBaseDir(pfp.GetBaseDir()),
		WithDecoderFPFiles(pfp.GetEnosHcl()),
		WithDecoderVarFiles(pfp.GetEnosVarsHcl()),
		WithDecoderEnv(pfp.GetEnosVarsEnv()),
		WithDecoderDecodeTarget(target),
	}

	sf, err := NewScenarioFilter(WithScenarioFilterDecode(f))
	if err != nil {
		res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(err)...)
	} else {
		opts = append(opts, WithDecoderScenarioFilter(sf))
	}

	dec, err := NewDecoder(opts...)
	if err != nil {
		res.Diagnostics = diagnostics.FromErr(err)

		return nil, res
	}

	hclDiags := dec.Parse()
	if len(hclDiags) > 0 {
		res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromHCL(dec.ParserFiles(), hclDiags)...)
	}

	if diagnostics.HasErrors(res.GetDiagnostics()) {
		return nil, res
	}

	fp, hclDiags := dec.Decode(ctx)
	if len(hclDiags) > 0 {
		res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromHCL(dec.ParserFiles(), hclDiags)...)
	}

	return fp, res
}
