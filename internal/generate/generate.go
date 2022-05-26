package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// Opt is a generate module option
type Opt func(*Generator) error

// Generator is a request to generate a Terraform module
type Generator struct {
	Scenario *flightplan.Scenario
	BaseDir  string
	OutDir   string
	UI       cli.Ui
}

// NewGenerator takes options and returns a new validated generator
func NewGenerator(opts ...Opt) (*Generator, error) {
	req := &Generator{}

	for _, opt := range opts {
		err := opt(req)
		if err != nil {
			return req, err
		}
	}

	if req.Scenario == nil {
		return req, fmt.Errorf("unable to generate without a scenario")
	}

	if req.BaseDir == "" {
		return req, fmt.Errorf("unable to generate without a base directory")
	}

	if req.OutDir == "" {
		return req, fmt.Errorf("unable to generate without an out directory")
	}

	return req, nil
}

// WithOutBaseDirectory is the destination base directory where modules will be written.
func WithOutBaseDirectory(dir string) Opt {
	return func(req *Generator) error {
		a, err := absoluteNoSymlinks(dir)
		if err != nil {
			return err
		}

		req.OutDir = a
		return nil
	}
}

func absoluteNoSymlinks(path string) (string, error) {
	a, err := filepath.Abs(path)
	if err != nil {
		return a, err
	}

	return filepath.EvalSymlinks(a)
}

// WithScenarioBaseDirectory is base directory where the scenario defintions reside.
func WithScenarioBaseDirectory(dir string) Opt {
	return func(req *Generator) error {
		a, err := absoluteNoSymlinks(dir)
		if err != nil {
			return err
		}
		req.BaseDir = a
		return nil
	}
}

// WithScenario is the scenario to generate into a module
func WithScenario(s *flightplan.Scenario) Opt {
	return func(req *Generator) error {
		req.Scenario = s
		return nil
	}
}

// WithUI is the UI to use for outputing information
func WithUI(ui cli.Ui) Opt {
	return func(req *Generator) error {
		req.UI = ui
		return nil
	}
}

// Generate converts the Scenario into a terraform module
func (g *Generator) Generate() error {
	err := g.generateCLIConfig()
	if err != nil {
		return err
	}

	return g.generateModule()
}

// TerraformRCPath is where the generated terraform.rc configuration file will
// be written
func (g *Generator) TerraformRCPath() string {
	return filepath.Join(g.TerraformModuleDir(), "terraform.rc")
}

// TerraformModulePath is where the generated Terraform module file will
// be written
func (g *Generator) TerraformModulePath() string {
	return filepath.Join(g.TerraformModuleDir(), "scenario.tf")
}

// TerraformModuleDir is the directory where the generated Terraform
func (g *Generator) TerraformModuleDir() string {
	return filepath.Join(g.OutDir, g.Scenario.UID())
}

