package execute

import (
	"context"
	"os/exec"

	tfjson "github.com/hashicorp/terraform-json"

	"github.com/hashicorp/enos/internal/execute/terraform"
)

// Opt is a validate module option
type Opt func(*Executor) error

// Executor is a Terraform module executor
type Executor struct {
	TFConfig *terraform.Config
}

// ValidateResponse is the response output from the validate command
type ValidateResponse struct {
	*tfjson.ValidateOutput
}

// RunResponse is the response output from the run command
type RunResponse struct {
	*ValidateResponse
}

// NewExecutor takes options and returns a new validated generator
func NewExecutor(opts ...Opt) (*Executor, error) {
	ex := &Executor{}

	for _, opt := range opts {
		err := opt(ex)
		if err != nil {
			return nil, err
		}
	}

	return ex, nil
}

// WithTerraformConfig sets the terraform configuration
func WithTerraformConfig(cfg *terraform.Config) Opt {
	return func(ex *Executor) error {
		ex.TFConfig = cfg
		return nil
	}
}

// Validate validates the generated Terraform module by installing any required
// providers or modules and planning it.
func (e *Executor) Validate(ctx context.Context) (*ValidateResponse, error) {
	var err error
	res := &ValidateResponse{}

	tf, err := e.TFConfig.Terraform()
	if err != nil {
		return res, err
	}

	err = tf.Init(ctx, e.TFConfig.Flags.InitOptions()...)
	if err != nil {
		return res, err
	}

	res.ValidateOutput, err = tf.Validate(ctx)
	if err != nil {
		return res, err
	}

	_, err = tf.Plan(ctx, e.TFConfig.Flags.PlanOptions()...)
	return res, err
}

// Launch execute the Terraform plan.
func (e *Executor) Launch(ctx context.Context) error {
	tf, err := e.TFConfig.Terraform()
	if err != nil {
		return err
	}

	return tf.Apply(ctx, e.TFConfig.Flags.ApplyOptions()...)
}

// Destroy destroys the Terraform plan.
func (e *Executor) Destroy(ctx context.Context) error {
	tf, err := e.TFConfig.Terraform()
	if err != nil {
		return err
	}

	return tf.Destroy(ctx, e.TFConfig.Flags.DestroyOptions()...)
}

// Run performs an entire test cycle
func (e *Executor) Run(ctx context.Context) (*RunResponse, error) {
	var err error
	res := &RunResponse{}

	res.ValidateResponse, err = e.Validate(ctx)
	if err != nil {
		return res, err
	}

	err = e.Launch(ctx)
	if err != nil {
		return res, err
	}

	return res, e.Destroy(ctx)
}

// Exec executes a raw terraform sub command
func (e *Executor) Exec(ctx context.Context) (*exec.Cmd, error) {
	return e.TFConfig.RunExecSubCmd(ctx)
}
