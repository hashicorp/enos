// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package operation

import (
	"context"
	"fmt"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// terraformOutput renders the Terraform output.
func (r *Runner) terraformOutput(
	ctx context.Context,
	req *pb.Operation_Request,
	events *EventSender,
) *pb.Terraform_Command_Output_Response {
	res := &pb.Terraform_Command_Output_Response{
		Diagnostics: []*pb.Diagnostic{},
	}
	log := r.log.With(RequestDebugArgs(req)...)

	ref, err := NewReferenceFromRequest(req)
	if err != nil {
		res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(err)...)
		log.Error("failed to create reference from request", "error", err)

		return res
	}

	// Notify running output
	eventVal := &pb.Operation_Event_Output{}
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
		eventVal.Output = res

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

	// Configure our Terraform executor to use the module that should have
	// already been generated.
	module, diags := moduleForReq(ctx, req)
	if diagnostics.HasFailed(r.TFConfig.FailOnWarnings, diags) {
		notifyFail(diags)

		return res
	} else {
		res.Diagnostics = append(res.GetDiagnostics(), diags...)
	}

	r.TFConfig.WithModule(module)

	// terraform output
	outText := NewTextOutput()
	tf.SetStdout(outText.Stdout)
	tf.SetStderr(outText.Stderr)

	metas, err := tf.Output(ctx, r.TFConfig.OutputOptions()...)
	if err != nil {
		notifyFail(diagnostics.FromErr(err))

		return res
	}

	if r.TFConfig.OutputName != "" {
		meta, found := metas[r.TFConfig.OutputName]
		if !found {
			notifyFail(diagnostics.FromErr(fmt.Errorf("no output with key %s", r.TFConfig.OutputName)))

			return res
		}

		res.Meta = append(res.GetMeta(), &pb.Terraform_Command_Output_Response_Meta{
			Name:      r.TFConfig.OutputName,
			Type:      []byte(meta.Type),
			Value:     []byte(meta.Value),
			Sensitive: meta.Sensitive,
			Stderr:    outText.Stderr.String(),
		})
	} else {
		for name, meta := range metas {
			res.Meta = append(res.GetMeta(), &pb.Terraform_Command_Output_Response_Meta{
				Name:      name,
				Type:      []byte(meta.Type),
				Value:     []byte(meta.Value),
				Sensitive: meta.Sensitive,
				Stderr:    outText.Stderr.String(),
			})
		}
	}

	// Finalize our responses and event
	event.Status = diagnostics.Status(r.TFConfig.FailOnWarnings, res.GetDiagnostics()...)
	event.Diagnostics = res.GetDiagnostics()
	eventVal.Output = res

	// Notify that we've finished
	if err := events.Publish(event); err != nil {
		log.Error("failed to send event", "error", err)
		res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(err)...)
	}
	log.Debug("finished output")

	return res
}
