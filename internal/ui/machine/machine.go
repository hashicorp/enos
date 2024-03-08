// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package machine

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/ui/status"
	"github.com/hashicorp/enos/internal/ui/terminal"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// View is our basic terminal CLI view.
type View struct {
	settings *pb.UI_Settings
	stderr   io.ReadWriteCloser
	stdout   io.ReadWriteCloser
	ui       *terminal.UI
}

type errJSON struct {
	Errors []string `json:"errors"`
}

type diagsJSON struct {
	Diags []json.RawMessage `json:"diagnostics"`
}

// Opt is a functional option.
type Opt func(*View)

// NewErrUnsupportedEncodingFormat returns a new unsupported encoding format
// error.
func NewErrUnsupportedEncodingFormat(format pb.UI_Settings_Format) error {
	friendlyName, ok := pb.UI_Settings_Format_name[int32(format)]
	msg := "unsupported output format type"
	if !ok {
		msg = fmt.Sprintf("%s: %s", msg, friendlyName)
	}

	return errors.New(msg)
}

// New takes options and returns a new basic.View.
func New(opts ...Opt) (*View, error) {
	v := &View{}

	for _, opt := range opts {
		opt(v)
	}

	if v.settings.GetFormat() != pb.UI_Settings_FORMAT_JSON {
		return nil, NewErrUnsupportedEncodingFormat(v.settings.GetFormat())
	}

	uiOpts := []terminal.Opt{
		terminal.WithStdin(os.Stdin),
		terminal.WithStdout(os.Stdout),
		terminal.WithStderr(os.Stderr),
	}
	if v.settings.GetStdoutPath() != "" {
		f, err := os.OpenFile(v.settings.GetStdoutPath(), os.O_RDWR|os.O_CREATE, 0o755)
		if err != nil {
			return nil, err
		}
		v.stdout = f

		uiOpts = append(uiOpts, terminal.WithStdout(f))
	}

	if v.settings.GetStderrPath() != "" {
		f, err := os.OpenFile(v.settings.GetStderrPath(), os.O_RDWR|os.O_CREATE, 0o755)
		if err != nil {
			return nil, err
		}
		v.stderr = f

		uiOpts = append(uiOpts, terminal.WithStderr(f))
	}

	v.ui = terminal.NewUI(uiOpts...)

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
	if v.stderr != nil {
		err := v.stderr.Close()
		if err != nil {
			return err
		}
	}

	if v.stdout == nil {
		return nil
	}

	err := v.stdout.Close()
	if err != nil {
		return err
	}

	return nil
}

// ShowFormat shows the output of a format request.
func (v *View) ShowFormat(cfg *pb.FormatRequest_Config, res *pb.FormatResponse) error {
	if err := v.write(res); err != nil {
		return err
	}

	return status.Format(cfg, res)
}

// ShowError writes the given error to stdout in the formatted version.
func (v *View) ShowError(err error) error {
	return v.writeError(err)
}

// ShowDiagnostics writes the given diagnostic to stdout in the formatted version.
func (v *View) ShowDiagnostics(diags []*pb.Diagnostic) error {
	return v.writeDiagnostics(diags)
}

// ShowVersion shows the version information.
func (v *View) ShowVersion(all bool, res *pb.GetVersionResponse) error {
	if err := v.write(res); err != nil {
		return err
	}

	return status.GetVersion(res)
}

// ShowScenariosValidateConfig shows the validation response.
func (v *View) ShowScenariosValidateConfig(res *pb.ValidateScenariosConfigurationResponse) error {
	if err := v.write(res); err != nil {
		return err
	}

	return status.ScenariosValidateConfig(v.settings.GetFailOnWarnings(), res)
}

// ShowScenarioList shows the a list of scenarios.
func (v *View) ShowScenarioList(res *pb.ListScenariosResponse) error {
	if err := v.write(res); err != nil {
		return err
	}

	return status.ListScenarios(v.settings.GetFailOnWarnings(), res)
}

// ShowSampleList shows the a list of samples.
func (v *View) ShowSampleList(res *pb.ListSamplesResponse) error {
	if err := v.write(res); err != nil {
		return err
	}

	return status.ListSamples(v.settings.GetFailOnWarnings(), res)
}

// ShowSample shows the sample observation.
func (v *View) ShowSampleObservation(res *pb.ObserveSampleResponse) error {
	if err := v.write(res); err != nil {
		return err
	}

	return status.ShowSampleObservation(v.settings.GetFailOnWarnings(), res)
}

// ShowDecode shows the decode response unless it's a incremental update.
func (v *View) ShowDecode(res *pb.DecodeResponse, incremental bool) error {
	if incremental {
		// machine output doesn't show incremental update so we early return
		return nil
	}

	if err := v.write(res); err != nil {
		return err
	}

	return status.Decode(v.Settings().GetFailOnWarnings(), res)
}

// ShowOutput shows output response.
func (v *View) ShowOutput(out *pb.OperationResponses) error {
	if err := v.write(out); err != nil {
		return err
	}

	return status.OperationResponses(v.Settings().GetFailOnWarnings(), out)
}