// generateCLIConfig converts the Scenario's terraform_cli into a terraformrc
// file.
func (g *Generator) generateCLIConfig() error {
	if g.Scenario.TerraformCLI == nil || g.Scenario.TerraformCLI.ConfigVal.IsNull() {
		return nil
	}

	rc := hclwrite.NewEmptyFile()
	body := rc.Body()

	blocks := 0
	for attr, val := range g.Scenario.TerraformCLI.ConfigVal.AsValueMap() {
		if blocks != 0 {
			body.AppendNewline()
		}
		blocks++

		switch attr {
		case "credentials", "credentials_helpers":
			for blkLabel, blkVal := range val.AsValueMap() {
				blk := body.AppendNewBlock(attr, []string{blkLabel})
				for blkAttr, blkAttrVal := range blkVal.AsValueMap() {
					if blkAttrVal.IsNull() {
						continue
					}
					blk.Body().SetAttributeValue(blkAttr, blkAttrVal)
				}
			}
		case "provider_installation":
			for _, pi := range val.AsValueSlice() {
				piBlocks := 0
				piblk := body.AppendNewBlock(attr, []string{})
				piBlks := pi.AsValueMap()

				// dev_overrides always needs to come first if it exists
				// so that it can properly bypass other methods.
				if devOverride, ok := piBlks["dev_overrides"]; ok {
					if !devOverride.IsNull() {
						blk := piblk.Body().AppendNewBlock("dev_overrides", []string{})
						for blkAttr, blkVal := range devOverride.AsValueMap() {
							blk.Body().AppendUnstructuredTokens(devOverridesTokens(blkAttr, blkVal))
						}
						piBlocks++
					}
				}

				for piBlkName, piBlkVal := range piBlks {
					switch piBlkName {
					case "dev_overrides":
						continue // we already handled it
					default:
						piBlocks++
						if piBlocks != 0 {
							piblk.Body().AppendNewline()
						}
						for _, blkVals := range piBlkVal.AsValueSlice() {
							blk := piblk.Body().AppendNewBlock(piBlkName, []string{})
							for blkAttr, blkVal := range blkVals.AsValueMap() {
								if blkVal.IsNull() {
									continue
								}
								blk.Body().SetAttributeValue(blkAttr, blkVal)
							}
						}
					}
				}
			}
		default:
			if !val.IsNull() {
				body.SetAttributeValue(attr, val)
			}
		}
	}

	// Make sure our out directory exists and is a directory
	err := g.ensureOutDir()
	if err != nil {
		return err
	}

	// Write our RC file to disk
	return g.write(g.TerraformRCPath(), rc.Bytes())
}

// generateModule converts a Scenario into an HCL Terraform module and writes
// it to a file in the OutDir.
func (g *Generator) generateModule() error {
	// Make sure our out directory exists and is a directory
	// NOTE: we have to do this before other steps since we'll be writing
	// things relative to this path.
	err := g.ensureOutDir()
	if err != nil {
		return err
	}

	mod := hclwrite.NewEmptyFile()
	modBody := mod.Body()

	// Write "terraform" settings block
	g.maybeWriteTerraformSettings(modBody)

	// Write provider level config
	g.maybeWriteProviderConfig(modBody)

	// Convert each step into a Terraform module
	err = g.convertStepsToModules(modBody)
	if err != nil {
		return err
	}

	// Write our outputs
	err = g.maybeWriteOutputs(modBody)
	if err != nil {
		return err
	}

	// Write our module to disk
	return g.write(g.TerraformModulePath(), mod.Bytes())
}

