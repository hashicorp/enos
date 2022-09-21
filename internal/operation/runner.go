package operation

import (
	"io"
	"strings"

	"github.com/hashicorp/enos/internal/operation/terraform"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	"github.com/hashicorp/go-hclog"
)

// RunnerOpt is a validate module option
type RunnerOpt func(*Runner)

// Runner is a Terraform command runner
type Runner struct {
	TFConfig *terraform.Config
	Module   *pb.Terraform_Module
	log      hclog.Logger
}

// NewTextOutput returns a new TextOutput
func NewTextOutput() *TextOutput {
	return &TextOutput{ // Stdout is currently discarded because we don't do anything with
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

// NewRunner takes options and returns a new validated generator
func NewRunner(opts ...RunnerOpt) *Runner {
	ex := &Runner{
		log: hclog.NewNullLogger(),
	}

	for _, opt := range opts {
		opt(ex)
	}

	return ex
}

// WithRunnerTerraformConfig configures the Runner with RunnerConfig passed over
// the wire.
func WithRunnerTerraformConfig(cfg *pb.Terraform_Runner_Config) RunnerOpt {
	return func(ex *Runner) {
		ex.TFConfig = terraform.NewConfig(terraform.WithProtoConfig(cfg))
	}
}

// WithLogger sets the logger
func WithLogger(log hclog.Logger) RunnerOpt {
	return func(ex *Runner) {
		ex.log = log
	}
}
