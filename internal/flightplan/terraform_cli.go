package flightplan

import (
	"os/exec"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
)

// terraformCLISpec represets a Terraform CLI config. Our intention is for
// it to map exactly to Terraform's CLI configuration. Since we don't actually
// care about the values we'll use hcldec to keep it in the CTY type system
// so that we can easily write it back to HCL when we render the config for
// the scenario.
var terraformCLISpec = hcldec.ObjectSpec{
	"disable_checkpoint": &hcldec.AttrSpec{
		Name:     "disable_checkpoint",
		Type:     cty.Bool,
		Required: false,
	},
	"disable_checkpoint_signature": &hcldec.AttrSpec{
		Name:     "disable_checkpoint_signature",
		Type:     cty.Bool,
		Required: false,
	},
	"plugin_cache_dir": &hcldec.AttrSpec{
		Name:     "plugin_cache_dir",
		Type:     cty.String,
		Required: false,
	},
	"credentials": &hcldec.BlockMapSpec{
		TypeName:   "credentials",
		LabelNames: []string{attrLabelNameDefault},
		Nested: hcldec.ObjectSpec{
			"token": &hcldec.AttrSpec{
				Name:     "token",
				Type:     cty.String,
				Required: false,
			},
		},
	},
	"credentials_helper": &hcldec.BlockMapSpec{
		TypeName:   "credentials_helper",
		LabelNames: []string{attrLabelNameDefault},
		Nested: hcldec.ObjectSpec{
			"args": &hcldec.AttrSpec{
				Name:     "args",
				Type:     cty.List(cty.String),
				Required: false,
			},
		},
	},
	"provider_installation": &hcldec.BlockListSpec{
		TypeName: "provider_installation",
		Nested: hcldec.ObjectSpec{
			"dev_overrides": &hcldec.AttrSpec{
				Name:     "dev_overrides",
				Type:     cty.Map(cty.String),
				Required: false,
			},
			"direct": &hcldec.BlockListSpec{
				TypeName: "direct",
				Nested: hcldec.ObjectSpec{
					"include": &hcldec.AttrSpec{
						Name:     "include",
						Type:     cty.List(cty.String),
						Required: false,
					},
					"exclude": &hcldec.AttrSpec{
						Name:     "exclude",
						Type:     cty.List(cty.String),
						Required: false,
					},
				},
			},
			"filesystem_mirror": &hcldec.BlockListSpec{
				TypeName: "filesystem_mirror",
				Nested: hcldec.ObjectSpec{
					"path": &hcldec.AttrSpec{
						Name:     "path",
						Type:     cty.String,
						Required: false,
					},
					"include": &hcldec.AttrSpec{
						Name:     "include",
						Type:     cty.List(cty.String),
						Required: false,
					},
					"exclude": &hcldec.AttrSpec{
						Name:     "exclude",
						Type:     cty.List(cty.String),
						Required: false,
					},
				},
			},
			"network_mirror": &hcldec.BlockListSpec{
				TypeName: "network_mirror",
				Nested: hcldec.ObjectSpec{
					"url": &hcldec.AttrSpec{
						Name:     "url",
						Type:     cty.String,
						Required: false,
					},
					"include": &hcldec.AttrSpec{
						Name:     "include",
						Type:     cty.List(cty.String),
						Required: false,
					},
					"exclude": &hcldec.AttrSpec{
						Name:     "exclude",
						Type:     cty.List(cty.String),
						Required: false,
					},
				},
			},
		},
	},
}

// terraformCLISchema are the pieces of terraform CLI configuration we care about.
var terraformCLISchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "path"},
		{Name: "env"},
	},
}

// TerraformCLI is a Terraform CLI configuration.
type TerraformCLI struct {
	Name      string            `cty:"name"   hcl:"name"`
	Path      string            `cty:"path"   hcl:"path,optional"`
	Env       map[string]string `cty:"env"    hcl:"env,optional"`
	ConfigVal cty.Value         `cty:"config" hcl:"config,optional"`
}

