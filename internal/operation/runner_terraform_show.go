// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package operation

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/enos/internal/diagnostics"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// terraformShow returns gets the Terraform State for a module.
func (r *Runner) terraformShow(
	ctx context.Context,
	req *pb.Operation_Request,
	events *EventSender,
) *pb.Terraform_Command_Show_Response {
	res := &pb.Terraform_Command_Show_Response{
		Diagnostics: []*pb.Diagnostic{},
	}
	log := r.log.With(RequestDebugArgs(req)...)

	ref, err := NewReferenceFromRequest(req)
	if err != nil {
		res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(err)...)
		log.Error("failed to create reference from request", "error", err)

		return res
	}

	// Notify running show
	eventVal := &pb.Operation_Event_Show{}
	event := newEvent(ref, pb.Operation_STATUS_RUNNING)
	event.Value = eventVal
	if err = events.Publish(event); err != nil {
		res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(err)...)
		log.Error("failed to publish event", "error", err)
	}

	// notifyFail prepares the response for failure and sends a failure
	// event
	notifyFail := func(diags []*pb.Diagnostic) {
		event.Status = pb.Operation_STATUS_FAILED
		res.Diagnostics = append(res.GetDiagnostics(), diags...)
		event.Diagnostics = append(event.GetDiagnostics(), res.GetDiagnostics()...)
		eventVal.Show = res

		if err := events.Publish(event); err != nil {
			res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(err)...)
			log.Error("failed to publish event", "error", err)
		}
	}

	// Create our terraform executor
	tf, err := r.TFConfig.Terraform()
	if err != nil {
		notifyFail(diagnostics.FromErr(err))

		return res
	}

	// Run show and save the output to our state
	showOut := NewTextOutput()
	tf.SetStdout(showOut.Stdout)
	tf.SetStderr(showOut.Stderr)
	state, err := tf.Show(ctx, r.TFConfig.ShowOptions()...)
	if err != nil {
		notifyFail(diagnostics.FromErr(err))

		return res
	}

	stateEnc, err := json.Marshal(state)
	if err != nil {
		notifyFail(diagnostics.FromErr(err))

		return res
	}

	res.State = stateEnc

	// Finalize our responses and event
	event.Status = diagnostics.Status(r.TFConfig.FailOnWarnings, res.GetDiagnostics()...)
	event.Diagnostics = res.GetDiagnostics()
	eventVal.Show = res

	// Notify that we've finished
	if err := events.Publish(event); err != nil {
		log.Error("failed to send event", "error", err)
		res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(err)...)
	}
	log.Debug("finished show")

	return res
}