// maybeWriteTerraformSettings writes any configured "terraform" settings
// nolint:cyclop
func (g *Generator) maybeWriteTerraformSettings(rootBody *hclwrite.Body) {
	s := g.Scenario.TerraformSetting
	if s == nil {
		return
	}

	block := rootBody.AppendNewBlock("terraform", []string{})
	body := block.Body()

	if s.RequiredVersion != cty.NullVal(cty.String) {
		body.SetAttributeValue("required_version", s.RequiredVersion)
		body.AppendNewline()
	}

	if !s.Experiments.IsNull() && s.Experiments.IsWhollyKnown() {
		exps := []string{}
		for _, exp := range s.Experiments.AsValueSlice() {
			exps = append(exps, exp.AsString())
		}
		body.SetAttributeRaw("experiments", experimentsTokens(exps))
		body.AppendNewline()
	}

	if s.RequiredProviders != nil && len(s.RequiredProviders) > 0 {
		rpBlock := body.AppendNewBlock("required_providers", []string{})
		rpBody := rpBlock.Body()

		for k, v := range s.RequiredProviders {
			rpBody.SetAttributeValue(k, v)
		}
		body.AppendNewline()
	}

	if s.ProviderMetas != nil {
		for pmName, pmAttrs := range s.ProviderMetas {
			pmBlock := body.AppendNewBlock("provider_meta", []string{pmName})
			pmBody := pmBlock.Body()
			for k, v := range pmAttrs {
				pmBody.SetAttributeValue(k, v)
			}
			body.AppendNewline()
		}
	}

	if s.Backend != nil && s.Backend.Name != "" {
		beBlock := body.AppendNewBlock("backend", []string{s.Backend.Name})
		beBody := beBlock.Body()

		for k, v := range s.Backend.Attrs {
			beBody.SetAttributeValue(k, v)
		}
		if len(s.Backend.Attrs) > 0 {
			beBody.AppendNewline()
		}

		if !s.Backend.Workspaces.IsNull() && s.Backend.Workspaces.IsWhollyKnown() {
			for i, wksp := range s.Backend.Workspaces.AsValueSlice() {
				if i != 0 {
					beBody.AppendNewline()
				}
				wkspBlock := beBody.AppendNewBlock("workspaces", []string{})
				wkspBody := wkspBlock.Body()
				for k, v := range wksp.AsValueMap() {
					if !v.IsNull() && v.IsWhollyKnown() {
						wkspBody.SetAttributeValue(k, v)
					}
				}
			}
		}
	}

	if !s.Cloud.IsNull() && s.Cloud.IsWhollyKnown() {
		cloudsList, ok := s.Cloud.AsValueMap()["cloud"]
		if ok {
			if !cloudsList.IsNull() && cloudsList.IsWhollyKnown() && len(cloudsList.AsValueSlice()) > 0 {
				cloud := cloudsList.AsValueSlice()[0]
				cBlock := body.AppendNewBlock("cloud", nil)
				cBody := cBlock.Body()

				for k, v := range cloud.AsValueMap() {
					switch k {
					case "hostname", "organization", "token":
						cBody.SetAttributeValue(k, v)
					case "workspaces":
						for i, wksp := range v.AsValueSlice() {
							if i != 0 {
								cBody.AppendNewline()
							}
							wkspBlock := cBody.AppendNewBlock("workspaces", []string{})
							wkspBody := wkspBlock.Body()
							for wk, wv := range wksp.AsValueMap() {
								if !wv.IsNull() && wv.IsWhollyKnown() {
									wkspBody.SetAttributeValue(wk, wv)
								}
							}
						}
					default:
					}
				}
			}
		}
	}

	rootBody.AppendNewline()
}

func (g *Generator) maybeWriteProviderConfig(rootBody *hclwrite.Body) {
	if len(g.Scenario.Providers) == 0 {
		return
	}

	count := 0
	for _, provider := range g.Scenario.Providers {
		if count > 0 {
			rootBody.AppendNewline()
		}
		count++
		block := rootBody.AppendNewBlock("provider", []string{provider.Type})
		body := block.Body()
		for name, val := range provider.Attrs {
			body.SetAttributeValue(name, val)
		}
		if provider.Alias != "" {
			body.SetAttributeValue("alias", cty.StringVal(provider.Alias))
		}
	}

	rootBody.AppendNewline()
}

func (g *Generator) convertStepsToModules(rootBody *hclwrite.Body) error {
	// module for each step
	for i, step := range g.Scenario.Steps {
		block := rootBody.AppendNewBlock("module", []string{step.Name})
		body := block.Body()

		// depends_on the previous step
		if i != 0 {
			body.SetAttributeRaw("depends_on", dependsOnTokens(g.Scenario.Steps[i-1].Name))
		}

		// source
		src, err := maybeUpdateRelativeSourcePaths(
			step.Module.Source, g.BaseDir, g.TerraformModuleDir(),
		)
		if err != nil {
			return err
		}
		body.SetAttributeValue("source", cty.StringVal(src))

		// version
		if step.Module.Version != "" {
			body.SetAttributeValue("version", cty.StringVal(step.Module.Version))
		}

		// providers
		if len(step.Providers) > 0 {
			body.AppendNewline()
			body.SetAttributeRaw("providers", stepProviderTokens(step.Providers))
		}

		// variable attributes
		if len(step.Module.Attrs) > 0 {
			body.AppendNewline()
		}
		for k, v := range step.Module.Attrs {
			stepVar, diags := flightplan.StepVariableFromVal(v)
			if diags.HasErrors() {
				return fmt.Errorf(diags.Error())
			}

			// Use the absolute value
			if stepVar.Value != cty.NilVal {
				body.SetAttributeValue(k, stepVar.Value)
				continue
			}

			if stepVar.Traversal == nil {
				continue
			}

			// It's a module reference
			// Rename the root of the traversal to "module" and write it out
			err := stepToModuleTraversal(stepVar.Traversal)
			if err != nil {
				return err
			}
			body.SetAttributeTraversal(k, stepVar.Traversal)
		}

		if i+1 < len(g.Scenario.Steps) {
			rootBody.AppendNewline()
		}
	}

	return nil
}

