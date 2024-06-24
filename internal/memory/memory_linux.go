// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build linux

package memory

import (
	"context"
	"fmt"
	"runtime"
	"syscall"
)

var _ Statable = (*LinuxMemory)(nil)

type LinuxMemory struct {
	*syscall.Sysinfo_t
}

func stat(ctx context.Context, opts ...Opt) (*LinuxMemory, error) {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("stat memory: %w", ctx.Err())
	default:
	}

	res := &LinuxMemory{
		Sysinfo_t: &syscall.Sysinfo_t{},
	}

	if cfg.GC {
		runtime.GC()
	}

	return res, syscall.Sysinfo(res.Sysinfo_t)
}

func (m *LinuxMemory) Total() uint64 {
	if m == nil || m.Sysinfo_t == nil {
		return 0
	}

	return m.Sysinfo_t.Totalram
}

func (m *LinuxMemory) Used() uint64 {
	if m == nil || m.Sysinfo_t == nil {
		return 0
	}

	return m.Sysinfo_t.Bufferram
}

func (m *LinuxMemory) Free() uint64 {
	if m == nil || m.Sysinfo_t == nil {
		return 0
	}

	return m.Sysinfo_t.Freeram
}

func (m *LinuxMemory) Available() uint64 {
	if m == nil || m.Sysinfo_t == nil {
		return 0
	}

	return m.Sysinfo_t.Totalram - m.Sysinfo_t.Bufferram
}

func (m *LinuxMemory) String() string {
	if m == nil {
		return ""
	}

	return fmt.Sprintf("Total: %d, Used: %d, Free: %d, Available: %d", m.Totalram, m.Bufferram, m.Freeram, m.Available())
}
