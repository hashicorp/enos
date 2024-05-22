// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package basic

import (
	"io"
	"os"

	"github.com/hashicorp/enos/internal/ui/terminal"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// View is our basic terminal CLI view.
type View struct {
	settings *pb.UI_Settings
	ui       *terminal.UI
	stdout   io.ReadWriteCloser
	stderr   io.ReadWriteCloser
}

// Opt is a functional option.
type Opt func(*View)

// New takes options and returns a new basic.View.
func New(opts ...Opt) (*View, error) {
	v := &View{}

	for _, opt := range opts {
		opt(v)
	}

	uiOpts := []terminal.Opt{
		terminal.WithStdin(os.Stdin),
		terminal.WithStdout(os.Stdout),
		terminal.WithStderr(os.Stderr),
	}
	if v.settings != nil {
		uiOpts = append(uiOpts, terminal.WithLevel(v.settings.GetLevel()))

		if v.settings.GetIsTty() {
			uiOpts = append(uiOpts, terminal.WithColor(true))
		}

		if v.settings.GetWidth() > 0 {
			uiOpts = append(uiOpts, terminal.WithWidth(uint(v.settings.GetWidth())))
		}

		if v.settings.GetStdoutPath() != "" {
			f, err := os.OpenFile(v.settings.GetStdoutPath(), os.O_RDWR|os.O_CREATE, 0o755)
			if err != nil {
				return nil, err
			}
			v.stdout = f

			uiOpts = append(uiOpts, terminal.WithStdout(f))
		}

		if v.settings.GetStderrPath() != "" {
			f, err := os.OpenFile(v.settings.GetStderrPath(), os.O_RDWR|os.O_CREATE, 0o755)
			if err != nil {
				return nil, err
			}
			v.stderr = f

			uiOpts = append(uiOpts, terminal.WithStderr(f))
		}
	}

	v.ui = terminal.NewUI(uiOpts...)

	return v, nil
}

// WithUISettings sets the ui settings.
func WithUISettings(ui *pb.UI_Settings) Opt {
	return func(view *View) {
		view.settings = ui
	}
}

// Settings returns the views UI settings.
func (v *View) Settings() *pb.UI_Settings {
	return v.settings
}

// Close closes any open file handles.
func (v *View) Close() error {
	if v.stderr != nil {
		err := v.stderr.Close()
		if err != nil {
			return err
		}
	}

	if v.stdout == nil {
		return nil
	}

	return v.stdout.Close()
}

func (v *View) opStatusString(status pb.Operation_Status) string {
	var res string
	tty := v.settings.GetIsTty()

	switch status {
	case pb.Operation_STATUS_CANCELLED:
		res = "❌"
		if !tty {
			res = "cancelled!"
		}
	case pb.Operation_STATUS_COMPLETED:
		res = "✅"
		if !tty {
			res = "success!"
		}
	case pb.Operation_STATUS_COMPLETED_WARNING:
		res = "⚠️"
		if !tty {
			res = "success! (warnings present)"
		}
	case pb.Operation_STATUS_FAILED:
		res = "❌"
		if !tty {
			res = "failed!"
		}
	case pb.Operation_STATUS_RUNNING_WARNING:
		res = "⚠️"
		if !tty {
			res = "running (warnings present)"
		}
	case pb.Operation_STATUS_RUNNING:
		res = "🚀"
		if !tty {
			res = "running"
		}
	case pb.Operation_STATUS_WAITING:
		res = "⏳"
		if !tty {
			res = "waiting"
		}
	case pb.Operation_STATUS_QUEUED:
		res = "⏳"
		if !tty {
			res = "queued"
		}
	case pb.Operation_STATUS_UNSPECIFIED, pb.Operation_STATUS_UNKNOWN:
		res = "⁉️"
		if !tty {
			res = "unknown"
		}
	default:
		res = "⁉️"
		if !tty {
			res = "unknown"
		}
	}

	return res
}

func (v *View) UI() *terminal.UI {
	if v == nil {
		return nil
	}

	return v.ui
}

func (v *View) writeMsg(
	status pb.Operation_Status,
	msg string,
) {
	if msg == "" {
		return
	}

	if status == pb.Operation_STATUS_FAILED {
		v.ui.Error(msg)

		return
	}

	if status == pb.Operation_STATUS_COMPLETED_WARNING && v.settings.GetLevel() >= pb.UI_Settings_LEVEL_WARN {
		v.ui.Warn(msg)

		return
	}

	if v.settings.GetLevel() >= pb.UI_Settings_LEVEL_INFO {
		v.ui.Info(msg)

		return
	}

	v.ui.Info(msg)
}
