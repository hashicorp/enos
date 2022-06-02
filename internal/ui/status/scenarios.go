package status

import (
	"google.golang.org/grpc/codes"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// GenerateScenarios returns the status response for a scenario generate
func GenerateScenarios(res *pb.GenerateScenariosResponse) error {
	var err error

	for _, out := range res.GetResponses() {
		if !HasErrorDiags(out) {
			continue
		}
		scenario := flightplan.NewScenario()
		scenario.FromRef(out.GetTerraformModule().GetScenarioRef())
		err = Error(scenario.String(), codes.Internal, err)
	}

	if HasErrorDiags(res) {
		err = Error("unable to generate scenarios", codes.Internal, err)
	}

	return err
}

// ListScenarios returns the status response for a scenario list
func ListScenarios(res *pb.ListScenariosResponse) error {
	if HasErrorDiags(res) {
		return Error("unable to list scenarios", codes.Internal)
	}

	return nil
}

// ValidateScenarios returns the status response for a scenario launch
func ValidateScenarios(res *pb.ValidateScenariosResponse) error {
	var err error

	for _, out := range res.GetResponses() {
		if !diagnostics.HasErrors(diagnostics.Concat(
			out.GetGenerate().GetDiagnostics(),
			out.GetInit().GetDiagnostics(),
			out.GetValidate().GetDiagnostics(),
			out.GetPlan().GetDiagnostics(),
		)) {
			continue
		}

		scenario := flightplan.NewScenario()
		scenario.FromRef(out.GetGenerate().GetTerraformModule().GetScenarioRef())
		err = Error(scenario.String(), codes.Internal, err)
	}

	if HasErrorDiags(res) {
		err = Error("unable to validate scenarios", codes.Internal, err)
	}

	return err
}

// LaunchScenarios returns the status response for a scenario launch
func LaunchScenarios(res *pb.LaunchScenariosResponse) error {
	var err error

	for _, out := range res.GetResponses() {
		if !diagnostics.HasErrors(diagnostics.Concat(
			out.GetGenerate().GetDiagnostics(),
			out.GetInit().GetDiagnostics(),
			out.GetValidate().GetDiagnostics(),
			out.GetPlan().GetDiagnostics(),
			out.GetApply().GetDiagnostics(),
		)) {
			continue
		}

		scenario := flightplan.NewScenario()
		scenario.FromRef(out.GetGenerate().GetTerraformModule().GetScenarioRef())
		err = Error(scenario.String(), codes.Internal, err)
	}

	if HasErrorDiags(res) {
		err = Error("unable to launch scenarios", codes.Internal, err)
	}

	return err
}

// RunScenarios returns the status response for a scenario run
func RunScenarios(res *pb.RunScenariosResponse) error {
	var err error

	for _, out := range res.GetResponses() {
		if !diagnostics.HasErrors(diagnostics.Concat(
			out.GetGenerate().GetDiagnostics(),
			out.GetInit().GetDiagnostics(),
			out.GetValidate().GetDiagnostics(),
			out.GetPlan().GetDiagnostics(),
			out.GetApply().GetDiagnostics(),
			out.GetDestroy().GetDiagnostics(),
		)) {
			continue
		}

		scenario := flightplan.NewScenario()
		scenario.FromRef(out.GetGenerate().GetTerraformModule().GetScenarioRef())
		err = Error(scenario.String(), codes.Internal, err)
	}

	if HasErrorDiags(res) {
		err = Error("unable to run scenarios", codes.Internal, err)
	}

	return err
}

// DestroyScenarios returns the status response for a scenario destroy
func DestroyScenarios(res *pb.DestroyScenariosResponse) error {
	var err error

	for _, out := range res.GetResponses() {
		if !HasErrorDiags(out) {
			continue
		}
		scenario := flightplan.NewScenario()
		scenario.FromRef(out.GetTerraformModule().GetScenarioRef())
		err = Error(scenario.String(), codes.Internal, err)
	}

	if HasErrorDiags(res) {
		err = Error("failed to destroy scenarios", codes.Internal, err)
	}

	return err
}

// ExecScenarios returns the status response for a scenario exec
func ExecScenarios(res *pb.ExecScenariosResponse) error {
	var err error

	for _, out := range res.GetResponses() {
		if !HasErrorDiags(out) {
			continue
		}
		scenario := flightplan.NewScenario()
		scenario.FromRef(out.GetTerraformModule().GetScenarioRef())
		err = Error(scenario.String(), codes.Internal, err)
	}

	if HasErrorDiags(res) {
		err = Error("unable to execute scenarios", codes.Internal, err)
	}

	return err
}

// OutputScenarios returns the status response for a scenario output
func OutputScenarios(res *pb.OutputScenariosResponse) error {
	var err error

	for _, out := range res.GetResponses() {
		if !HasErrorDiags(out) {
			continue
		}
		scenario := flightplan.NewScenario()
		scenario.FromRef(out.GetTerraformModule().GetScenarioRef())
		err = Error(scenario.String(), codes.Internal, err)
	}

	if HasErrorDiags(res) {
		err = Error("unable to output scenarios", codes.Internal, err)
	}

	return err
}
