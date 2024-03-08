// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"

	"github.com/hashicorp/enos/internal/ui/terminal"
)

// Command is a functional options wrapper around exec.Cmd.
type Command struct {
	Name           string
	EnvPassthrough bool
	ExecOpts       []ExecOpt
}

// Opt is a functional options wrapper around Command.
type Opt func(*Command)

// ExecOpt is a functional options wrapper around *exec.Cmd. These options
// are applied to the Cmd instance before it is run.
type ExecOpt func(*exec.Cmd)

// NewCommand creates a new command.
func NewCommand(name string, opts ...Opt) *Command {
	cmd := &Command{
		Name:     name,
		ExecOpts: []ExecOpt{},
	}

	for _, opt := range opts {
		opt(cmd)
	}

	return cmd
}

// WithUI takes a UI and configures the cmd's STDERR, STDIN, and STDOUT
// to use the UI's outputs.
func WithUI(ui *terminal.UI) Opt {
	return func(cmd *Command) {
		cmd.ExecOpts = append(cmd.ExecOpts, func(ecmd *exec.Cmd) {
			ecmd.Stderr = ui.Stderr
			ecmd.Stdout = ui.Stdout
			ecmd.Stdin = ui.Stdin
		})
	}
}

// WithArgs sets the command arguments.
func WithArgs(args ...string) Opt {
	return func(cmd *Command) {
		cmd.ExecOpts = append(cmd.ExecOpts, func(ecmd *exec.Cmd) {
			ecmd.Args = append(ecmd.Args, args...)
		})
	}
}

// WithDir sets the command directory.
func WithDir(dir string) Opt {
	return func(cmd *Command) {
		cmd.ExecOpts = append(cmd.ExecOpts, func(ecmd *exec.Cmd) {
			ecmd.Dir = dir
		})
	}
}

// WithEnv sets the command environment variables.
func WithEnv(vars map[string]string) Opt {
	return func(cmd *Command) {
		env := []string{}
		for k, v := range vars {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		sort.Strings(env)
		cmd.ExecOpts = append(cmd.ExecOpts, func(ecmd *exec.Cmd) {
			ecmd.Env = append(ecmd.Env, env...)
		})
	}
}

// WithEnvPassthrough passes the current process environment to the command.
func WithEnvPassthrough() Opt {
	return func(cmd *Command) {
		cmd.EnvPassthrough = true
	}
}

// Cmd takes a context and returns an instance of *exec.Cmd.
func (c *Command) Cmd(ctx context.Context) *exec.Cmd {
	//nolint:gosec // G204 We know we're passing through to exec.
	cmd := exec.CommandContext(ctx, c.Name)
	if c.EnvPassthrough {
		cmd.Env = os.Environ()
	}
	for _, opt := range c.ExecOpts {
		opt(cmd)
	}

	return cmd
}

// Run takes a context.Context and executes itself. It returns an instance of
// *exec.Cmd and an error.
func (c *Command) Run(ctx context.Context) (*exec.Cmd, error) {
	cmd := c.Cmd(ctx)
	err := cmd.Run()

	return cmd, err
}
