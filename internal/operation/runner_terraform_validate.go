package operation

import (
	"context"
	"fmt"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// terraformValidate validates a Terraform module
func (r *Runner) terraformValidate(
	ctx context.Context,
	req *pb.Operation_Request,
	events *EventSender,
) *pb.Terraform_Command_Validate_Response {
	res := &pb.Terraform_Command_Validate_Response{
		Diagnostics: []*pb.Diagnostic{},
	}
	log := r.log.With(RequestDebugArgs(req)...)

	ref, err := NewReferenceFromRequest(req)
	if err != nil {
		res.Diagnostics = append(res.Diagnostics, diagnostics.FromErr(err)...)
		log.Error("failed to create reference from request", "error", err)
		return res
	}

	// Notify running validate
	eventVal := &pb.Operation_Event_Validate{}
	event := newEvent(ref, pb.Operation_STATUS_RUNNING)
	event.Value = eventVal
	if err = events.Publish(event); err != nil {
		res.Diagnostics = append(res.Diagnostics, diagnostics.FromErr(err)...)
		log.Error("failed to publish event", "error", err)
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
			log.Error("failed to publish event", "error", err)
		}
	}

	// Create our terraform executor
	tf, err := r.TFConfig.Terraform()
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

		if r.TFConfig.FailOnWarnings && !res.Valid {
			err = fmt.Errorf("failing on validation warnings")
			// We'll handle this error below and exit after notifyFail
		}
	}
	if err != nil {
		notifyFail(diagnostics.FromErr(err))
		return res
	}

	// Finalize our responses and event
	event.Status = diagnostics.Status(r.TFConfig.FailOnWarnings, res.GetDiagnostics()...)
	eventVal.Validate = res

	// Notify that we've finished
	if err := events.Publish(event); err != nil {
		log.Error("failed to send event", "error", err)
		res.Diagnostics = append(res.Diagnostics, diagnostics.FromErr(err)...)
	}
	log.Debug("finished validate")

	return res
}
