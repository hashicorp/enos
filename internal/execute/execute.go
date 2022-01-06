package execute

import (
	"context"
	"os/exec"

	"github.com/hashicorp/enos/internal/execute/terraform"
)

// Opt is a validate module option
type Opt func(*Executor) error

// Executor is a Terraform module executor
type Executor struct {
	TFConfig *terraform.Config
}

// NewExecutor takes options and returns a new validated generator
func NewExecutor(opts ...Opt) (*Executor, error) {
	ex := &Executor{}

	for _, opt := range opts {
		err := opt(ex)
		if err != nil {
			return ex, err
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
func (e *Executor) Validate(ctx context.Context) error {
	_, err := e.RunTerraformCommand(ctx, terraform.Init)
	if err != nil {
		return err
	}

	_, err = e.RunTerraformCommand(ctx, terraform.Validate)
	if err != nil {
		return err
	}

	_, err = e.RunTerraformCommand(ctx, terraform.Plan)
	if err != nil {
		return err
	}

	return nil
}

// Launch execute the Terraform plan.
func (e *Executor) Launch(ctx context.Context) error {
	_, err := e.RunTerraformCommand(ctx, terraform.Apply)
	return err
}

// Destroy execute the Terraform plan.
func (e *Executor) Destroy(ctx context.Context) error {
	_, err := e.RunTerraformCommand(ctx, terraform.Destroy)
	return err
}

// Run performs an entire test cycle
func (e *Executor) Run(ctx context.Context) error {
	err := e.Validate(ctx)
	if err != nil {
		return err
	}

	err = e.Launch(ctx)
	if err != nil {
		return err
	}

	return e.Destroy(ctx)
}

// Exec executes a raw terraform sub command
func (e *Executor) Exec(ctx context.Context) (*exec.Cmd, error) {
	return e.RunTerraformCommand(ctx, terraform.Exec)
}

// RunTerraformCommand executes a terraform command with the confiuration
func (e *Executor) RunTerraformCommand(ctx context.Context, subCmd terraform.Command) (*exec.Cmd, error) {
	return subCmd.Run(ctx, e.TFConfig)
}
