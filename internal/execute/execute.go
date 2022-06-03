package execute

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/execute/terraform"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// Opt is a validate module option
type Opt func(*Executor)

// Executor is a Terraform module executor
type Executor struct {
	TFConfig *terraform.Config
	Module   *pb.Terraform_Module
}

// NewTextOutput returns a new TextOutput
func NewTextOutput() *TextOutput {
	return &TextOutput{
		// Stdout is currently discarded because we don't do anything with
		// terraform's raw output.
		Stdout: io.Discard,
		Stderr: &strings.Builder{},
	}
}

// TextOutput is a terraform text output collector
type TextOutput struct {
	Stdout io.Writer
	Stderr *strings.Builder
}

// NewExecutor takes options and returns a new validated generator
func NewExecutor(opts ...Opt) *Executor {
	ex := &Executor{}

	for _, opt := range opts {
		opt(ex)
	}

	return ex
}

// WithProtoModuleAndConfig configures the executor with configuration
// from the proto module and executor configuration.
func WithProtoModuleAndConfig(mod *pb.Terraform_Module, cfg *pb.Terraform_Executor_Config) Opt {
	return func(ex *Executor) {
		ex.Module = mod
		ex.TFConfig = terraform.NewConfig(
			terraform.WithProtoConfig(cfg),
			terraform.WithDirPath(filepath.Dir(mod.GetModulePath())),
			terraform.WithConfigPath(mod.GetRcPath()),
		)
	}
}

// Validate validates the generated Terraform module by installing any required
// providers or modules and planning it.
func (e *Executor) Validate(ctx context.Context) *pb.Scenario_Command_Validate_Response {
	res := &pb.Scenario_Command_Validate_Response{
		Generate: &pb.Scenario_Command_Generate_Response{
			TerraformModule: e.Module,
		},
	}

	tf, err := e.TFConfig.Terraform()
	if err != nil {
		res.Diagnostics = diagnostics.FromErr(err)
		return res
	}

	initOut := NewTextOutput()
	tf.SetStdout(initOut.Stdout)
	tf.SetStderr(initOut.Stderr)
	err = tf.Init(ctx, e.TFConfig.InitOptions()...)
	res.Init = &pb.Terraform_Command_Init_Response{
		Stderr: initOut.Stderr.String(),
	}
	if err != nil {
		res.Init.Diagnostics = diagnostics.FromErr(err)
		return res
	}

	res.Validate = &pb.Terraform_Command_Validate_Response{}
	jsonOut, err := tf.Validate(ctx)
	if err != nil {
		res.Validate.Diagnostics = diagnostics.FromErr(err)
		return res
	}

	if err == nil && jsonOut != nil {
		res.Validate.FormatVersion = jsonOut.FormatVersion
		res.Validate.Valid = jsonOut.Valid
		res.Validate.ErrorCount = int64(jsonOut.ErrorCount)
		res.Validate.WarningCount = int64(jsonOut.WarningCount)
		res.Validate.Diagnostics = append(res.Validate.Diagnostics, diagnostics.FromTFJSON(jsonOut.Diagnostics)...)

		if e.TFConfig.FailOnWarnings && !res.Validate.Valid {
			return res
		}
	}

	planOut := NewTextOutput()
	tf.SetStdout(planOut.Stdout)
	tf.SetStderr(planOut.Stderr)
	changes, err := tf.Plan(ctx, e.TFConfig.PlanOptions()...)
	res.Plan = &pb.Terraform_Command_Plan_Response{
		ChangesPresent: changes,
		Stderr:         planOut.Stderr.String(),
	}
	if err != nil {
		res.Plan.Diagnostics = diagnostics.FromErr(err)
		return res
	}

	return res
}

// Launch execute the Terraform plan.
func (e *Executor) Launch(ctx context.Context) *pb.Scenario_Command_Launch_Response {
	res := &pb.Scenario_Command_Launch_Response{
		Generate: &pb.Scenario_Command_Generate_Response{
			TerraformModule: e.Module,
		},
	}

	tf, err := e.TFConfig.Terraform()
	if err != nil {
		res.Diagnostics = diagnostics.FromErr(err)
		return res
	}

	validateRes := e.Validate(ctx)
	res.Diagnostics = validateRes.GetDiagnostics()
	res.Generate = validateRes.GetGenerate()
	res.Init = validateRes.GetInit()
	res.Validate = validateRes.GetValidate()
	res.Plan = validateRes.GetPlan()

	if diagnostics.HasErrors(
		res.GetDiagnostics(),
		res.GetGenerate().GetDiagnostics(),
		res.GetInit().GetDiagnostics(),
		res.GetValidate().GetDiagnostics(),
		res.GetPlan().GetDiagnostics(),
	) {
		return res
	}

	if e.TFConfig.FailOnWarnings && diagnostics.HasWarnings(
		res.GetDiagnostics(),
		res.GetGenerate().GetDiagnostics(),
		res.GetInit().GetDiagnostics(),
		res.GetValidate().GetDiagnostics(),
		res.GetPlan().GetDiagnostics(),
	) {
		return res
	}

	applyOut := NewTextOutput()
	tf.SetStdout(applyOut.Stdout)
	tf.SetStderr(applyOut.Stderr)
	err = tf.Apply(ctx, e.TFConfig.ApplyOptions()...)
	res.Apply = &pb.Terraform_Command_Apply_Response{
		Stderr:      applyOut.Stderr.String(),
		Diagnostics: diagnostics.FromErr(err),
	}

	return res
}