// ShowOperationEvent does nothing as the machine output doesn't stream events.
func (v *View) ShowOperationEvent(*pb.Operation_Event) {
}

// ShowOperationResponse shows an operation response.
func (v *View) ShowOperationResponse(res *pb.Operation_Response) error {
	if err := v.write(res); err != nil {
		return err
	}

	return status.OperationResponses(
		v.settings.GetFailOnWarnings(),
		&pb.OperationResponses{
			Responses: []*pb.Operation_Response{
				res,
			},
		},
	)
}

// ShowOperationResponses shows the results of multiple operations.
func (v *View) ShowOperationResponses(res *pb.OperationResponses) error {
	if err := v.write(res); err != nil {
		return err
	}

	return status.OperationResponses(v.settings.GetFailOnWarnings(), res)
}

// writeError does our best to write the given error to our stderr.
func (v *View) writeError(err error) error {
	tryJSON := func(err error) error {
		msg := &errJSON{
			Errors: []string{err.Error()},
		}

		bytes, err2 := json.Marshal(msg)
		if err2 != nil {
			return fmt.Errorf("%w: %s", err, err2.Error())
		}

		_, err2 = v.ui.Stderr.Write(bytes)
		if err2 != nil {
			return fmt.Errorf("%w: %s", err, err2.Error())
		}

		return nil
	}

	tryPlainText := func(err error) error {
		_, err2 := v.ui.Stderr.Write([]byte(err.Error()))
		if err2 != nil {
			return fmt.Errorf("%w: %s", err, err2.Error())
		}

		return nil
	}

	switch v.settings.GetFormat() {
	case pb.UI_Settings_FORMAT_JSON:
		err := tryJSON(err)
		if err != nil {
			return tryPlainText(err)
		}

		return nil
	case pb.UI_Settings_FORMAT_UNSPECIFIED, pb.UI_Settings_FORMAT_BASIC_TEXT:
		err := tryJSON(fmt.Errorf("%w: %s", err, NewErrUnsupportedEncodingFormat(v.settings.GetFormat()).Error()))
		if err != nil {
			return tryPlainText(err)
		}

		return nil
	default:
		err := tryJSON(fmt.Errorf("%w: %s", err, NewErrUnsupportedEncodingFormat(v.settings.GetFormat()).Error()))
		if err != nil {
			return tryPlainText(err)
		}

		return nil
	}
}

// writeDiagnostics does our best to write the diagnostics to our stderr.
func (v *View) writeDiagnostics(diags []*pb.Diagnostic) error {
	tryJSON := func(diags []*pb.Diagnostic) error {
		msg := &diagsJSON{
			Diags: []json.RawMessage{},
		}

		for _, diag := range diags {
			bytes, err := protojson.Marshal(diag)
			if err != nil {
				msg.Diags = append(msg.Diags, []byte(err.Error()))

				continue
			}
			msg.Diags = append(msg.Diags, bytes)
		}

		bytes, err := json.Marshal(msg)
		if err != nil {
			_, _ = v.ui.Stderr.Write([]byte(err.Error()))

			return err
		}

		_, err = v.ui.Stderr.Write(bytes)

		return err
	}

	tryPlainText := func(err error) error {
		_, err2 := v.ui.Stderr.Write([]byte(err.Error()))
		if err2 != nil {
			return fmt.Errorf("%w: %s", err, err2.Error())
		}

		return nil
	}

	switch v.settings.GetFormat() {
	case pb.UI_Settings_FORMAT_JSON:
		err := tryJSON(diags)
		if err != nil {
			return tryPlainText(err)
		}

		return nil
	case pb.UI_Settings_FORMAT_UNSPECIFIED, pb.UI_Settings_FORMAT_BASIC_TEXT:
		diags = append(diags, diagnostics.FromErr(
			NewErrUnsupportedEncodingFormat(v.settings.GetFormat()),
		)...)
		err := tryJSON(diags)
		if err != nil {
			return tryPlainText(err)
		}

		return nil
	default:
		diags = append(diags, diagnostics.FromErr(
			NewErrUnsupportedEncodingFormat(v.settings.GetFormat()),
		)...)
		err := tryJSON(diags)
		if err != nil {
			return tryPlainText(err)
		}

		return nil
	}
}

// write takes a proto messages and writes it to the desired output.
func (v *View) write(msg proto.Message) error {
	var err error
	var bytes []byte

	switch v.settings.GetFormat() {
	case pb.UI_Settings_FORMAT_JSON:
		bytes, err = protojson.Marshal(msg)
		if err != nil {
			return err
		}
	case pb.UI_Settings_FORMAT_UNSPECIFIED, pb.UI_Settings_FORMAT_BASIC_TEXT:
		return NewErrUnsupportedEncodingFormat(v.settings.GetFormat())
	default:
		return NewErrUnsupportedEncodingFormat(v.settings.GetFormat())
	}

	_, err = v.ui.Stdout.Write(bytes)

	return err
}