// NewTerraformCLI returns a new TerraformCLI.
func NewTerraformCLI() *TerraformCLI {
	return &TerraformCLI{
		Env:       map[string]string{},
		ConfigVal: cty.NilVal,
	}
}

// DefaultTerraformCLI returns a "default" Terraform CLI that attempts to resolve
// terraform from the system PATH.
func DefaultTerraformCLI() *TerraformCLI {
	cli := NewTerraformCLI()
	cli.Env = nil
	cli.Name = "default"
	cli.ConfigVal = cty.NilVal
	cli.Path, _ = exec.LookPath("terraform")

	return cli
}

// decode takes in an HCL block of a terraform_cli and an eval context and
// decodes from the block onto itself. Any errors that are encountered are
// returned as hcl diagnostics.
func (m *TerraformCLI) decode(block *hcl.Block, ctx *hcl.EvalContext) hcl.Diagnostics {
	var diags hcl.Diagnostics

	content, remain, moreDiags := block.Body.PartialContent(terraformCLISchema)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	m.Name = block.Labels[0]

	path, ok := content.Attributes["path"]
	if ok {
		val, moreDiags := path.Expr.Value(ctx)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			return diags
		}

		if val.Type() == cty.String {
			m.Path = val.AsString()
		} else {
			pathVal, err := convert.Convert(val, cty.String)
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "invalid value",
					Detail:   "terraform_cli path must be a string value",
					Subject:  path.Expr.Range().Ptr(),
					Context:  hcl.RangeBetween(path.Expr.StartRange(), path.Expr.Range()).Ptr(),
				})
			} else {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  "invalid value",
					Detail:   "terraform_cli should be a string value, consider changing it",
					Subject:  path.Expr.Range().Ptr(),
					Context:  hcl.RangeBetween(path.Expr.StartRange(), path.Expr.Range()).Ptr(),
				})
				m.Path = pathVal.AsString()
			}
		}
	}

	env, ok := content.Attributes["env"]
	if ok {
		val, moreDiags := env.Expr.Value(ctx)
		diags = diags.Extend(moreDiags)
		if moreDiags.HasErrors() {
			return diags
		}
		for k, v := range val.AsValueMap() {
			if v.Type() == cty.String {
				m.Env[k] = v.AsString()
			} else {
				vVal, err := convert.Convert(v, cty.String)
				if err != nil {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "invalid value",
						Detail:   "terraform_cli env must be a map of strings",
						Subject:  env.Range.Ptr(),
						Context:  hcl.RangeBetween(env.Expr.StartRange(), env.Expr.Range()).Ptr(),
					})
				} else {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagWarning,
						Summary:  "invalid value",
						Detail:   "terraform_cli env should be a map of strings, consider changing them",
						Subject:  env.Range.Ptr(),
						Context:  hcl.RangeBetween(env.Expr.StartRange(), env.Expr.Range()).Ptr(),
					})
					m.Env[k] = vVal.AsString()
				}
			}
		}
	}

	// The remaining portion of our HCL body is our Terraform CLI config. We'll
	// evaluate it and keep it around so that we can write it out as HCL during
	// execution.
	m.ConfigVal, moreDiags = hcldec.Decode(remain, terraformCLISpec, ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	return diags
}

// evalCtx returns the terraform_cli contents as an object cty.Value. We can then
// embed this into the Variables section of the eval context to allowed method
// style expression references.
func (m *TerraformCLI) evalCtx() cty.Value {
	vals := map[string]cty.Value{
		"name":   cty.StringVal(m.Name),
		"path":   cty.StringVal(m.Path),
		"config": m.ConfigVal,
	}

	envVals := map[string]cty.Value{}
	for k, v := range m.Env {
		envVals[k] = cty.StringVal(v)
	}
	if len(envVals) > 0 {
		vals["env"] = cty.MapVal(envVals)
	} else {
		vals["env"] = cty.NullVal(cty.Map(cty.String))
	}

	return cty.ObjectVal(vals)
}
