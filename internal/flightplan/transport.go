package flightplan

import (
	"fmt"
	"path/filepath"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"

	hcl "github.com/hashicorp/hcl/v2"
)

// enosTransportSchema is the transport block top-level schema
var enosTransportSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "ssh", Required: false},
	},
}

// Transport is a Enos transport configuration
type Transport struct {
	Name string        `cty:"name" hcl:"name,label"`
	SSH  *TransportSSH `cty:"ssh" hcl:"ssh,optional"`
}

// TransportSSH is Enos transport ssh configuration
type TransportSSH struct {
	User           string `cty:"user" hcl:"user,optional"`
	Host           string `cty:"host" hcl:"host,optional"`
	PrivateKey     string `cty:"private_key" hcl:"private_key,optional"`
	PrivateKeyPath string `cty:"private_key_path" hcl:"private_key_path,optional"`
	Passphrase     string `cty:"passphrase" hcl:"passphrase,optional"`
	PassphrasePath string `cty:"passphrase_path" hcl:"passphrase_path,optional"`
}

// NewTransport returns a new Transport
func NewTransport() *Transport {
	return &Transport{
		SSH: &TransportSSH{},
	}
}

// decode takes in an HCL block of a transport and an eval context and
// decodes from the block onto itself. Any errors that are encountered are
// returned as hcl diagnostics.
// NOTE: we manually decode the attributes because we might need to expand the
// the private_key_path and passphrase_path and don't want to lose context in
// the diagnostics.
func (t *Transport) decode(block *hcl.Block, ctx *hcl.EvalContext) hcl.Diagnostics {
	var diags hcl.Diagnostics

	t.Name = block.Labels[0]
	content, moreDiags := block.Body.Content(enosTransportSchema)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	mapAttr := func(ctx *hcl.EvalContext, attr *hcl.Attribute, val cty.Value, dst *string) hcl.Diagnostics {
		var diags hcl.Diagnostics

		if val.IsNull() || !val.IsWhollyKnown() {
			return diags
		}

		if val.Type() != cty.String {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "value must be a string",
				Subject:  attr.Expr.Range().Ptr(),
				Context:  attr.Range.Ptr(),
			})

			return diags
		}
		*dst = val.AsString()

		return diags
	}

	ssh, ok := content.Attributes["ssh"]
	if !ok {
		return diags
	}

	sshVal, moreDiags := ssh.Expr.Value(ctx)
	diags = diags.Extend(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}
	for k, v := range sshVal.AsValueMap() {
		switch k {
		case "host":
			diags = diags.Extend(mapAttr(ctx, ssh, v, &t.SSH.Host))
		case "user":
			diags = diags.Extend(mapAttr(ctx, ssh, v, &t.SSH.User))
		case "private_key":
			diags = diags.Extend(mapAttr(ctx, ssh, v, &t.SSH.PrivateKey))
		case "private_key_path":
			diags = diags.Extend(mapAttr(ctx, ssh, v, &t.SSH.PrivateKeyPath))
			t.SSH.PrivateKeyPath, moreDiags = t.maybeExpandRelativePaths(ssh, t.SSH.PrivateKeyPath)
			diags = diags.Extend(moreDiags)
		case "passphrase":
			diags = diags.Extend(mapAttr(ctx, ssh, v, &t.SSH.Passphrase))
		case "passphrase_path":
			diags = diags.Extend(mapAttr(ctx, ssh, v, &t.SSH.PassphrasePath))
			t.SSH.PassphrasePath, moreDiags = t.maybeExpandRelativePaths(ssh, t.SSH.PassphrasePath)
			diags = diags.Extend(moreDiags)
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unsupported attribute",
				Detail:   fmt.Sprintf("'%s' is not a supported attribute", k),
				Subject:  ssh.NameRange.Ptr(),
				Context:  ssh.Range.Ptr(),
			})
		}
	}

	return diags
}

// maybeExpandRelativePaths takes a path and expands it. If an error is encountered
// when expanding a relative path an error diagnostic will be returned.
func (t *Transport) maybeExpandRelativePaths(attr *hcl.Attribute, path string) (string, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	abs, err := filepath.Abs(path)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "expanding path",
			Detail:   err.Error(),
			Subject:  attr.Expr.Range().Ptr(),
			Context:  attr.Range.Ptr(),
		})
	}

	return abs, diags
}

// evalCtx returns the transport contents as an object cty.Value. We can then
// embed this into the Variables section of the eval context to allowed method
// style expression references.
func (t *Transport) evalCtx() (cty.Value, error) {
	sshType := cty.Object(map[string]cty.Type{
		"user":             cty.String,
		"host":             cty.String,
		"private_key":      cty.String,
		"private_key_path": cty.String,
		"passphrase":       cty.String,
		"passphrase_path":  cty.String,
	})

	ssh, err := gocty.ToCtyValue(t.SSH, sshType)
	if err != nil {
		return cty.NullVal(sshType), err
	}

	vals := map[string]cty.Value{
		"name": cty.StringVal(t.Name),
		"ssh":  ssh,
	}

	return cty.ObjectVal(vals), err
}
