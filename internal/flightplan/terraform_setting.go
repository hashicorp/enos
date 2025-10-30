// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"errors"
	"fmt"

	"github.com/zclconf/go-cty/cty"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
)

// TerraformSetting is a terraform settings configuration.
type TerraformSetting struct {
	Name            string
	RequiredVersion cty.Value
	Experiments     cty.Value
	// name, object with source and version
	RequiredProviders map[string]cty.Value
	// name, map of attribute names and values
	ProviderMetas map[string]map[string]cty.Value
	Backend       *TerraformSettingBackend
	Cloud         cty.Value
}

// TerraformSettingBackend is the "backend".
type TerraformSettingBackend struct {
	Name       string
	Attrs      map[string]cty.Value
	Workspaces cty.Value
}

// NewTerraformSetting returns a new TerraformSetting.
func NewTerraformSetting() *TerraformSetting {
	return &TerraformSetting{
		RequiredVersion:   cty.NullVal(cty.String),
		Experiments:       cty.NullVal(cty.List(cty.String)),
		RequiredProviders: map[string]cty.Value{},
		ProviderMetas:     map[string]map[string]cty.Value{},
		Backend:           nil,
		Cloud:             cty.NullVal(cty.EmptyObject),
	}
}

// NewTerraformSettingBackend returns a new TerraformSettingBackend.
func NewTerraformSettingBackend() *TerraformSettingBackend {
	return &TerraformSettingBackend{
		Attrs:      map[string]cty.Value{},
		Workspaces: cty.NullVal(cty.EmptyObject),
	}
}

// decode takes in an HCL block of a terraform and an eval context and
// decodes from the block onto itself. Any errors that are encountered are
// returned as hcl diagnostics. As we don't directly use the values of the settings
// in enos, nor do we need to modify them, they're usually decoded and left as
// cty.Values so that we can pass them directly to the generator.
func (t *TerraformSetting) decode(block *hcl.Block, ctx *hcl.EvalContext) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	t.Name = block.Labels[0]

	diags = diags.Extend(t.ensureOnlyCloudOrBackendDefined(block.Body))

	// Handle our known schema and remove them from the body
	remain, moreDiags := t.decodeRequiredVersion(ctx, block.Body)
	diags = diags.Extend(moreDiags)
	remain, moreDiags = t.decodeExperiments(ctx, remain)
	diags = diags.Extend(moreDiags)
	remain, moreDiags = t.decodeCloud(ctx, remain)
	diags = diags.Extend(moreDiags)

	if remain == nil {
		return diags
	}

	// Handle the rest of our schema manually since it isn't strictly defined
	content, moreDiags := remain.Content(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: blockTypeRequiredProviders},
			{Type: blockTypeProviderMeta, LabelNames: []string{attrLabelNameDefault}},
			{Type: blockTypeBackend, LabelNames: []string{attrLabelNameDefault}},
		},
	})
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return diags
	}

	diags = diags.Extend(t.decodeRequiredProviders(ctx, content))
	diags = diags.Extend(t.decodeProviderMeta(ctx, content))
	diags = diags.Extend(t.decodeBackend(ctx, content))

	return diags
}

// ensureOnlyCloudOrBackendDefined ensures that only a cloud or backend is defined.
func (t *TerraformSetting) ensureOnlyCloudOrBackendDefined(body hcl.Body) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	content, _, moreDiags := body.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: blockTypeCloud},
			{Type: blockTypeBackend, LabelNames: []string{attrLabelNameDefault}},
		},
	})
	diags = diags.Extend(moreDiags)
	if moreDiags != nil && moreDiags.HasErrors() {
		return diags
	}

	clouds := content.Blocks.OfType(blockTypeCloud)
	backends := content.Blocks.OfType(blockTypeBackend)

	if len(clouds) > 0 && len(backends) > 0 {
		for _, cloud := range clouds {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "incompatible block",
				Detail:   "cloud blocks are not supported if backend blocks have been defined",
				Subject:  cloud.TypeRange.Ptr(),
				Context:  cloud.DefRange.Ptr(),
			})
		}

		for _, backend := range backends {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "incompatible block",
				Detail:   "backend blocks are not supported if a cloud block has been defined",
				Subject:  backend.TypeRange.Ptr(),
				Context:  backend.DefRange.Ptr(),
			})
		}
	}

	return diags
}

// decodeRequiredVersion decodes the "required_version" attribute.
func (t *TerraformSetting) decodeRequiredVersion(ctx *hcl.EvalContext, body hcl.Body) (hcl.Body, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	var remain hcl.Body

	t.RequiredVersion, remain, diags = hcldec.PartialDecode(
		body, &hcldec.AttrSpec{
			Name:     "required_version",
			Type:     cty.String,
			Required: false,
		}, ctx,
	)

	return remain, diags
}

