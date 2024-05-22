// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package operation

import (
	"errors"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/proto"
	"github.com/hashicorp/enos/pb/hashicorp/enos/v1"
	"github.com/hashicorp/go-multierror"
)

// RequestDebugArgs takes a reference to an operation request and returns a slice
// of arguments that can be passed ton an hclog.Logger instance for debugging.
func RequestDebugArgs(req *pb.Operation_Request) []any {
	if req == nil {
		return nil
	}
	args := []any{}

	scenario := flightplan.NewScenario()
	scenario.FromRef(req.GetScenario())
	if s := scenario.String(); s != "" {
		args = append(args, "scenario", s)
	}

	if opType := RequestTypeString(req); opType != "" {
		args = append(args, "operation_req", opType)
	}

	if i := req.GetId(); i != "" {
		args = append(args, "operation_id", i)
	}

	return args
}

// EventDebugArgs takes a reference to an operation event and returns a slice
// of arguments that can be passed ton an hclog.Logger instance for debugging.
func EventDebugArgs(event *pb.Operation_Event) []any {
	if event == nil {
		return nil
	}
	args := []any{}

	if ref := event.GetOp().GetScenario(); ref != nil {
		scenario := flightplan.NewScenario()
		scenario.FromRef(ref)
		if s := scenario.String(); s != "" {
			args = append(args, "scenario", s)
		}
	}

	if eventType := EventTypeString(event); eventType != "" {
		args = append(args, "sub_req", eventType)
	}

	if i := event.GetOp().GetId(); i != "" {
		args = append(args, "operation_id", i)
	}

	if s := event.GetStatus(); s != pb.Operation_STATUS_UNSPECIFIED {
		args = append(args, "status", pb.Operation_Status_name[int32(s)])
	}

	if event.GetDone() {
		args = append(args, "done", true)
	}

	if diags := event.GetDiagnostics(); diags != nil {
		for _, d := range diags {
			args = append(args, "diagnostic", diagnostics.String(d,
				diagnostics.WithStringSnippetEnabled(false),
			))
		}
	}

	return args
}

// ResponseDebugArgs takes a reference to an operation response and returns a slice
// of arguments that can be passed ton an hclog.Logger instance for debugging.
func ResponseDebugArgs(res *pb.Operation_Response) []any {
	if res == nil {
		return nil
	}
	args := []any{}

	if ref := res.GetOp().GetScenario(); ref != nil {
		scenario := flightplan.NewScenario()
		scenario.FromRef(ref)
		if s := scenario.String(); s != "" {
			args = append(args, "scenario", s)
		}
	}

	if opType := ResponseTypeString(res); opType != "" {
		args = append(args, "operation_req", opType)
	}

	if i := res.GetOp().GetId(); i != "" {
		args = append(args, "operation_id", i)
	}

	if s := res.GetStatus(); s != pb.Operation_STATUS_UNSPECIFIED {
		args = append(args, "status", pb.Operation_Status_name[int32(s)])
	}

	if diags := res.GetDiagnostics(); diags != nil {
		for _, d := range diags {
			args = append(args, "diagnostic", diagnostics.String(d,
				diagnostics.WithStringSnippetEnabled(false),
			))
		}
	}

	return args
}

// ReferenceDebugArgs takes a reference to an operation reference and returns a slice
// of arguments that can be passed ton an hclog.Logger instance for debugging.
func ReferenceDebugArgs(ref *pb.Ref_Operation) []any {
	if ref == nil {
		return nil
	}
	args := []any{}

	if sref := ref.GetScenario(); sref != nil {
		scenario := flightplan.NewScenario()
		scenario.FromRef(sref)
		if s := scenario.String(); s != "" {
			args = append(args, "scenario", s)
		}
	}

	if i := ref.GetId(); i != "" {
		args = append(args, "operation_id", i)
	}

	return args
}

