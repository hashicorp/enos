package basic

import (
	"fmt"
	"strings"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// ShowOperationEvent shows the human friendly view on an operation event
func (v *View) ShowOperationEvent(event *pb.Operation_Event) {
	if event == nil {
		return
	}

	if event.Done {
		// We don't show done events as they should have already been
		// reported.
		return
	}

	msg := new(strings.Builder)

	switch event.GetValue().(type) {
	case *pb.Operation_Event_Decode:
		v.writeEventDecode(event, msg)
	case *pb.Operation_Event_Generate:
		v.writeEventGenerate(event, msg)
	case *pb.Operation_Event_Init:
		v.writeEventInit(event, msg)
	case *pb.Operation_Event_Validate:
		v.writeEventValidate(event, msg)
	case *pb.Operation_Event_Plan:
		v.writeEventPlan(event, msg)
	case *pb.Operation_Event_Apply:
		v.writeEventApply(event, msg)
	case *pb.Operation_Event_Destroy:
		v.writeEventDestroy(event, msg)
	case *pb.Operation_Event_Exec:
		v.writeEventExec(event, msg)
	case *pb.Operation_Event_Show:
		v.writeEventShow(event, msg)
	case *pb.Operation_Event_Output:
		// Don't display output events by default since outputs have their own
		// view.
		if v.settings.Level == pb.UI_Settings_LEVEL_TRACE {
			v.writeEventOutput(event, msg)
		}
	default:
		event.Diagnostics = append(event.GetDiagnostics(), diagnostics.FromErr(
			fmt.Errorf("unable to handle event type. This is a bug in Enos. Please report it. %v", event),
		)...)
	}

	v.writeMsg(event.GetStatus(), msg.String())
}