// decodeExperiments decodes the "experiments" attribute. NOTE: that in enos we
// strictly support input as HCL, therefore the special Terraform language experiments
// have to be converted from literals to strings.
func (t *TerraformSetting) decodeExperiments(ctx *hcl.EvalContext, body hcl.Body) (hcl.Body, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	var remain hcl.Body

	t.Experiments, remain, diags = hcldec.PartialDecode(
		body, &hcldec.AttrSpec{
			Name:     "experiments",
			Type:     cty.List(cty.String),
			Required: false,
		}, ctx,
	)

	return remain, diags
}

// decodeCloud decodes the "cloud" block.
func (t *TerraformSetting) decodeCloud(ctx *hcl.EvalContext, body hcl.Body) (hcl.Body, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	var remain hcl.Body

	t.Cloud, remain, diags = hcldec.PartialDecode(
		body, hcldec.ObjectSpec{
			"cloud": &hcldec.BlockListSpec{
				TypeName: "cloud",
				Nested: hcldec.ObjectSpec{
					"organization": &hcldec.AttrSpec{
						Name:     "organization",
						Type:     cty.String,
						Required: true,
					},
					"hostname": &hcldec.AttrSpec{
						Name:     "hostname",
						Type:     cty.String,
						Required: false,
					},
					"token": &hcldec.AttrSpec{
						Name:     "token",
						Type:     cty.String,
						Required: false,
					},
					"workspaces": &hcldec.BlockListSpec{
						TypeName: "workspaces",
						Nested: hcldec.ObjectSpec{
							"name": &hcldec.AttrSpec{
								Name:     "name",
								Type:     cty.String,
								Required: false,
							},
							"tags": &hcldec.AttrSpec{
								Name:     "tags",
								Type:     cty.List(cty.String),
								Required: false,
							},
						},
					},
				},
			},
		}, ctx,
	)

	return remain, diags
}

// decodeRequiredProviders decodes the "required_providers" block.
func (t *TerraformSetting) decodeRequiredProviders(ctx *hcl.EvalContext, content *hcl.BodyContent) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	for _, block := range content.Blocks.OfType("required_providers") {
		diags = diags.Extend(verifyNoBlockInAttrOnlySchema(block.Body))

		attrs, moreDiags := block.Body.JustAttributes()
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		rp := map[string]cty.Value{}
		for name, attr := range attrs {
			val, moreDiags := attr.Expr.Value(ctx)
			diags = diags.Extend(moreDiags)
			if moreDiags != nil && moreDiags.HasErrors() {
				continue
			}

			if val.IsNull() || !val.IsWhollyKnown() {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "unknown attribute value",
					Detail:   attr.Name + " required_providers value is not fully known",
					Subject:  attr.Expr.Range().Ptr(),
					Context:  attr.Range.Ptr(),
				})

				continue
			}

			if !val.CanIterateElements() {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "unexpected attribute value",
					Detail:   attr.Name + " value must be an object",
					Subject:  attr.Expr.Range().Ptr(),
					Context:  attr.Range.Ptr(),
				})

				continue
			}

			for attrName := range val.AsValueMap() {
				switch attrName {
				case "source", "version":
				default:
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "unexpected attribute",
						Detail:   attr.Name + " is not an allowed required_providers attribute",
						Subject:  attr.Expr.Range().Ptr(),
						Context:  attr.Range.Ptr(),
					})

					continue
				}
			}
			rp[name] = val
		}

		t.RequiredProviders = rp
	}

	return diags
}

// decodeProviderMeta decodes the "provider_meta" block.
func (t *TerraformSetting) decodeProviderMeta(ctx *hcl.EvalContext, content *hcl.BodyContent) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	for _, block := range content.Blocks.OfType("provider_meta") {
		diags = diags.Extend(verifyNoBlockInAttrOnlySchema(block.Body))

		attrs, moreDiags := block.Body.JustAttributes()
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}

		pm := map[string]cty.Value{}
		for _, attr := range attrs {
			val, moreDiags := attr.Expr.Value(ctx)
			diags = diags.Extend(moreDiags)
			if moreDiags != nil && moreDiags.HasErrors() {
				continue
			}
			pm[attr.Name] = val
		}

		t.ProviderMetas[block.Labels[0]] = pm
	}

	return diags
}

