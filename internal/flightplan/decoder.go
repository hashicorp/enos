// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"context"
	"fmt"
	"path/filepath"

	yaml "github.com/zclconf/go-cty-yaml"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan/funcs"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
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
	DecodeTargetQualities
	DecodeTargetScenariosNamesNoVariants
	DecodeTargetScenariosMatrixOnly
	DecodeTargetScenariosNamesExpandVariants
	DecodeTargetTerraformSettings
	DecodeTargetTerraformCLIs
	DecodeTargetProviders
	DecodeTargetModules
	DecodeTargetScenariosOutlines
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
			"abs":                    stdlib.AbsoluteFunc,
			"absolute":               stdlib.AbsoluteFunc,
			"abspath":                funcs.AbsPathFunc(d.dir),
			"add":                    stdlib.AddFunc,
			"alltrue":                funcs.AllTrueFunc,
			"and":                    stdlib.AndFunc,
			"anytrue":                funcs.AnyTrueFunc,
			"base64decode":           funcs.Base64DecodeFunc,
			"base64encode":           funcs.Base64EncodeFunc,
			"base64gzip":             funcs.Base64GzipFunc,
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
			"endswith":               funcs.EndsWithFunc,
			"file":                   funcs.FileFunc(d.dir),
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
			"matchkeys":              funcs.MatchkeysFunc,
			"max":                    stdlib.MaxFunc,
			"merge":                  stdlib.MergeFunc,
			"min":                    stdlib.MinFunc,
			"modulo":                 stdlib.ModuloFunc,
			"multiply":               stdlib.MultiplyFunc,
			"negate":                 stdlib.NegateFunc,
			"not":                    stdlib.NotFunc,
			"notequal":               stdlib.NotEqualFunc,
			"or":                     stdlib.OrFunc,
			"one":                    funcs.OneFunc,
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
			"startswith":             funcs.StartsWithFunc,
			"strcontains":            funcs.StrContainsFunc,
			"strlen":                 stdlib.StrlenFunc,
			"substr":                 stdlib.SubstrFunc,
			"subtract":               stdlib.SubtractFunc,
			"sum":                    funcs.SumFunc,
			"textdecodebase64":       funcs.TextDecodeBase64Func,
			"textencodebase64":       funcs.TextEncodeBase64Func,
			"timeadd":                stdlib.TimeAddFunc,
			"timestamp":              funcs.TimestampFunc,
			"timecmp":                funcs.TimeCmpFunc,
			"title":                  stdlib.TitleFunc,
			"transpose":              funcs.TransposeFunc,
			"trim":                   stdlib.TrimFunc,
			"trimprefix":             stdlib.TrimPrefixFunc,
			"trimspace":              stdlib.TrimSpaceFunc,
			"trimsuffix":             stdlib.TrimSuffixFunc,
			"try":                    tryfunc.TryFunc,
			"upper":                  stdlib.UpperFunc,
			"urlencode":              funcs.URLEncodeFunc,
			"values":                 stdlib.ValuesFunc,
			"yamldecode":             yaml.YAMLDecodeFunc,
			"yamlencode":             yaml.YAMLEncodeFunc,
			"zipmap":                 stdlib.ZipmapFunc,
		},
	}
}

// decodeScenarios decodes the "scenario" blocks that are defined in the top-level schema.
func (d *Decoder) scenarioDecoder(
	evalCtx *hcl.EvalContext,
	fp *FlightPlan,
	target DecodeTarget,
	filter *ScenarioFilter,
) (*ScenarioDecoder, hcl.Diagnostics) {
	diags := hcl.Diagnostics{}

	scenarioDecoder, err := NewScenarioDecoder(
		WithScenarioDecoderEvalContext(evalCtx),
		WithScenarioDecoderDecodeTarget(target),
		WithScenarioDecoderScenarioFilter(filter),
		WithScenarioDecoderBlocks(fp.BodyContent.Blocks.OfType(blockTypeScenario)),
	)
	if err != nil {
		return nil, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "unable to initialize scenario decoder",
			Detail:   err.Error(),
			Subject:  fp.BodyContent.MissingItemRange.Ptr(),
		})
	}

	return scenarioDecoder, nil
}

