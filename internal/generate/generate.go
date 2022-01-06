package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/enos/internal/flightplan"
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
		a, err := filepath.Abs(dir)
		if err != nil {
			return err
		}
		req.OutDir = a
		return nil
	}
}

// WithScenarioBaseDirectory is base directory where the scenario defintions reside.
func WithScenarioBaseDirectory(dir string) Opt {
	return func(req *Generator) error {
		a, err := filepath.Abs(dir)
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
	return filepath.Join(g.OutDir, g.Scenario.Name)
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
					blk := piblk.Body().AppendNewBlock("dev_overrides", []string{})
					for blkAttr, blkVal := range devOverride.AsValueMap() {
						blk.Body().AppendUnstructuredTokens(devOverridesTokens(blkAttr, blkVal))
					}
					piBlocks++
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
						blk := piblk.Body().AppendNewBlock(piBlkName, []string{})
						for _, blkVals := range piBlkVal.AsValueSlice() {
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
	mod := hclwrite.NewEmptyFile()
	modBody := mod.Body()

	// Convert each step into a Terraform module
	err := g.convertStepsToModules(modBody)
	if err != nil {
		return err
	}

	// Make sure our out directory exists and is a directory
	err = g.ensureOutDir()
	if err != nil {
		return err
	}

	// Write our module to disk
	return g.write(g.TerraformModulePath(), mod.Bytes())
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

		// variable attributes
		if len(step.Module.Attrs) > 0 {
			body.AppendNewline()
		}
		for k, v := range step.Module.Attrs {
			body.SetAttributeValue(k, v)
		}

		if i+1 < len(g.Scenario.Steps) {
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
