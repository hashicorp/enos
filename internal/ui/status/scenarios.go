package status

import (
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// GenerateScenarios returns the status response for a scenario generate
func GenerateScenarios(failOnWarn bool, res *pb.GenerateScenariosResponse) error {
	var err error

	for _, out := range res.GetResponses() {
		if !HasFailed(failOnWarn, out) {
			continue
		}
		scenario := flightplan.NewScenario()
		scenario.FromRef(out.GetTerraformModule().GetScenarioRef())
		err = Error(scenario.String(), err)
	}

	if HasFailed(failOnWarn, res, res.GetDecode()) {
		err = Error("failed to generate scenarios", err)
	}

	return err
}

// ListScenarios returns the status response for a scenario list
func ListScenarios(failOnWarn bool, res *pb.ListScenariosResponse) error {
	if HasFailed(failOnWarn, res, res.GetDecode()) {
		return Error("failed to list scenarios")
	}

	return nil
}

// ValidateScenarios returns the status response for a scenario launch
func ValidateScenarios(failOnWarn bool, res *pb.ValidateScenariosResponse) error {
	var err error

	for _, out := range res.GetResponses() {
		if !HasFailed(failOnWarn,
			out.GetGenerate(),
			out.GetInit(),
			out.GetValidate(),
			out.GetPlan(),
		) {
			continue
		}

		scenario := flightplan.NewScenario()
		scenario.FromRef(out.GetGenerate().GetTerraformModule().GetScenarioRef())
		err = Error(scenario.String(), err)
	}

	if HasFailed(failOnWarn, res, res.GetDecode()) {
		err = Error("failed to validate scenarios", err)
	}

	return err
}

// LaunchScenarios returns the status response for a scenario launch
func LaunchScenarios(failOnWarn bool, res *pb.LaunchScenariosResponse) error {
	var err error

	for _, out := range res.GetResponses() {
		if !HasFailed(failOnWarn,
			out.GetGenerate(),
			out.GetInit(),
			out.GetValidate(),
			out.GetPlan(),
			out.GetApply(),
		) {
			continue
		}

		scenario := flightplan.NewScenario()
		scenario.FromRef(out.GetGenerate().GetTerraformModule().GetScenarioRef())
		err = Error(scenario.String(), err)
	}

	if HasFailed(failOnWarn, res, res.GetDecode()) {
		err = Error("failed to launch scenarios", err)
	}

	return err
}

// RunScenarios returns the status response for a scenario run
func RunScenarios(failOnWarn bool, res *pb.RunScenariosResponse) error {
	var err error

	for _, out := range res.GetResponses() {
		if !HasFailed(failOnWarn,
			out.GetGenerate(),
			out.GetInit(),
			out.GetValidate(),
			out.GetPlan(),
			out.GetApply(),
			out.GetDestroy(),
		) {
			continue
		}

		scenario := flightplan.NewScenario()
		scenario.FromRef(out.GetGenerate().GetTerraformModule().GetScenarioRef())
		err = Error(scenario.String(), err)
	}

	if HasFailed(failOnWarn, res, res.GetDecode()) {
		err = Error("failed to run scenarios", err)
	}

	return err
}

// DestroyScenarios returns the status response for a scenario destroy
func DestroyScenarios(failOnWarn bool, res *pb.DestroyScenariosResponse) error {
	var err error

	for _, out := range res.GetResponses() {
		if !HasFailed(failOnWarn, out) {
			continue
		}
		scenario := flightplan.NewScenario()
		scenario.FromRef(out.GetTerraformModule().GetScenarioRef())
		err = Error(scenario.String(), err)
	}

	if HasFailed(failOnWarn, res, res.GetDecode()) {
		err = Error("failed to destroy scenarios", err)
	}

	return err
}

// ExecScenarios returns the status response for a scenario exec
func ExecScenarios(failOnWarn bool, res *pb.ExecScenariosResponse) error {
	var err error

	for _, out := range res.GetResponses() {
		if !HasFailed(true, res) {
			continue
		}
		scenario := flightplan.NewScenario()
		scenario.FromRef(out.GetTerraformModule().GetScenarioRef())
		err = Error(scenario.String(), err)
	}

	if HasFailed(failOnWarn, res, res.GetDecode()) {
		err = Error("failed to execute command in context of scenarios", err)
	}

	return err
}

// OutputScenarios returns the status response for a scenario output
func OutputScenarios(failOnWarn bool, res *pb.OutputScenariosResponse) error {
	var err error

	for _, out := range res.GetResponses() {
		if !HasFailed(failOnWarn, out) {
			continue
		}
		scenario := flightplan.NewScenario()
		scenario.FromRef(out.GetTerraformModule().GetScenarioRef())
		err = Error(scenario.String(), err)
	}

	if HasFailed(failOnWarn, res, res.GetDecode()) {
		err = Error("failed to output scenario outputs", err)
	}

	return err
}
