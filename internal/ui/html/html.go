// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package html

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"strings"

	"github.com/hashicorp/enos/internal/ui/basic"
	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

//go:embed template/*
var templates embed.FS

// View is our html view. At the current time it only implements a limited set of the interface to
// allow for writing scenarios outlines as HTML, and the CLI should only ever be configured with
// html from that command. If the html view is given to other commands it will return an error
// through the basic CLI view.
type View struct {
	basic    *basic.View
	settings *pb.UI_Settings
}

// Opt is a functional option.
type Opt func(*View)

// New takes options and returns a new html.View.
func New(opts ...Opt) (*View, error) {
	v := &View{}
	for _, opt := range opts {
		opt(v)
	}
	basic, err := basic.New(basic.WithUISettings(v.settings))
	if err != nil {
		return nil, err
	}
	v.basic = basic

	return v, nil
}

// WithUISettings configures the view with the UI settings.
func WithUISettings(settings *pb.UI_Settings) Opt {
	return func(view *View) {
		view.settings = settings
	}
}

// Settings returns the views UI settings.
func (v *View) Settings() *pb.UI_Settings {
	return v.settings
}

// Close closes any open file handles.
func (v *View) Close() error {
	if v == nil || v.basic == nil {
		return nil
	}

	return v.basic.Close()
}

// ShowFormat shows the output of a format request.
func (v *View) ShowFormat(cfg *pb.FormatRequest_Config, res *pb.FormatResponse) error {
	return v.ShowError(status.Unimplemented("html/ui: ShowFormat"))
}

// ShowError writes the given error to stdout in the formatted version.
func (v *View) ShowError(err error) error {
	return v.basic.ShowError(err)
}

// ShowDiagnostics writes the given diagnostic to stdout in the formatted version.
func (v *View) ShowDiagnostics(diags []*pb.Diagnostic) error {
	return v.basic.ShowDiagnostics(diags)
}

// ShowVersion shows the version information.
func (v *View) ShowVersion(all bool, res *pb.GetVersionResponse) error {
	return v.ShowError(status.Unimplemented("html/ui: ShowVersion"))
}

// ShowScenariosValidateConfig shows the validation response.
func (v *View) ShowScenariosValidateConfig(res *pb.ValidateScenariosConfigurationResponse) error {
	return v.ShowError(status.Unimplemented("html/ui: ShowScenariosValidateConfig"))
}

// ShowScenarioList shows the a list of scenarios.
func (v *View) ShowScenarioList(res *pb.ListScenariosResponse) error {
	return v.ShowError(status.Unimplemented("html/ui: ShowScenarioList"))
}

// ShowScenarioOutline shows the scenario outlines.
func (v *View) ShowScenarioOutline(res *pb.OutlineScenariosResponse) error {
	if res == nil || res.GetOutlines() == nil || len(res.GetOutlines()) < 1 {
		return nil
	}

	t, err := template.New("outline.html.tmpl").Funcs(template.FuncMap{
		"scenarioName": func(g *pb.Scenario_Outline) string {
			if g == nil {
				return ""
			}

			return g.GetScenario().GetId().GetName()
		},
		"scenarioDescription": func(g *pb.Scenario_Outline) string {
			if g == nil {
				return ""
			}

			return g.GetScenario().GetId().GetDescription()
		},
		"scenarioVariants": func(g *pb.Scenario_Outline) []*pb.Matrix_Vector {
			if g == nil {
				return nil
			}

			return g.GetMatrix().GetVectors()
		},
		"scenarioSteps": func(g *pb.Scenario_Outline) []*pb.Scenario_Outline_Step {
			if g == nil {
				return nil
			}

			return g.GetSteps()
		},
		"scenarioVerifies": func(g *pb.Scenario_Outline) []*pb.Quality {
			if g == nil {
				return nil
			}

			return g.GetVerifies()
		},
		"scenarioStepVerifies": func(g *pb.Scenario_Outline_Step) []*pb.Quality {
			if g == nil {
				return nil
			}

			return g.GetVerifies()
		},
		"printVariantKey": func(vec *pb.Matrix_Vector) string {
			if vec == nil || vec.GetElements() == nil || len(vec.GetElements()) < 1 {
				return ""
			}

			return vec.GetElements()[0].GetKey()
		},
		"printVariantValues": func(vec *pb.Matrix_Vector) string {
			if vec == nil || vec.GetElements() == nil || len(vec.GetElements()) < 1 {
				return ""
			}
			b := new(strings.Builder)
			for i := range vec.GetElements() {
				if i != 0 {
					fmt.Fprint(b, ", ")
				}
				fmt.Fprintf(b, "%s", vec.GetElements()[i].GetValue())
			}

			return b.String()
		},
	}).ParseFS(templates, "template/outline.html.tmpl")
	if err != nil {
		return v.ShowError(err)
	}

	buf := bytes.Buffer{}
	err = t.Execute(&buf, res)
	if err != nil {
		return v.ShowError(err)
	}

	v.basic.UI().Output(buf.String())

	return status.OutlineScenarios(v.settings.GetFailOnWarnings(), res)
}

// ShowSampleList shows the a list of samples.
func (v *View) ShowSampleList(res *pb.ListSamplesResponse) error {
	return v.ShowError(status.Unimplemented("html/ui: ShowSampleList"))
}

// ShowSample shows the sample observation.
func (v *View) ShowSampleObservation(res *pb.ObserveSampleResponse) error {
	return v.ShowError(status.Unimplemented("html/ui: ShowSampleObservation"))
}

// ShowDecode shows the decode response unless it's a incremental update.
func (v *View) ShowDecode(res *pb.DecodeResponse, incremental bool) error {
	return v.basic.ShowDecode(res, incremental)
}

// ShowOutput shows output response.
func (v *View) ShowOutput(out *pb.OperationResponses) error {
	return v.ShowError(status.Unimplemented("html/ui: ShowOutput"))
}

// ShowOperationEvent does nothing as the html output doesn't stream events.
func (v *View) ShowOperationEvent(*pb.Operation_Event) {
}

// ShowOperationResponse shows an operation response.
func (v *View) ShowOperationResponse(res *pb.Operation_Response) error {
	return v.ShowError(status.Unimplemented("html/ui: ShowOperationResponse"))
}

// ShowOperationResponses shows the results of multiple operations.
func (v *View) ShowOperationResponses(res *pb.OperationResponses) error {
	return v.ShowError(status.Unimplemented("html/ui: ShowOperationResponses"))
}
