package operation

import (
	"context"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// terraformDestroy destroys resources created by the Terraform module
func (e *Runner) terraformDestroy(
	ctx context.Context,
	req *pb.Operation_Request,
	events *EventSender,
) *pb.Terraform_Command_Destroy_Response {
	res := &pb.Terraform_Command_Destroy_Response{
		Diagnostics: []*pb.Diagnostic{},
	}

	ref, err := NewReferenceFromRequest(req)
	if err != nil {
		res.Diagnostics = append(res.Diagnostics, diagnostics.FromErr(err)...)
		e.log.Error("failed to create reference from request", "error", err)
		return res
	}

	// Notify running destroy
	eventVal := &pb.Operation_Event_Destroy{}
	event := newEvent(ref, pb.Operation_STATUS_RUNNING)
	event.Value = eventVal
	if err = events.Publish(event); err != nil {
		res.Diagnostics = append(res.Diagnostics, diagnostics.FromErr(err)...)
		e.log.Error("failed to publish event", "error", err)
	}

	// notifyFail prepares the response for failure and sends a failure
	// event
	notifyFail := func(diags []*pb.Diagnostic) {
		event.Status = pb.Operation_STATUS_FAILED
		res.Diagnostics = append(res.Diagnostics, diags...)
		event.Diagnostics = append(event.Diagnostics, res.GetDiagnostics()...)
		eventVal.Destroy = res

		if err := events.Publish(event); err != nil {
			res.Diagnostics = append(res.Diagnostics, diagnostics.FromErr(err)...)
			e.log.Error("failed to publish event", "error", err)
		}
	}

	// Create our terraform executor
	tf, err := e.TFConfig.Terraform()
	if err != nil {
		notifyFail(diagnostics.FromErr(err))
		return res
	}

	destroyOut := NewTextOutput()
	tf.SetStdout(destroyOut.Stdout)
	tf.SetStderr(destroyOut.Stderr)
	err = tf.Destroy(ctx, e.TFConfig.DestroyOptions()...)
	res.Diagnostics = diagnostics.FromErr(err)
	res.Stderr = destroyOut.Stderr.String()
	if err != nil {
		notifyFail(diagnostics.FromErr(err))
		return res
	}

	// Finalize our responses and event
	event.Status = diagnostics.Status(e.TFConfig.FailOnWarnings, res.GetDiagnostics()...)
	event.Diagnostics = res.Diagnostics
	eventVal.Destroy = res

	// Notify that we've finished
	if err := events.Publish(event); err != nil {
		e.log.Error("failed to send event", "error", err)
		res.Diagnostics = append(res.Diagnostics, diagnostics.FromErr(err)...)
	}
	e.log.Debug("finished destroy", RequestDebugArgs(req)...)

	return res
}
