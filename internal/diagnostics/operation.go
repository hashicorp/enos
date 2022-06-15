package diagnostics

import (
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// OpResFailed checks an operation response for failure diagnostics
func OpResFailed(failOnWarn bool, res *pb.Operation_Response) bool {
	return OpResErrors(res) || (failOnWarn && OpResWarnings(res))
}

// OpResErrors checks an operation response failure diagnostics
func OpResErrors(res *pb.Operation_Response) bool {
	return HasErrors(
		res.GetGenerate().GetDiagnostics(),
		res.GetCheck().GetInit().GetDiagnostics(),
		res.GetCheck().GetValidate().GetDiagnostics(),
		res.GetCheck().GetPlan().GetDiagnostics(),
		res.GetLaunch().GetDiagnostics(),
		res.GetLaunch().GetApply().GetDiagnostics(),
		res.GetDestroy().GetDiagnostics(),
		res.GetDestroy().GetDestroy().GetDiagnostics(),
		res.GetExec().GetDiagnostics(),
		res.GetOutput().GetDiagnostics(),
		res.GetOutput().GetOutput().GetDiagnostics(),
	)
}

// OpResWarnings checks an operation response warning diagnostics
func OpResWarnings(res *pb.Operation_Response) bool {
	return HasWarnings(
		res.GetGenerate().GetDiagnostics(),
		res.GetCheck().GetInit().GetDiagnostics(),
		res.GetCheck().GetValidate().GetDiagnostics(),
		res.GetCheck().GetPlan().GetDiagnostics(),
		res.GetLaunch().GetDiagnostics(),
		res.GetLaunch().GetApply().GetDiagnostics(),
		res.GetDestroy().GetDiagnostics(),
		res.GetDestroy().GetDestroy().GetDiagnostics(),
		res.GetExec().GetDiagnostics(),
		res.GetOutput().GetDiagnostics(),
		res.GetOutput().GetOutput().GetDiagnostics(),
	)
}

// OpEventFailed returns whether or not the event has failed
func OpEventFailed(failOnWarn bool, e *pb.Operation_Event) bool {
	return OpEventErrors(e) || (failOnWarn && OpEventWarnings(e))
}

// OpEventErrors returns whether the event has errors
func OpEventErrors(e *pb.Operation_Event) bool {
	return HasErrors(
		e.GetDiagnostics(),
		e.GetDecode().GetDiagnostics(),
		e.GetGenerate().GetDiagnostics(),
		e.GetInit().GetDiagnostics(),
		e.GetValidate().GetDiagnostics(),
		e.GetApply().GetDiagnostics(),
		e.GetDestroy().GetDiagnostics(),
		e.GetExec().GetDiagnostics(),
		e.GetOutput().GetDiagnostics(),
	)
}

// OpEventWarnings returns whether the event has warnings
func OpEventWarnings(e *pb.Operation_Event) bool {
	return HasWarnings(
		e.GetDiagnostics(),
		e.GetDecode().GetDiagnostics(),
		e.GetGenerate().GetDiagnostics(),
		e.GetInit().GetDiagnostics(),
		e.GetValidate().GetDiagnostics(),
		e.GetApply().GetDiagnostics(),
		e.GetDestroy().GetDiagnostics(),
		e.GetExec().GetDiagnostics(),
		e.GetOutput().GetDiagnostics(),
	)
}

func updateFinalStatus(failOnWarn bool, res *pb.Operation_Response) {
	res.Status = pb.Operation_STATUS_COMPLETED

	if OpResFailed(failOnWarn, res) {
		res.Status = pb.Operation_STATUS_FAILED
		return
	}

	if OpResWarnings(res) {
		res.Status = pb.Operation_STATUS_COMPLETED_WARNING
		return
	}
}

func updateRunningStatus(failOnWarn bool, res *pb.Operation_Response) {
	if OpResFailed(failOnWarn, res) {
		res.Status = pb.Operation_STATUS_FAILED
		return
	}

	if OpResWarnings(res) {
		res.Status = pb.Operation_STATUS_RUNNING_WARNING
		return
	}
}

func OperationStatus(failOnWarn bool, res *pb.Operation_Response) pb.Operation_Status {
	status := pb.Operation_STATUS_COMPLETED
	if OpResFailed(failOnWarn, res) {
		status = pb.Operation_STATUS_FAILED
	} else if OpResWarnings(res) {
		status = pb.Operation_STATUS_COMPLETED_WARNING
	}

	return status
}

func Status(failOnWarn bool, diags ...*pb.Diagnostic) pb.Operation_Status {
	status := pb.Operation_STATUS_COMPLETED
	if HasFailed(failOnWarn, diags) {
		status = pb.Operation_STATUS_FAILED
	} else if HasWarnings(diags) {
		status = pb.Operation_STATUS_COMPLETED_WARNING
	}

	return status
}

// UpdateResponseStatus takes a reference to an operation response and updates
// the Status field. It considers whether or not a value has been set, whether it
// has completed all sub-operations, and whether or not it has failing diagnostics
// or should fail on warnings.
func UpdateResponseStatus(failOnWarn bool, res *pb.Operation_Response) {
	v := res.GetValue()

	if v == nil {
		res.Status = pb.Operation_STATUS_UNKNOWN
		if OpResFailed(failOnWarn, res) {
			res.Status = pb.Operation_STATUS_FAILED
		} else if OpResWarnings(res) {
			res.Status = pb.Operation_STATUS_RUNNING_WARNING
		}

		return
	}

	// Check has multiple sub-operations. The last sub-operation is plan so we'll
	// check for it existing to determine if we should update the a final or updateRunningStatus
	// status.
	if t, ok := v.(*pb.Operation_Response_Check_); ok && t.Check.GetPlan() == nil {
		updateRunningStatus(failOnWarn, res)
	} else {
		updateFinalStatus(failOnWarn, res)
	}
}
