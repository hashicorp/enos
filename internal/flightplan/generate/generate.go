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

// WithOutDirectory is the destination directory where modules will be written.
func WithOutDirectory(dir string) Opt {
	return func(req *Generator) error {
		a, err := filepath.Abs(dir)
		if err != nil {
			return err
		}
		req.OutDir = a
		return nil
	}
}

// WithBaseDirectory is base directory where the scenario defintions reside.
func WithBaseDirectory(dir string) Opt {
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
	return g.generateModule()
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
	return g.writeModule(mod.Bytes())
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
			step.Module.Source, g.BaseDir, g.OutDir,
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
	d, err := os.Open(g.OutDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		if g.UI != nil {
			g.UI.Info(fmt.Sprintf("creating directory %s", g.OutDir))
		}

		return os.MkdirAll(g.OutDir, 0o755)
	}

	info, err := d.Stat()
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("out directory path (%s) is not a directory", g.OutDir)
	}

	return nil
}

func (g *Generator) writeModule(bytes []byte) error {
	path := filepath.Join(g.OutDir, g.Scenario.Name+".tf")
	if g.UI != nil {
		g.UI.Info(fmt.Sprintf("writing module to %s", path))
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
		return source, nil
	}

	if !strings.HasPrefix(source, "./") && !strings.HasPrefix(source, "../") {
		return source, nil
	}

	return relativePath(outDir, filepath.Join(baseDir, source))
}

// relativePath builds the shortest relative path from one path to another.
// This function differs from `filepath.Rel()` in that it will produce a Terraform
// friendly path that will alwasy start with `./` or `./..`, wherease the
// `filepath.Rel()` passes all results through `filepath.Clean()`, which can
// produce absolute paths.
func relativePath(from, to string) (string, error) {
	sepS := string(os.PathSeparator)

	fromAbs, err := filepath.Abs(from)
	if err != nil {
		return "", err
	}
	toAbs, err := filepath.Abs(to)
	if err != nil {
		return "", err
	}

	// They're the same directory our relative path is "./"
	if to == from {
		return "./", nil
	}

	// Find the most recent directory ancestor
	commonDirs := 0
	fromParts := strings.Split(fromAbs, sepS)
	toParts := strings.Split(toAbs, sepS)
	for i, d := range fromParts {
		if toParts[i] != d {
			break
		}
		commonDirs++
	}

	// Determine how many steps from our from path to our most recent
	// ancestor
	fromStepsBack := len(fromParts) - commonDirs
	if fromStepsBack < 0 {
		return "", fmt.Errorf("cannot step back past root")
	}

	// Generate our relative from path step back
	var fromRel string
	switch fromStepsBack {
	case 0:
		// NOTE: we don't use filepath join on "./" because it calls Clean which
		// removes the prefix.
		return fmt.Sprintf(".%s%s", sepS, strings.Join(toParts[commonDirs:], sepS)), nil
	default:
		for i := 0; i < fromStepsBack; i++ {
			fromRel = fmt.Sprintf("%s../", fromRel)
		}
		return filepath.Join(fromRel, strings.Join(toParts[commonDirs:], sepS)), nil
	}
}