// RequestTypeString takes a reference to an operation request and returns the type
// of request as a string.
func RequestTypeString(op *pb.Operation_Request) string {
	if op == nil {
		return ""
	}

	switch op.GetValue().(type) {
	case *pb.Operation_Request_Generate_:
		return "generate"
	case *pb.Operation_Request_Check_:
		return "check"
	case *pb.Operation_Request_Launch_:
		return "launch"
	case *pb.Operation_Request_Destroy_:
		return "destroy"
	case *pb.Operation_Request_Run_:
		return "run"
	case *pb.Operation_Request_Exec_:
		return "exec"
	case *pb.Operation_Request_Output_:
		return "output"
	default:
		return "unknown"
	}
}

// EventTypeString takes a reference to an operation event and returns the type
// of event as a string.
func EventTypeString(event *pb.Operation_Event) string {
	if event == nil {
		return ""
	}

	switch event.GetValue().(type) {
	case *pb.Operation_Event_Decode:
		return "enos_decode"
	case *pb.Operation_Event_Generate:
		return "enos_generate_module"
	case *pb.Operation_Event_Init:
		return "terraform_init"
	case *pb.Operation_Event_Validate:
		return "terraform_validate"
	case *pb.Operation_Event_Plan:
		return "terraform_plan"
	case *pb.Operation_Event_Apply:
		return "terraform_apply"
	case *pb.Operation_Event_Destroy:
		return "terraform_destroy"
	case *pb.Operation_Event_Exec:
		return "terraform_exec"
	case *pb.Operation_Event_Output:
		return "terraform_output"
	default:
		return "unknown"
	}
}

// ResponseTypeString takes a reference to an operation response and returns the type
// of response as a string.
func ResponseTypeString(op *pb.Operation_Response) string {
	if op == nil {
		return ""
	}

	switch op.GetValue().(type) {
	case *pb.Operation_Response_Generate_:
		return "generate"
	case *pb.Operation_Response_Check_:
		return "check"
	case *pb.Operation_Response_Launch_:
		return "launch"
	case *pb.Operation_Response_Destroy_:
		return "destroy"
	case *pb.Operation_Response_Run_:
		return "run"
	case *pb.Operation_Response_Exec_:
		return "exec"
	case *pb.Operation_Response_Output_:
		return "output"
	default:
		return "unknown"
	}
}

//nolint:unparam // right now all callers use pb.Operation_STATUS_RUNNING but that's not guaranteed.
func newEvent(
	ref *pb.Ref_Operation,
	status pb.Operation_Status,
) *pb.Operation_Event {
	return &pb.Operation_Event{
		Op:     ref,
		Status: status,
	}
}

// NewEventFromResponse takes a reference to an operation response and returns a
// reference to a new operation event. If an error is encountered it will return
// an event that contains diagnostics of the error as well as an error if possible.
func NewEventFromResponse(res *pb.Operation_Response) (*pb.Operation_Event, error) {
	merr := &multierror.Error{}

	event := &pb.Operation_Event{
		Op:          res.GetOp(),
		Diagnostics: res.GetDiagnostics(),
		Status:      res.GetStatus(),
	}

	switch t := res.GetValue().(type) {
	case *pb.Operation_Response_Generate_:
		event.Value = &pb.Operation_Event_Generate{
			Generate: res.GetGenerate(),
		}
	case *pb.Operation_Response_Check_:
		// For operations responses that may include multiple sub operations, well
		// always need to check for them in reverse order so our event always has
		// the most up-to-date information.
		if p := t.Check.GetPlan(); p != nil {
			event.Value = &pb.Operation_Event_Plan{
				Plan: p,
			}
		} else if v := t.Check.GetValidate(); v != nil {
			event.Value = &pb.Operation_Event_Validate{
				Validate: v,
			}
		} else if i := t.Check.GetInit(); i != nil {
			event.Value = &pb.Operation_Event_Init{
				Init: i,
			}
		}
	case *pb.Operation_Response_Launch_:
		event.Value = &pb.Operation_Event_Apply{
			Apply: t.Launch.GetApply(),
		}
	case *pb.Operation_Response_Destroy_:
		event.Value = &pb.Operation_Event_Destroy{
			Destroy: t.Destroy.GetDestroy(),
		}
	case *pb.Operation_Response_Run_:
	case *pb.Operation_Response_Exec_:
		event.Value = &pb.Operation_Event_Exec{
			Exec: t.Exec.GetExec(),
		}
	case *pb.Operation_Response_Output_:
		event.Value = &pb.Operation_Event_Output{
			Output: t.Output.GetOutput(),
		}
	default:
		err := errors.New("cannot convert response type to event")
		merr = multierror.Append(merr, err)
		event.Diagnostics = append(
			event.GetDiagnostics(),
			diagnostics.FromErr(err)...,
		)
		event.Status = pb.Operation_STATUS_FAILED
	}

	newEvent := &pb.Operation_Event{}
	err := proto.Copy(event, newEvent)
	merr = multierror.Append(merr, err)
	if err != nil {
		newEvent.Status = pb.Operation_STATUS_FAILED
		newEvent.Diagnostics = append(
			newEvent.GetDiagnostics(),
			diagnostics.FromErr(err)...,
		)
	}

	return newEvent, merr.ErrorOrNil()
}