// Decode decodes the HCL into a flightplan and a scenario decoder. Use the scenario decoder to
// decode individual scenarios.
//
//nolint:cyclop // it's a complex func
func (d *Decoder) Decode(ctx context.Context) (*FlightPlan, *ScenarioDecoder, hcl.Diagnostics) {
	diags := hcl.Diagnostics{}

	fp, err := NewFlightPlan(WithFlightPlanBaseDirectory(d.dir))
	if err != nil {
		return fp, nil, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "unable to create new flight plan",
			Detail:   "unable to create new flight plan: " + err.Error(),
		})
	}

	if fp.BaseDir == "" {
		return fp, nil, diags.Append(&hcl.Diagnostic{
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
		return fp, nil, diags
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
			if diags != nil && diags.HasErrors() {
				return diags
			}
		}

		if d.target >= DecodeTargetGlobals {
			// Decode our globals and add them to the eval context.
			diags = diags.Extend(fp.decodeGlobals(evalCtx))
			if diags != nil && diags.HasErrors() {
				return diags
			}
		}

		if d.target >= DecodeTargetSamples {
			// Decode to only our samples but does not verify correctness or an intersection with scenarios.
			diags = diags.Extend(fp.decodeSamples(evalCtx))
			if diags != nil && diags.HasErrors() {
				return diags
			}
		}

		if d.target >= DecodeTargetQualities {
			// Decode out qualities and add them to the eval context.
			diags = diags.Extend(fp.decodeQualities(evalCtx))
			if diags != nil && diags.HasErrors() {
				return diags
			}
		}

		switch d.target {
		case
			// Decode to only our scenario names. Useful for only decoding the scenario names for
			// listing but not validating their internal references or expanding their variants.
			DecodeTargetScenariosNamesNoVariants,
			// Decode scenarios to name and matrix only. Useful for shallow decoding scenarios
			// and building sample frames.
			DecodeTargetScenariosMatrixOnly,
			// Decode to only our scenario names and variants. Useful for listing all scenarios
			// and variant combinations.
			DecodeTargetScenariosNamesExpandVariants:
			return diags // The caller will need to DecodeAll() if they want scenarios
		default:
		}

		if d.target >= DecodeTargetTerraformSettings {
			diags = diags.Extend(fp.decodeTerraformSettings(evalCtx))
			if diags != nil && diags.HasErrors() {
				return diags
			}
		}

		if d.target >= DecodeTargetTerraformCLIs {
			diags = diags.Extend(fp.decodeTerraformCLIs(evalCtx))
			if diags != nil && diags.HasErrors() {
				return diags
			}
		}

		if d.target >= DecodeTargetProviders {
			diags = diags.Extend(fp.decodeProviders(evalCtx))
			if diags != nil && diags.HasErrors() {
				return diags
			}
		}

		if d.target >= DecodeTargetModules {
			diags = diags.Extend(fp.decodeModules(evalCtx))
			if diags != nil && diags.HasErrors() {
				return diags
			}
		}

		return diags
	}

	diags = diags.Extend(decodeToLevel())
	if diags.HasErrors() {
		return fp, nil, diags
	}

	if d.target > DecodeTargetAll {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unsupported flight plan decode target level",
			Detail:   fmt.Sprintf("The configured target decode level was greater than maximum allowed. Expected a level <= %d, Received level: %d", DecodeTargetAll, d.target),
			Subject:  body.MissingItemRange().Ptr(),
		})
	}

	// Handle decoding our scenarios as an interator to allow callers more control of them.
	scenarioDecoder, moreDiags := d.scenarioDecoder(evalCtx, fp, d.target, d.filter)
	diags = diags.Extend(moreDiags)

	return fp, scenarioDecoder, diags
}

// DecodeProto takes a wire request of a FlightPlan and returns a new flight plan and a wire encodable
// decode response. It's up to the caller to utilize the ScenarioDecoder to decode individual
// scenarios.
func DecodeProto(
	ctx context.Context,
	pfp *pb.FlightPlan,
	target DecodeTarget,
	f *pb.Scenario_Filter,
) (*FlightPlan, *ScenarioDecoder, *pb.DecodeResponse) {
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

		return nil, nil, res
	}

	hclDiags := dec.Parse()
	if len(hclDiags) > 0 {
		res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromHCL(dec.ParserFiles(), hclDiags)...)
	}

	if diagnostics.HasErrors(res.GetDiagnostics()) {
		return nil, nil, res
	}

	fp, scenarioDecoder, hclDiags := dec.Decode(ctx)
	if len(hclDiags) > 0 {
		res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromHCL(dec.ParserFiles(), hclDiags)...)
	}

	return fp, scenarioDecoder, res
}
