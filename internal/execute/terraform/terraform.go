package terraform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/hashicorp/enos/internal/execute/command"
	"github.com/hashicorp/enos/internal/ui/terminal"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	"github.com/hashicorp/terraform-exec/tfexec"
)

// Config is the Terraform CLI executor configuration
type Config struct {
	UI             *terminal.UI      // UI to use for input/output
	BinPath        string            // where terraform binary is
	ConfigPath     string            // where the terraformrc config is
	DirPath        string            // what directory to execute the command in
	Env            map[string]string // envrionment variables
	ExecSubCmd     string            // raw command to run
	OutputName     string            // output name
	FailOnWarnings bool              // fail on warning diagnostics
	Flags          *pb.Terraform_Executor_Config_Flags
}

// Proto returns the instance of config as proto terraform executor config
func (c *Config) Proto() *pb.Terraform_Executor_Config {
	return &pb.Terraform_Executor_Config{
		Flags:          c.Flags,
		BinPath:        c.BinPath,
		ConfigPath:     c.ConfigPath,
		WorkingDirPath: c.DirPath,
		Env:            c.Env,
		UserSubCommand: c.ExecSubCmd,
		OutputFilter:   c.OutputName,
		FailOnWarnings: c.FailOnWarnings,
	}
}

// FromProto unmarshals and instance of a proto terraform executor configuration
// into itself.
func (c *Config) FromProto(pcfg *pb.Terraform_Executor_Config) {
	c.Flags = pcfg.GetFlags()
	c.BinPath = pcfg.GetBinPath()
	c.ConfigPath = pcfg.GetConfigPath()
	c.DirPath = pcfg.GetWorkingDirPath()
	c.Env = pcfg.GetEnv()
	c.ExecSubCmd = pcfg.GetUserSubCommand()
	c.OutputName = pcfg.GetOutputFilter()
	c.FailOnWarnings = pcfg.GetFailOnWarnings()
}

func (c *Config) tfPath() (string, error) {
	if c.BinPath != "" {
		return filepath.Abs(c.BinPath)
	}

	return exec.LookPath("terraform")
}

func (c *Config) tfEnv() map[string]string {
	env := map[string]string{}

	for _, val := range os.Environ() {
		if i := strings.Index(val, "="); i >= 0 {
			env[val[:i]] = val[i+len("="):]
		}
	}

	for k, v := range c.Env {
		env[k] = v
	}

	if c.ConfigPath != "" {
		env["TF_CLI_CONFIG_FILE"] = c.ConfigPath
	}

	return tfexec.CleanEnv(env)
}

// NewExecSubCmd creates a new instance of a command to run a terraform
// sub-command
func (c *Config) NewExecSubCmd() *command.Command {
	execPath, err := c.tfPath()
	if err != nil {
		return nil
	}

	opts := []command.Opt{
		command.WithEnv(c.tfEnv()),
		command.WithArgs(strings.Split(c.ExecSubCmd, " ")...),
	}

	if c.DirPath != "" {
		opts = append(opts, command.WithDir(c.DirPath))
	}

	if c.UI != nil {
		opts = append(opts, command.WithUI(c.UI))
	}

	return command.NewCommand(execPath, opts...)
}

// Terraform returns a new instance of a configured *tfexec.Terraform
func (c *Config) Terraform() (*tfexec.Terraform, error) {
	var err error
	var tf *tfexec.Terraform

	execPath, err := c.tfPath()
	if err != nil {
		return nil, err
	}

	tf, err = tfexec.NewTerraform(c.DirPath, execPath)
	if err != nil {
		return tf, err
	}

	err = tf.SetEnv(c.tfEnv())
	if err != nil {
		return tf, err
	}

	if c.UI != nil {
		tf.SetStderr(c.UI.Stderr)
		tf.SetStdout(c.UI.Stdout)
	}

	return tf, nil
}

// ConfigOpt is a Terraform CLI executor configuration option
type ConfigOpt func(*Config)

type (
	// Command is a Terraform sub-command
	Command int
	// Flag is a Terraform sub-command config flag
	Flag int
)

