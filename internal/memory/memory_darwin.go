// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build darwin

package memory

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

var _ Statable = (*DarwinMemory)(nil)

type DarwinMemory struct {
	free     uint64
	inactive uint64
	pageSize uint64
	memSize  uint64
}

func stat(ctx context.Context, opts ...Opt) (*DarwinMemory, error) {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("stat memory: %w", ctx.Err())
	default:
	}

	var err error
	res := &DarwinMemory{}
	res.pageSize = uint64(unix.Getpagesize())
	res.memSize, err = unix.SysctlUint64("hw.memsize")
	if err != nil {
		return nil, err
	}

	// Set takes a string value, converts it to a uint64, and sets the value
	set := func(val string, dst *uint64) error {
		value, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return err
		}
		*dst = value * res.pageSize

		return nil
	}

	if cfg.GC {
		runtime.GC()
	}

	cmd := exec.CommandContext(ctx, "vm_stat")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.Trim(parts[1], " .")
		switch k {
		case "Pages free":
			if err := set(v, &res.free); err != nil {
				return nil, err
			}
		case "Pages inactive":
			if err := set(v, &res.inactive); err != nil {
				return nil, err
			}
		default:
		}
	}

	return res, nil
}

func (m *DarwinMemory) Total() uint64 {
	if m == nil {
		return 0
	}

	return m.memSize
}

func (m *DarwinMemory) Used() uint64 {
	if m == nil {
		return 0
	}

	return m.memSize - (m.free + m.inactive)
}

func (m *DarwinMemory) Free() uint64 {
	if m == nil {
		return 0
	}

	return m.free
}

func (m *DarwinMemory) Available() uint64 {
	if m == nil {
		return 0
	}

	return m.free + m.inactive
}

func (m *DarwinMemory) String() string {
	if m == nil {
		return ""
	}

	return fmt.Sprintf("Total: %d, Used: %d, Free: %d, Available: %d", m.Total(), m.Used(), m.Free(), m.Available())
}