// Destroy destroys the Terraform plan.
func (e *Executor) Destroy(ctx context.Context) *pb.Scenario_Command_Destroy_Response {
	res := &pb.Scenario_Command_Destroy_Response{
		TerraformModule: e.Module,
	}

	tf, err := e.TFConfig.Terraform()
	if err != nil {
		res.Diagnostics = diagnostics.FromErr(err)
		return res
	}

	destroyOut := NewTextOutput()
	tf.SetStdout(destroyOut.Stdout)
	tf.SetStderr(destroyOut.Stderr)
	err = tf.Destroy(ctx, e.TFConfig.DestroyOptions()...)
	res.Destroy = &pb.Terraform_Command_Destroy_Response{
		Stderr: destroyOut.Stderr.String(),
	}
	if err != nil {
		res.Destroy.Diagnostics = diagnostics.FromErr(err)
	}

	return res
}

// Run performs an entire scenario execution cycle
func (e *Executor) Run(ctx context.Context) *pb.Scenario_Command_Run_Response {
	res := &pb.Scenario_Command_Run_Response{
		Generate: &pb.Scenario_Command_Generate_Response{
			TerraformModule: e.Module,
		},
	}

	launchRes := e.Launch(ctx)
	res.Diagnostics = launchRes.GetDiagnostics()
	res.Generate = launchRes.GetGenerate()
	res.Init = launchRes.GetInit()
	res.Validate = launchRes.GetValidate()
	res.Plan = launchRes.GetPlan()
	res.Apply = launchRes.GetApply()

	if diagnostics.HasErrors(
		res.GetDiagnostics(),
		res.GetGenerate().GetDiagnostics(),
		res.GetInit().GetDiagnostics(),
		res.GetValidate().GetDiagnostics(),
		res.GetPlan().GetDiagnostics(),
		res.GetApply().GetDiagnostics(),
	) {
		return res
	}

	if e.TFConfig.FailOnWarnings && diagnostics.HasWarnings(
		res.GetDiagnostics(),
		res.GetGenerate().GetDiagnostics(),
		res.GetInit().GetDiagnostics(),
		res.GetValidate().GetDiagnostics(),
		res.GetPlan().GetDiagnostics(),
		res.GetApply().GetDiagnostics(),
	) {
		return res
	}

	destroyRes := e.Destroy(ctx)
	res.Diagnostics = destroyRes.GetDiagnostics()
	res.Destroy = destroyRes.GetDestroy()

	return res
}

// Exec executes a raw terraform sub command
func (e *Executor) Exec(ctx context.Context) *pb.Scenario_Command_Exec_Response {
	execOut := NewTextOutput()
	stdout := &strings.Builder{}
	execOut.Stdout = stdout
	cmd := e.TFConfig.NewExecSubCmd()
	cmd.ExecOpts = append(cmd.ExecOpts, func(ecmd *exec.Cmd) {
		ecmd.Stderr = execOut.Stderr
		ecmd.Stdout = execOut.Stdout
	})

	_, err := cmd.Run(ctx)
	return &pb.Scenario_Command_Exec_Response{
		TerraformModule: e.Module,
		SubCommand:      e.TFConfig.ExecSubCmd,
		Exec: &pb.Terraform_Command_Exec_Response{
			Stdout:      stdout.String(),
			Stderr:      execOut.Stderr.String(),
			Diagnostics: diagnostics.FromErr(err),
		},
	}
}

// Output returns the state output
func (e *Executor) Output(ctx context.Context) *pb.Scenario_Command_Output_Response {
	res := &pb.Scenario_Command_Output_Response{
		TerraformModule: e.Module,
		Output: &pb.Terraform_Command_Output_Response{
			Meta: []*pb.Terraform_Command_Output_Response_Meta{},
		},
	}

	tf, err := e.TFConfig.Terraform()
	if err != nil {
		res.Diagnostics = diagnostics.FromErr(err)
		return res
	}

	outText := NewTextOutput()
	tf.SetStdout(outText.Stdout)
	tf.SetStderr(outText.Stderr)

	metas, err := tf.Output(ctx, e.TFConfig.OutputOptions()...)
	if err != nil {
		res.Diagnostics = diagnostics.FromErr(err)
		return res
	}

	if e.TFConfig.OutputName != "" {
		meta, found := metas[e.TFConfig.OutputName]
		if !found {
			err := fmt.Errorf("no output with key %s", e.TFConfig.OutputName)
			res.Diagnostics = diagnostics.FromErr(err)
			return res
		}

		res.Output.Meta = append(res.Output.Meta, &pb.Terraform_Command_Output_Response_Meta{
			Name:      e.TFConfig.OutputName,
			Type:      []byte(meta.Type),
			Value:     []byte(meta.Value),
			Sensitive: meta.Sensitive,
			Stderr:    outText.Stderr.String(),
		})

		return res
	}

	for name, meta := range metas {
		res.Output.Meta = append(res.Output.Meta, &pb.Terraform_Command_Output_Response_Meta{
			Name:      name,
			Type:      []byte(meta.Type),
			Value:     []byte(meta.Value),
			Sensitive: meta.Sensitive,
			Stderr:    outText.Stderr.String(),
		})
	}

	return res
}