func (g *Generator) maybeWriteOutputs(rootBody *hclwrite.Body) error {
	// Output value for each output
	for i, output := range g.Scenario.Outputs {
		if i == 0 {
			rootBody.AppendNewline()
		}

		block := rootBody.AppendNewBlock("output", []string{output.Name})
		body := block.Body()

		if output.Description != "" {
			body.SetAttributeValue("description", cty.StringVal(output.Description))
		}

		if output.Sensitive {
			body.SetAttributeValue("sensitive", cty.BoolVal(true))
		}

		writeOutput := func(output *flightplan.ScenarioOutput) error {
			if output.Value == cty.NilVal {
				return nil
			}

			stepVar, diags := flightplan.StepVariableFromVal(output.Value)
			if diags.HasErrors() {
				return fmt.Errorf(diags.Error())
			}

			// Use the absolute value if it exists
			if stepVar.Value != cty.NilVal {
				body.SetAttributeValue("value", stepVar.Value)
				return nil
			}

			if stepVar.Traversal != nil {
				err := stepToModuleTraversal(stepVar.Traversal)
				if err != nil {
					return err
				}
				body.SetAttributeTraversal("value", stepVar.Traversal)
			}

			return nil
		}

		err := writeOutput(output)
		if err != nil {
			return err
		}

		if i+1 < len(g.Scenario.Outputs) {
			rootBody.AppendNewline()
		}
	}

	return nil
}

func (g *Generator) ensureOutDir() error {
	d, err := os.Open(g.TerraformModuleDir())
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		if g.UI != nil {
			g.UI.Info(fmt.Sprintf("creating directory %s", g.TerraformModuleDir()))
		}

		return os.MkdirAll(g.TerraformModuleDir(), 0o755)
	}

	info, err := d.Stat()
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("out directory path (%s) is not a directory", g.TerraformModuleDir())
	}

	return nil
}

func (g *Generator) write(path string, bytes []byte) error {
	if g.UI != nil {
		g.UI.Info(fmt.Sprintf("writing to %s", path))
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(hclwrite.Format(bytes))
	return err
}

// experimentsTokens returns the terraform settings experiments tokens.
func experimentsTokens(experiments []string) hclwrite.Tokens {
	tokens := hclwrite.Tokens{
		&hclwrite.Token{
			Type:  hclsyntax.TokenOBrack,
			Bytes: []byte{'['},
		},
	}

	for i, exp := range experiments {
		if i > 0 {
			tokens = append(tokens, &hclwrite.Token{
				Type:  hclsyntax.TokenComma,
				Bytes: []byte{','},
			})
		}

		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte(exp),
		})
	}

	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenOBrack,
		Bytes: []byte{']'},
	})

	return tokens
}

func stepProviderTokens(providers map[string]*flightplan.Provider) hclwrite.Tokens {
	if len(providers) == 0 {
		return hclwrite.Tokens{
			&hclwrite.Token{
				Type:  hclsyntax.TokenEqual,
				Bytes: []byte{'='},
			},
			&hclwrite.Token{
				Type:  hclsyntax.TokenIdent,
				Bytes: []byte("null"),
			},
			&hclwrite.Token{
				Type:  hclsyntax.TokenNewline,
				Bytes: []byte{'\n'},
			},
		}
	}

	tokens := hclwrite.Tokens{
		&hclwrite.Token{
			Type:  hclsyntax.TokenOBrace,
			Bytes: []byte{'{'},
		},
		&hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte{'\n'},
		},
	}

	i := 0
	for importName, provider := range providers {
		if i > 0 {
			tokens = append(tokens,
				&hclwrite.Token{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte{'\n'},
				},
			)
		}
		i++

		tokens = append(tokens,
			&hclwrite.Token{
				Type:  hclsyntax.TokenIdent,
				Bytes: []byte(importName),
			},
			&hclwrite.Token{
				Type:  hclsyntax.TokenEqual,
				Bytes: []byte{'='},
			},
			&hclwrite.Token{
				Type:  hclsyntax.TokenIdent,
				Bytes: []byte(fmt.Sprintf("%s.%s", provider.Type, provider.Alias)),
			},
		)
	}

	tokens = append(tokens,
		&hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte{'\n'},
		},
		&hclwrite.Token{
			Type:  hclsyntax.TokenOBrace,
			Bytes: []byte{'}'},
		},
	)

	return tokens
}

