// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package memory

import "context"

type Statable interface {
	Free() uint64
	Total() uint64
	Used() uint64
	Available() uint64
	String() string
}

type Config struct {
	GC bool
}

type Opt func(*Config)

func WithGC() Opt {
	return func(c *Config) {
		c.GC = true
	}
}

func DefaultConfig() *Config {
	return &Config{GC: false}
}

// Stat gathers memory information from the host system and returns the information.
func Stat(ctx context.Context, opts ...Opt) (Statable, error) {
	return stat(ctx, opts...)
}
