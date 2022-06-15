package operation

import (
	"context"
	"fmt"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// terraformValidate validates a Terraform module
func (e *Runner) terraformValidate(
	ctx context.Context,
	req *pb.Operation_Request,
	events *EventSender,
) *pb.Terraform_Command_Validate_Response {
	res := &pb.Terraform_Command_Validate_Response{
		Diagnostics: []*pb.Diagnostic{},
	}

	ref, err := NewReferenceFromRequest(req)
	if err != nil {
		res.Diagnostics = append(res.Diagnostics, diagnostics.FromErr(err)...)
		e.log.Error("failed to create reference from request", "error", err)
		return res
	}

	// Notify running validate
	eventVal := &pb.Operation_Event_Validate{}
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
		eventVal.Validate = res

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

	// terraform validate
	jsonOut, err := tf.Validate(ctx)
	if err == nil && jsonOut != nil {
		res.FormatVersion = jsonOut.FormatVersion
		res.Valid = jsonOut.Valid
		res.ErrorCount = int64(jsonOut.ErrorCount)
		res.WarningCount = int64(jsonOut.WarningCount)
		res.Diagnostics = append(
			res.Diagnostics,
			diagnostics.FromTFJSON(jsonOut.Diagnostics)...,
		)

		if e.TFConfig.FailOnWarnings && !res.Valid {
			err = fmt.Errorf("failing on validation warnings")
			// exit after we update our event handler with validation info
		}
	}
	if err != nil {
		notifyFail(diagnostics.FromErr(err))
		return res
	}

	// Finalize our responses and event
	event.Status = diagnostics.Status(e.TFConfig.FailOnWarnings, res.GetDiagnostics()...)
	eventVal.Validate = res

	// Notify that we've finished
	if err := events.Publish(event); err != nil {
		e.log.Error("failed to send event", "error", err)
		res.Diagnostics = append(res.Diagnostics, diagnostics.FromErr(err)...)
	}
	e.log.Debug("finished validate", RequestDebugArgs(req)...)

	return res
}