// dependsOnTokens takes the name of module traversal target and returns the
// tokens necessary to write the HCL. We do this manually because hclwrite
// does not include a helper for converting cty.Values that contain an absolute
// traversal into an expression.
func dependsOnTokens(name string) hclwrite.Tokens {
	return hclwrite.Tokens{
		&hclwrite.Token{
			Type:  hclsyntax.TokenOBrack,
			Bytes: []byte{'['},
		},
		&hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("module"),
		},
		&hclwrite.Token{
			Type:  hclsyntax.TokenDot,
			Bytes: []byte{'.'},
		},
		&hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte(name),
		},
		&hclwrite.Token{
			Type:  hclsyntax.TokenOBrack,
			Bytes: []byte{']'},
		},
		&hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte{'\n'},
		},
	}
}

// devOverridesTokens creates the dev_overrides tokens as that stanza is not
// valid HCL2.
func devOverridesTokens(name string, val cty.Value) hclwrite.Tokens {
	tokens := hclwrite.Tokens{
		&hclwrite.Token{
			Type:  hclsyntax.TokenStringLit,
			Bytes: []byte(fmt.Sprintf(`"%s"`, name)),
		},
		&hclwrite.Token{
			Type:  hclsyntax.TokenEqual,
			Bytes: []byte{'='},
		},
	}

	tokens = append(tokens, hclwrite.TokensForValue(val)...)
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenNewline,
		Bytes: []byte{'\n'},
	})

	return tokens
}

// maybeUpdateRelativeSourcePaths is how we handle relative source paths in
// terraform modules. That is, Terraform "requires"[0] that module source
// paths that are local must begin with "./" or "../" so that the getter can
// distinguish it from a registry address. As we may be generating a root module
// in a different directory than the base directory, we need to dynamically
// update the relative source address so that it is relative from the generated
// module.
//
// [0]: https://www.terraform.io/docs/language/modules/sources.html#local-paths
func maybeUpdateRelativeSourcePaths(source, baseDir, outDir string) (string, error) {
	if filepath.IsAbs(source) {
		return filepath.EvalSymlinks(source)
	}

	// Probably not a filepath
	if !strings.HasPrefix(source, "./") && !strings.HasPrefix(source, "../") {
		return source, nil
	}

	return relativePath(outDir, filepath.Join(baseDir, source))
}

// relativePath builds the shortest relative path from one path to another. This
// function differs from the standard `filepath.Rel()` because it evaluates symlinks
// and returns the path with "./" or "../" prepended to adhere to the Terraform
// source path schema.
func relativePath(from, to string) (string, error) {
	var err error
	from, err = filepath.Abs(from)
	if err != nil {
		return "", err
	}

	to, err = filepath.Abs(to)
	if err != nil {
		return "", err
	}

	from, err = filepath.EvalSymlinks(from)
	if err != nil {
		return "", err
	}

	to, err = filepath.EvalSymlinks(to)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(from, to)
	if err != nil {
		return rel, err
	}

	if rel == "." {
		return "./", nil
	}

	if !strings.HasPrefix(rel, ".") {
		return fmt.Sprintf("./%s", rel), nil
	}

	return rel, nil
}

// stepToModuleTraversal takes a "step" traversal and updates the root of the
// the traversal to "module"
func stepToModuleTraversal(in hcl.Traversal) error {
	if len(in) == 0 {
		return nil
	}
	root, ok := in[0].(hcl.TraverseRoot)
	if !ok {
		return fmt.Errorf("malformed step variable reference")
	}
	root.Name = "module"
	in[0] = root

	return nil
}
