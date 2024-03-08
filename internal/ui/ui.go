// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package ui

import (
	"errors"
	"io"

	"github.com/hashicorp/enos/internal/ui/basic"
	"github.com/hashicorp/enos/internal/ui/machine"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

var (
	_ View = (*basic.View)(nil)
	_ View = (*machine.View)(nil)
)

// View is a UI view. ShowX() methods are responsible for taking a command output
// response, displaying it appropriately, and exiting with an error if _any_
// error diagnostics are present in the response.
//
//nolint:interfacebloat // we have reason for a complex UI interface
type View interface {
	io.Closer
	Settings() *pb.UI_Settings
	ShowError(err error) error
	ShowDiagnostics(diags []*pb.Diagnostic) error
	ShowVersion(all bool, res *pb.GetVersionResponse) error
	ShowFormat(cfg *pb.FormatRequest_Config, res *pb.FormatResponse) error
	ShowScenarioList(res *pb.ListScenariosResponse) error
	ShowDecode(res *pb.DecodeResponse, incremental bool) error
	ShowOutput(res *pb.OperationResponses) error
	ShowScenariosValidateConfig(res *pb.ValidateScenariosConfigurationResponse) error
	ShowOperationEvent(res *pb.Operation_Event)
	ShowOperationResponse(res *pb.Operation_Response) error
	ShowOperationResponses(res *pb.OperationResponses) error
	ShowSampleList(res *pb.ListSamplesResponse) error
	ShowSampleObservation(res *pb.ObserveSampleResponse) error
}

// New takes a UI configuration settings and returns a new view.
func New(s *pb.UI_Settings) (View, error) {
	switch s.GetFormat() {
	case pb.UI_Settings_FORMAT_JSON:
		return machine.New(machine.WithUISettings(s))
	case pb.UI_Settings_FORMAT_BASIC_TEXT:
		return basic.New(basic.WithUISettings(s))
	case pb.UI_Settings_FORMAT_UNSPECIFIED:
		return basic.New(basic.WithUISettings(s))
	default:
		msg := "unsupported UI format"
		name, ok := pb.UI_Settings_Format_name[int32(s.GetFormat())]
		if ok {
			msg = name + " is not a supported UI format"
		}

		return nil, errors.New(msg)
	}
}
