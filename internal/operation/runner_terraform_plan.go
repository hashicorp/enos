// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package operation

import (
	"context"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// terraformPlan plans a Terraform module.
func (r *Runner) terraformPlan(
	ctx context.Context,
	req *pb.Operation_Request,
	events *EventSender,
) *pb.Terraform_Command_Plan_Response {
	res := &pb.Terraform_Command_Plan_Response{
		Diagnostics: []*pb.Diagnostic{},
	}
	log := r.log.With(RequestDebugArgs(req)...)

	ref, err := NewReferenceFromRequest(req)
	if err != nil {
		res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(err)...)
		log.Error("failed to create reference from request", "error", err)

		return res
	}

	// Notify running init
	eventVal := &pb.Operation_Event_Plan{}
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
		eventVal.Plan = res

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

	// terraform plan
	planOut := NewTextOutput()
	tf.SetStdout(planOut.Stdout)
	tf.SetStderr(planOut.Stderr)
	changes, err := tf.Plan(ctx, r.TFConfig.PlanOptions()...)
	res.ChangesPresent = changes
	res.Stderr = planOut.Stderr.String()
	if err != nil {
		notifyFail(diagnostics.FromErr(err))

		return res
	}

	// Finalize our responses and event
	event.Status = diagnostics.Status(r.TFConfig.FailOnWarnings, res.GetDiagnostics()...)
	event.Diagnostics = res.GetDiagnostics()
	eventVal.Plan = res

	// Notify that we've finished
	if err := events.Publish(event); err != nil {
		log.Error("failed to send event", "error", err)
		res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(err)...)
	}
	log.Debug("finished plan")

	return res
}
