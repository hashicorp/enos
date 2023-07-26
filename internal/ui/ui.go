package ui

import (
	"fmt"
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
	ShowError(error) error
	ShowDiagnostics([]*pb.Diagnostic) error
	ShowVersion(all bool, res *pb.GetVersionResponse) error
	ShowFormat(*pb.FormatRequest_Config, *pb.FormatResponse) error
	ShowScenarioList(*pb.ListScenariosResponse) error
	ShowDecode(*pb.DecodeResponse, bool) error
	ShowOutput(*pb.OperationResponses) error
	ShowScenariosValidateConfig(*pb.ValidateScenariosConfigurationResponse) error
	ShowOperationEvent(*pb.Operation_Event)
	ShowOperationResponse(*pb.Operation_Response) error
	ShowOperationResponses(*pb.OperationResponses) error
}

// New takes a UI configuration settings and returns a new view.
func New(s *pb.UI_Settings) (View, error) {
	switch s.Format {
	case pb.UI_Settings_FORMAT_JSON:
		return machine.New(machine.WithUISettings(s))
	case pb.UI_Settings_FORMAT_BASIC_TEXT:
		return basic.New(basic.WithUISettings(s))
	case pb.UI_Settings_FORMAT_UNSPECIFIED:
		return basic.New(basic.WithUISettings(s))
	default:
		msg := "unsupported UI format"
		name, ok := pb.UI_Settings_Format_name[int32(s.Format)]
		if ok {
			msg = fmt.Sprintf("%s is not a supported UI format", name)
		}

		return nil, fmt.Errorf(msg)
	}
}