// decodeBackend decodes the "backend" block.
func (t *TerraformSetting) decodeBackend(ctx *hcl.EvalContext, content *hcl.BodyContent) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	for i, block := range content.Blocks.OfType("backend") {
		if i != 0 {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "redefined block",
				Detail:   "only one backend block is allowed to be defined",
				Subject:  block.TypeRange.Ptr(),
				Context:  block.DefRange.Ptr(),
			})

			continue
		}

		backend := &TerraformSettingBackend{
			Name:  block.Labels[0],
			Attrs: map[string]cty.Value{},
		}

		val, remain, moreDiags := hcldec.PartialDecode(
			block.Body, hcldec.ObjectSpec{
				"workspaces": &hcldec.BlockListSpec{
					TypeName: "workspaces",
					Nested: hcldec.ObjectSpec{
						"name": &hcldec.AttrSpec{
							Name:     "name",
							Type:     cty.String,
							Required: false,
						},
						"prefix": &hcldec.AttrSpec{
							Name:     "prefix",
							Type:     cty.String,
							Required: false,
						},
					},
				},
			}, ctx,
		)
		diags = diags.Extend(moreDiags)
		if moreDiags != nil && moreDiags.HasErrors() {
			continue
		}
		backend.Workspaces = val.AsValueMap()["workspaces"]

		// NOTE: JustAttributes() will raise an error if the body has a block
		// defined. At the time of writing even "hidden" blocks from partial
		// decoding are not exempt, hence we're ignoring the error.
		if remain != nil {
			attrs, _ := remain.JustAttributes()
			for _, attr := range attrs {
				val, moreDiags := attr.Expr.Value(ctx)
				diags = diags.Extend(moreDiags)
				if moreDiags != nil && moreDiags.HasErrors() {
					continue
				}
				backend.Attrs[attr.Name] = val
			}
		}

		t.Backend = backend
	}

	return diags
}

// FromCtyValue takes a cty.Value and unmasharls the value onto itself. Any
// errors that are encountered are returned. It is expected that the cty.Value
// is the cty.Value in the eval context.
func (t *TerraformSetting) FromCtyValue(val cty.Value) error {
	if !val.CanIterateElements() {
		return errors.New("you must provide a cty.Value which can interate elements")
	}

	for k, v := range val.AsValueMap() {
		switch k {
		case "name":
			if v.Type() != cty.String {
				return fmt.Errorf("name type is %s, must be string", v.Type().FriendlyName())
			}
			t.Name = v.AsString()
		case "required_version":
			t.RequiredVersion = v
		case "experiments":
			t.Experiments = v
		case "cloud":
			t.Cloud = v
		case "required_providers":
			if v.IsNull() {
				continue
			}

			if !v.CanIterateElements() {
				return errors.New("required_providers must provide a cty.Value which can interate elements")
			}

			for rpName, rpValue := range v.AsValueMap() {
				t.RequiredProviders[rpName] = rpValue
			}
		case "provider_meta":
			if v.IsNull() {
				continue
			}

			if !v.CanIterateElements() {
				return errors.New("provider_meta must provide a cty.Value which can interate elements")
			}

			for pmName, pmAttrs := range v.AsValueMap() {
				if !pmAttrs.CanIterateElements() {
					return errors.New("provider_meta attributes must provide a cty.Value which can interate elements")
				}

				t.ProviderMetas[pmName] = pmAttrs.AsValueMap()
			}
		case "backend":
			t.Backend = NewTerraformSettingBackend()
			if v.IsNull() {
				continue
			}

			if !v.CanIterateElements() {
				return errors.New("provider_meta must provide a cty.Value which can interate elements")
			}

			for beK, beV := range v.AsValueMap() {
				switch beK {
				case "name":
					if beV.Type() != cty.String {
						return errors.New("provider_meta name must be a string")
					}
					if !beV.IsWhollyKnown() {
						return errors.New("backend name attribute must be known")
					}
					t.Backend.Name = beV.AsString()
				case "workspaces":
					t.Backend.Workspaces = beV
				default:
					t.Backend.Attrs[beK] = beV
				}
			}
		default:
			return fmt.Errorf("%s is not a known terraform setting", k)
		}
	}

	return nil
}

// ToCtyValue returns the terraform contents as an object cty.Value. We can then
// embed this into the Variables section of the eval context to allowed method
// style expression references.
func (t *TerraformSetting) ToCtyValue() cty.Value {
	vals := map[string]cty.Value{
		"name":             cty.StringVal(t.Name),
		"required_version": t.RequiredVersion,
		"experiments":      t.Experiments,
		"cloud":            t.Cloud,
	}

	if len(t.RequiredProviders) == 0 {
		vals["required_providers"] = cty.NullVal(cty.Object(map[string]cty.Type{
			"source":  cty.String,
			"version": cty.String,
		}))
	} else {
		vals["required_providers"] = cty.ObjectVal(t.RequiredProviders)
	}

	if len(t.ProviderMetas) == 0 {
		vals["provider_meta"] = cty.NullVal(cty.EmptyObject)
	} else {
		metas := map[string]cty.Value{}
		for name, attrs := range t.ProviderMetas {
			metas[name] = cty.ObjectVal(attrs)
		}
		vals["provider_meta"] = cty.ObjectVal(metas)
	}

	if t.Backend == nil {
		vals["backend"] = cty.NullVal(cty.EmptyObject)
	} else {
		backend := map[string]cty.Value{
			"name":       cty.StringVal(t.Backend.Name),
			"workspaces": t.Backend.Workspaces,
		}

		for attr, val := range t.Backend.Attrs {
			backend[attr] = val
		}

		vals["backend"] = cty.ObjectVal(backend)
	}

	return cty.ObjectVal(vals)
}
