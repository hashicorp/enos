// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package operation

import (
	"context"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	tfjson "github.com/hashicorp/terraform-json"
)

// terraformDestroy destroys resources created by the Terraform module.
func (r *Runner) terraformDestroy(
	ctx context.Context,
	req *pb.Operation_Request,
	events *EventSender,
	state *tfjson.State,
) *pb.Terraform_Command_Destroy_Response {
	res := &pb.Terraform_Command_Destroy_Response{
		Diagnostics: []*pb.Diagnostic{},
	}
	log := r.log.With(RequestDebugArgs(req)...)

	ref, err := NewReferenceFromRequest(req)
	if err != nil {
		res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(err)...)
		log.Error("failed to create reference from request", "error", err)

		return res
	}

	// Notify running destroy
	eventVal := &pb.Operation_Event_Destroy{}
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
		eventVal.Destroy = res

		if err := events.Publish(event); err != nil {
			res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(err)...)
			log.Error("failed to publish event", "error", err)
		}
	}

	// Determine if we have deletable state. If we don't this action is a no-op
	// and we can terminate early.
	if !hasDeleteableState(state) {
		// We don't have deletable state, we can return.
		log.Debug("skipping delete because state file contained no deletable values")

		// Finalize our event
		event.Status = diagnostics.Status(r.TFConfig.FailOnWarnings, res.GetDiagnostics()...)
		event.Diagnostics = res.GetDiagnostics()
		eventVal.Destroy = res

		// Notify that we've finished
		if err := events.Publish(event); err != nil {
			log.Error("failed to send event", "error", err)
			res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(err)...)
		}
		log.Debug("finished destroy")

		return res
	}

	// Create our terraform executor
	tf, err := r.TFConfig.Terraform()
	if err != nil {
		notifyFail(diagnostics.FromErr(err))

		return res
	}

	destroyOut := NewTextOutput()
	tf.SetStdout(destroyOut.Stdout)
	tf.SetStderr(destroyOut.Stderr)
	err = tf.Destroy(ctx, r.TFConfig.DestroyOptions()...)
	res.Stderr = destroyOut.Stderr.String()
	if err != nil {
		notifyFail(diagnostics.FromErr(err))

		return res
	}

	// Finalize our responses and event
	event.Status = diagnostics.Status(r.TFConfig.FailOnWarnings, res.GetDiagnostics()...)
	event.Diagnostics = res.GetDiagnostics()
	eventVal.Destroy = res

	// Notify that we've finished
	if err := events.Publish(event); err != nil {
		log.Error("failed to send event", "error", err)
		res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(err)...)
	}
	log.Debug("finished destroy")

	return res
}

// hasDeleteableState inspects the JSON representation of a Terraform state and
// returns a boolean of whether or not the state contains deletable values.
func hasDeleteableState(state *tfjson.State) bool {
	if state == nil {
		return false
	}

	if state.Values == nil {
		return false
	}

	// We've seen Terraform occasionally leave behind a partial set of outputs
	// after a module has been destroyed. As such, we can't on their presence
	// to detect whether or not we have deleteable state.
	/*
		if state.Values.Outputs != nil {
			// If outputs are present we have applied but not destroyed
			if len(state.Values.Outputs) > 0 {
				return true
			}
		}
	*/

	if state.Values.RootModule == nil {
		return false
	}

	if state.Values.RootModule.Resources != nil {
		// If our root module has resources we have state
		if len(state.Values.RootModule.Resources) > 0 {
			return true
		}
	}

	if state.Values.RootModule.ChildModules != nil {
		for i := range state.Values.RootModule.ChildModules {
			// If any child module has resources we have state
			if len(state.Values.RootModule.ChildModules[i].Resources) > 0 {
				return true
			}
		}
	}

	return false
}
