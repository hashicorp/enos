package terraform

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/hashicorp/enos/internal/execute/command"
	"github.com/hashicorp/enos/internal/ui"
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

// Flags are all supported Terraform flags.
// NOTE: this subset includes only flags that we're currently supporting, it
// is not an exhaustive list of every flag Terraform supports.
type Flags struct {
	AutoApprove         bool          // -auto-approve
	BackupStateFilePath string        // -backup=path
	CompactWarnings     bool          // -compact-warnings
	LockTimeout         time.Duration // -lock-timeout=10s
	NoBackend           bool          // -backend=false
	NoInput             bool          // -input=false
	NoColor             bool          // -no-color
	NoLock              bool          // -lock=false
	NoDownload          bool          // -get=false
	NoRefresh           bool          // -refresh=false
	OutPath             string        // -out=path
	Parallelism         int           // -parallelism=n
	RefreshOnly         bool          // -refresh-only
	StateFilePath       string        // -state=statefile
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

// WithEnv set envrionment variables
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

// WithAutoApprove enables auto-approve
func WithAutoApprove() ConfigOpt {
	return func(cfg *Config) {
		cfg.Flags.AutoApprove = true
	}
}

// WithBackupStateFilePath sets the backup state file path
func WithBackupStateFilePath(path string) ConfigOpt {
	return func(cfg *Config) {
		cfg.Flags.BackupStateFilePath = path
	}
}

// WithCompactWarnings sets compact warnings
func WithCompactWarnings() ConfigOpt {
	return func(cfg *Config) {
		cfg.Flags.CompactWarnings = true
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

// WithNoInput disables user input for missing variables
func WithNoInput() ConfigOpt {
	return func(cfg *Config) {
		cfg.Flags.NoInput = true
	}
}

// WithNoColor disables color output
func WithNoColor() ConfigOpt {
	return func(cfg *Config) {
		cfg.Flags.NoColor = true
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

// WithStateFilePath sets the state file path
func WithStateFilePath(path string) ConfigOpt {
	return func(cfg *Config) {
		cfg.Flags.StateFilePath = path
	}
}

// WithUpgrade upgrades the terraform providers and modules during init
func WithUpgrade() ConfigOpt {
	return func(cfg *Config) {
		cfg.Flags.Upgrade = true
	}
}

// Commands are some of the Terraform commands we can execute
const (
	UnknownCommand Command = iota
	Init
	Validate
	Plan
	Apply
	Destroy
	Console
	Fmt
	ForceUnlock
	Get
	Graph
	Import
	Login
	Logout
	Output
	Providers
	Refresh
	Show
	State
	Taint
	Test
	Untaint
	Version
	Workspace
	Exec
)

// Flags are the sub-command flags we know about
const (
	UnknownFlag Flag = iota
	AutoApprove
	BackupStateFilePath
	CompactWarnings
	LockTimeout
	NoBackend
	NoColor
	NoInput
	NoLock
	NoDownload
	NoRefresh
	OutPath
	Parallelism
	RefreshOnly
	StateFilePath
	Upgrade
)

// String returns the sub-command name
func (c Command) String() string {
	switch c {
	case UnknownCommand:
		return "unknown"
	case Init:
		return "init"
	case Validate:
		return "validate"
	case Plan:
		return "plan"
	case Apply:
		return "apply"
	case Destroy:
		return "destroy"
	case Console:
		return "console"
	case Fmt:
		return "fmt"
	case ForceUnlock:
		return "force-unlock"
	case Get:
		return "get"
	case Graph:
		return "graph"
	case Import:
		return "import"
	case Login:
		return "login"
	case Logout:
		return "logout"
	case Output:
		return "output"
	case Providers:
		return "providers"
	case Refresh:
		return "refresh"
	case Show:
		return "show"
	case State:
		return "state"
	case Taint:
		return "taint"
	case Test:
		return "test"
	case Untaint:
		return "untaint"
	case Version:
		return "version"
	case Workspace:
		return "workspace"
	case Exec:
		return "raw"
	default:
		return "unknown"
	}
}

// SupportedFlags returns a set of supported flags for the command.
// NOTE: we've only added partial flag support to only a few commands.
func (c Command) SupportedFlags() []Flag {
	switch c {
	case Init:
		return []Flag{
			LockTimeout, NoBackend, NoColor, NoDownload, NoInput, NoLock, Upgrade,
		}
	case Validate:
		return []Flag{NoColor}
	case Plan:
		return []Flag{
			CompactWarnings, NoRefresh, NoColor, NoLock, NoInput, LockTimeout,
			OutPath, Parallelism, RefreshOnly, StateFilePath,
		}
	case Apply:
		return []Flag{
			AutoApprove, BackupStateFilePath, CompactWarnings, NoLock, LockTimeout,
			NoInput, Parallelism, StateFilePath,
		}
	case Destroy:
		return []Flag{
			AutoApprove, BackupStateFilePath, CompactWarnings, NoLock, LockTimeout,
			NoInput, Parallelism, StateFilePath,
		}
	default:
		return []Flag{}
	}
}

// Command takes a Terraform executor configuration and expands into an
// instance of *command.Command.
func (c Command) Command(cfg *Config) (*command.Command, error) {
	var cmd *command.Command

	subCommandName := c.String()
	if subCommandName == "unknown" || subCommandName == "" {
		return cmd, fmt.Errorf("no sub-command provided")
	}

	tfPath := "terraform" // search the path by default
	if cfg.BinPath != "" {
		tfPath = cfg.BinPath
	}

	env := cfg.Env
	if cfg.ConfigPath != "" {
		env["TF_CLI_CONFIG_FILE"] = cfg.ConfigPath
	}

	var args []string
	if c == Exec {
		args = strings.Split(cfg.ExecSubCmd, " ")
	} else {
		args = append([]string{subCommandName}, cfg.FlagStrings(c.SupportedFlags())...)
	}

	opts := []command.Opt{
		// pass through the env so we can find terraform if the path isn't set
		command.WithEnvPassthrough(),
		command.WithEnv(env),
		command.WithArgs(args...),
	}
	if cfg.DirPath != "" {
		opts = append(opts, command.WithDir(cfg.DirPath))
	}
	if cfg.UI != nil {
		opts = append(opts, command.WithUI(cfg.UI))
	}

	return command.NewCommand(tfPath, opts...), nil
}

// RunWith takes a context and Terraform executor configuration options, expand
// them into an instance of *command.Command and runs it.
func (c Command) RunWith(ctx context.Context, opts ...ConfigOpt) (*exec.Cmd, error) {
	var res *exec.Cmd
	select {
	case <-ctx.Done():
		return res, ctx.Err()
	default:
	}

	cfg := &Config{
		BinPath: "terraform",
		Env:     map[string]string{},
	}
	for _, o := range opts {
		o(cfg)
	}

	return c.Run(ctx, cfg)
}

// Run takes a context and Terraform executor configuration and runs the command.
func (c Command) Run(ctx context.Context, cfg *Config) (*exec.Cmd, error) {
	var res *exec.Cmd

	select {
	case <-ctx.Done():
		return res, ctx.Err()
	default:
	}

	cmd, err := c.Command(cfg)
	if err != nil {
		return res, err
	}

	return cmd.Run(ctx)
}

// FlagStrings returns the string representation of the flags for a given set
func (c *Config) FlagStrings(flags []Flag) []string {
	out := []string{}

	for _, flag := range flags {
		switch flag {
		case AutoApprove:
			if c.Flags.AutoApprove {
				out = append(out, "-auto-approve")
			}
		case BackupStateFilePath:
			if c.Flags.BackupStateFilePath != "" {
				out = append(out, fmt.Sprintf("-backup=%s", c.Flags.BackupStateFilePath))
			}
		case CompactWarnings:
			if c.Flags.NoRefresh {
				out = append(out, "-compact-warnings")
			}
		case LockTimeout:
			if c.Flags.LockTimeout > 0 {
				out = append(out, fmt.Sprintf("-lock-timeout=%fs", c.Flags.LockTimeout.Seconds()))
			}
		case NoBackend:
			if c.Flags.NoBackend {
				out = append(out, "-backend=false")
			}
		case NoInput:
			if c.Flags.NoInput {
				out = append(out, "-input=false")
			}
		case NoColor:
			if c.Flags.NoColor {
				out = append(out, "-no-color")
			}
		case NoLock:
			if c.Flags.NoLock {
				out = append(out, "-lock=false")
			}
		case NoDownload:
			if c.Flags.NoDownload {
				out = append(out, "-get=false")
			}
		case NoRefresh:
			if c.Flags.NoRefresh {
				out = append(out, "-refresh=false")
			}
		case OutPath:
			if c.Flags.OutPath != "" {
				out = append(out, fmt.Sprintf("-out=%s", c.Flags.OutPath))
			}
		case Parallelism:
			if c.Flags.Parallelism > 0 {
				out = append(out, fmt.Sprintf("-parallelism=%d", c.Flags.Parallelism))
			}
		case RefreshOnly:
			if c.Flags.RefreshOnly {
				out = append(out, "-refresh-only")
			}
		case StateFilePath:
			if c.Flags.StateFilePath != "" {
				out = append(out, fmt.Sprintf("-state=%s", c.Flags.StateFilePath))
			}
		case Upgrade:
			if c.Flags.Upgrade {
				out = append(out, "-upgrade")
			}
		default:
		}
	}

	return out
}