// NewResponseFromRequest takes a reference for an operation request and returns a reference
// to a new operation response.
func NewResponseFromRequest(op *pb.Operation_Request) (*pb.Operation_Response, error) {
	merr := &multierror.Error{}

	res := &pb.Operation_Response{
		Op: &pb.Ref_Operation{
			Id:       op.GetId(),
			Scenario: op.GetScenario(),
		},
	}

	switch op.GetValue().(type) {
	case *pb.Operation_Request_Generate_:
		res.Value = &pb.Operation_Response_Generate_{}
	case *pb.Operation_Request_Check_:
		res.Value = &pb.Operation_Response_Check_{}
	case *pb.Operation_Request_Launch_:
		res.Value = &pb.Operation_Response_Launch_{}
	case *pb.Operation_Request_Destroy_:
		res.Value = &pb.Operation_Response_Destroy_{}
	case *pb.Operation_Request_Run_:
		res.Value = &pb.Operation_Response_Run_{}
	case *pb.Operation_Request_Exec_:
		res.Value = &pb.Operation_Response_Exec_{}
	case *pb.Operation_Request_Output_:
		res.Value = &pb.Operation_Response_Output_{}
	default:
		err := errors.New("cannot convert response type to event")
		merr = multierror.Append(merr, err)
		res.Diagnostics = append(
			res.GetDiagnostics(),
			diagnostics.FromErr(err)...,
		)
		res.Status = pb.Operation_STATUS_FAILED
	}

	newRes := &pb.Operation_Response{}
	err := proto.Copy(res, newRes)
	merr = multierror.Append(merr, err)
	if err != nil {
		newRes.Status = pb.Operation_STATUS_FAILED
		res.Diagnostics = append(
			res.GetDiagnostics(),
			diagnostics.FromErr(err)...,
		)
	}

	return newRes, merr.ErrorOrNil()
}

// NewReferenceFromRequest takes a reference to an operation request and returns
// a new reference to the associated operation.
func NewReferenceFromRequest(op *pb.Operation_Request) (*pb.Ref_Operation, error) {
	ref := &pb.Ref_Operation{}

	err := proto.Copy(&pb.Ref_Operation{
		Id:       op.GetId(),
		Scenario: op.GetScenario(),
	}, ref)

	return ref, err
}

func hasFailedStatus(s pb.Operation_Status) bool {
	switch s {
	case pb.Operation_STATUS_CANCELLED, pb.Operation_STATUS_FAILED:
		return true
	case pb.Operation_STATUS_UNSPECIFIED, pb.Operation_STATUS_UNKNOWN, pb.Operation_STATUS_QUEUED, pb.Operation_STATUS_WAITING, pb.Operation_STATUS_RUNNING, pb.Operation_STATUS_RUNNING_WARNING, pb.Operation_STATUS_COMPLETED, pb.Operation_STATUS_COMPLETED_WARNING:
		return false
	default:
		return false
	}
}
