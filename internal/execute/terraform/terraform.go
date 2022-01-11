package terraform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/enos/internal/execute/command"
	"github.com/hashicorp/enos/internal/ui"
	"github.com/hashicorp/terraform-exec/tfexec"
)

// Config is the Terraform CLI executor configuration
type Config struct {
	UI         *ui.UI            // UI to use for input/output
	BinPath    string            // where terraform binary is
	ConfigPath string            // where the terraformrc config is
	DirPath    string            // what directory to execute the command in
	Env        map[string]string // envrionment variables
	ExecSubCmd string            // raw command to run
	Flags      *Flags
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

// RunExecSubCmd executes the Terraform sub-command
func (c *Config) RunExecSubCmd(ctx context.Context) (*exec.Cmd, error) {
	execPath, err := c.tfPath()
	if err != nil {
		return nil, err
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

	return command.NewCommand(execPath, opts...).Run(ctx)
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

// Flags are a subset of the Terraform flags that we allow to be settable.
type Flags struct {
	BackupStateFilePath string        // -backup=path
	LockTimeout         time.Duration // -lock-timeout=10s
	NoBackend           bool          // -backend=false
	NoLock              bool          // -lock=false
	NoDownload          bool          // -get=false
	NoRefresh           bool          // -refresh=false
	OutPath             string        // -out=path
	Parallelism         int           // -parallelism=n
	RefreshOnly         bool          // -refresh-only
	Upgrade             bool          // -upgrade
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
		Flags: &Flags{},
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// WithUI sets the UI
func WithUI(ui *ui.UI) ConfigOpt {
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
		cfg.Flags.LockTimeout = timeout
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

// WithOutPath sets the outfile path
func WithOutPath(out string) ConfigOpt {
	return func(cfg *Config) {
		cfg.Flags.OutPath = out
	}
}

// WithParallelism sets the parallelism
func WithParallelism(p int) ConfigOpt {
	return func(cfg *Config) {
		cfg.Flags.Parallelism = p
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

func (f *Flags) lockTimeoutString() string {
	if f.LockTimeout == 0 {
		return "0s"
	}

	return fmt.Sprintf("%dms", f.LockTimeout.Milliseconds())
}

// InitOptions are the init command options
func (f *Flags) InitOptions() []tfexec.InitOption {
	return []tfexec.InitOption{
		tfexec.Backend(f.NoBackend),
		tfexec.Get(!f.NoDownload),
		tfexec.Upgrade(!f.NoLock),
	}
}

// PlanOptions are the plan command options
func (f *Flags) PlanOptions() []tfexec.PlanOption {
	return []tfexec.PlanOption{
		tfexec.Refresh(!f.NoRefresh),
		tfexec.LockTimeout(f.lockTimeoutString()),
		tfexec.Lock(!f.NoLock),
		tfexec.Out(f.OutPath),
		tfexec.Parallelism(f.Parallelism),
		tfexec.Refresh(f.RefreshOnly),
	}
}

// ApplyOptions are the apply command options
func (f *Flags) ApplyOptions() []tfexec.ApplyOption {
	return []tfexec.ApplyOption{
		tfexec.Backup(f.BackupStateFilePath),
		tfexec.Refresh(!f.NoRefresh),
		tfexec.LockTimeout(f.lockTimeoutString()),
		tfexec.Lock(!f.NoLock),
		tfexec.Parallelism(f.Parallelism),
		tfexec.Refresh(f.RefreshOnly),
	}
}

// DestroyOptions are the destroy command options
func (f *Flags) DestroyOptions() []tfexec.DestroyOption {
	return []tfexec.DestroyOption{
		tfexec.Backup(f.BackupStateFilePath),
		tfexec.Refresh(!f.NoRefresh),
		tfexec.LockTimeout(f.lockTimeoutString()),
		tfexec.Lock(!f.NoLock),
		tfexec.Parallelism(f.Parallelism),
		tfexec.Refresh(f.RefreshOnly),
	}
}
