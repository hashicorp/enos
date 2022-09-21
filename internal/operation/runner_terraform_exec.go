package operation

import (
	"context"
	"os/exec"
	"strings"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// terraformExec executes any Terraform sub-command within the context of a generated
// module.
func (e *Runner) terraformExec(
	ctx context.Context,
	req *pb.Operation_Request,
	events *EventSender,
) *pb.Terraform_Command_Exec_Response {
	res := &pb.Terraform_Command_Exec_Response{
		Diagnostics: []*pb.Diagnostic{},
	}
	log := e.log.With(RequestDebugArgs(req)...)

	ref, err := NewReferenceFromRequest(req)
	if err != nil {
		res.Diagnostics = append(res.Diagnostics, diagnostics.FromErr(err)...)
		log.Error("failed to create reference from request", "error", err)
		return res
	}

	// Notify running exec
	eventVal := &pb.Operation_Event_Exec{}
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
		eventVal.Exec = res

		if err := events.Publish(event); err != nil {
			res.Diagnostics = append(res.Diagnostics, diagnostics.FromErr(err)...)
			log.Error("failed to publish event", "error", err)
		}
	}

	// terraform exec
	execOut := NewTextOutput()
	stdout := &strings.Builder{}
	execOut.Stdout = stdout
	cmd := e.TFConfig.NewExecSubCmd()
	cmd.ExecOpts = append(cmd.ExecOpts, func(ecmd *exec.Cmd) {
		ecmd.Stderr = execOut.Stderr
		ecmd.Stdout = execOut.Stdout
	})

	_, err = cmd.Run(ctx)
	res.SubCommand = e.TFConfig.ExecSubCmd
	res.Stdout = stdout.String()
	res.Stderr = execOut.Stderr.String()
	res.Diagnostics = diagnostics.FromErr(err)

	if err != nil {
		notifyFail(diagnostics.FromErr(err))
		return res
	}

	// Finalize our responses and event
	event.Status = diagnostics.Status(e.TFConfig.FailOnWarnings, res.GetDiagnostics()...)
	event.Diagnostics = res.Diagnostics
	eventVal.Exec = res

	// Notify that we've finished
	if err := events.Publish(event); err != nil {
		log.Error("failed to send event", "error", err)
		res.Diagnostics = append(res.Diagnostics, diagnostics.FromErr(err)...)
	}
	log.Debug("finished exec")

	return res
}