// NewConfig takes options and returns a new instance of config
func NewConfig(opts ...ConfigOpt) *Config {
	cfg := &Config{
		Env:   map[string]string{},
		Flags: &pb.Terraform_Executor_Config_Flags{},
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// WithUI sets the UI
func WithUI(ui *terminal.UI) ConfigOpt {
	return func(cfg *Config) {
		cfg.UI = ui
	}
}

// WithBinPath sets the terraform binary path
func WithBinPath(path string) ConfigOpt {
	return func(cfg *Config) {
		cfg.BinPath = path
	}
}

// WithConfigPath sets the terraform.rc path
func WithConfigPath(path string) ConfigOpt {
	return func(cfg *Config) {
		cfg.ConfigPath = path
	}
}

// WithDirPath sets the terraform module directory path
func WithDirPath(path string) ConfigOpt {
	return func(cfg *Config) {
		cfg.DirPath = path
	}
}

// WithEnv set environment variables
func WithEnv(env map[string]string) ConfigOpt {
	return func(cfg *Config) {
		cfg.Env = env
	}
}

// WithExecSubCommand set the raw sub-command
func WithExecSubCommand(cmd string) ConfigOpt {
	return func(cfg *Config) {
		cfg.ExecSubCmd = cmd
	}
}

// WithLockTimeout sets the state lock timeout
func WithLockTimeout(timeout time.Duration) ConfigOpt {
	return func(cfg *Config) {
		cfg.Flags.LockTimeout = durationpb.New(timeout)
	}
}

// WithNoBackend disables the configured backend
func WithNoBackend() ConfigOpt {
	return func(cfg *Config) {
		cfg.Flags.NoBackend = true
	}
}

// WithNoLock disables waiting for the state lock
func WithNoLock() ConfigOpt {
	return func(cfg *Config) {
		cfg.Flags.NoLock = true
	}
}

// WithNoDownload disables module and provider downloading during init
func WithNoDownload() ConfigOpt {
	return func(cfg *Config) {
		cfg.Flags.NoDownload = true
	}
}

// WithNoRefresh disables refreshing during plan and apply
func WithNoRefresh() ConfigOpt {
	return func(cfg *Config) {
		cfg.Flags.NoRefresh = true
	}
}

// WithParallelism sets the parallelism
func WithParallelism(p int) ConfigOpt {
	return func(cfg *Config) {
		cfg.Flags.Parallelism = uint32(p)
	}
}

// WithRefreshOnly does refresh only mode
func WithRefreshOnly() ConfigOpt {
	return func(cfg *Config) {
		cfg.Flags.RefreshOnly = true
	}
}

// WithUpgrade upgrades the terraform providers and modules during init
func WithUpgrade() ConfigOpt {
	return func(cfg *Config) {
		cfg.Flags.Upgrade = true
	}
}

// WithProtoConfig sets configuration from a proto config
func WithProtoConfig(pcfg *pb.Terraform_Executor_Config) ConfigOpt {
	return func(cfg *Config) {
		cfg.FromProto(pcfg)
	}
}

func (c *Config) lockTimeoutString() string {
	if c.Flags.GetLockTimeout().AsDuration() == 0 {
		return "0s"
	}

	return fmt.Sprintf("%dms", c.Flags.GetLockTimeout().AsDuration().Milliseconds())
}

// InitOptions are the init command options
func (c *Config) InitOptions() []tfexec.InitOption {
	return []tfexec.InitOption{
		tfexec.Backend(c.Flags.GetNoBackend()),
		tfexec.Get(!c.Flags.GetNoDownload()),
		tfexec.Upgrade(!c.Flags.GetNoLock()),
	}
}

// PlanOptions are the plan command options
func (c *Config) PlanOptions() []tfexec.PlanOption {
	return []tfexec.PlanOption{
		tfexec.Refresh(!c.Flags.GetNoRefresh()),
		tfexec.LockTimeout(c.lockTimeoutString()),
		tfexec.Lock(!c.Flags.GetNoLock()),
		tfexec.Parallelism(int(c.Flags.GetParallelism())),
		tfexec.Refresh(c.Flags.GetRefreshOnly()),
	}
}

// ApplyOptions are the apply command options
func (c *Config) ApplyOptions() []tfexec.ApplyOption {
	return []tfexec.ApplyOption{
		tfexec.Backup(c.Flags.GetBackupStateFilePath()),
		tfexec.Refresh(!c.Flags.GetNoRefresh()),
		tfexec.LockTimeout(c.lockTimeoutString()),
		tfexec.Lock(!c.Flags.GetNoLock()),
		tfexec.Parallelism(int(c.Flags.GetParallelism())),
		tfexec.Refresh(c.Flags.GetRefreshOnly()),
	}
}

// DestroyOptions are the destroy command options
func (c *Config) DestroyOptions() []tfexec.DestroyOption {
	return []tfexec.DestroyOption{
		tfexec.Backup(c.Flags.GetBackupStateFilePath()),
		tfexec.Refresh(!c.Flags.GetNoRefresh()),
		tfexec.LockTimeout(c.lockTimeoutString()),
		tfexec.Lock(!c.Flags.GetNoLock()),
		tfexec.Parallelism(int(c.Flags.GetParallelism())),
		tfexec.Refresh(c.Flags.GetRefreshOnly()),
	}
}

// OutputOptions are the output commands options
func (c *Config) OutputOptions() []tfexec.OutputOption {
	return []tfexec.OutputOption{}
}
