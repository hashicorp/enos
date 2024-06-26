// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
	"github.com/hashicorp/go-hclog"
)

// Connection is a client connection to the enos server.
type Connection struct {
	Addr     net.Addr
	Client   pb.EnosServiceClient
	Log      hclog.Logger
	Level    pb.UI_Settings_Level
	DialOpts []grpc.DialOption
}

// Opt is a client connection option.
type Opt func(*Connection) error

// WithGRPCListenURL sets the listener address from a URL.
func WithGRPCListenURL(url *url.URL) Opt {
	return func(c *Connection) error {
		var err error
		switch url.Scheme {
		case "unix", "unixpacket":
			c.Addr, err = net.ResolveUnixAddr(url.Scheme, url.Host)
		default:
			addr := url.Host
			if p := url.Port(); p != "" {
				addr = fmt.Sprintf("%s:%s", addr, p)
			}
			c.Addr, err = net.ResolveTCPAddr("tcp", addr)
		}

		return err
	}
}

func WithGRPCListenAddr(addr net.Addr) Opt {
	return func(c *Connection) error {
		c.Addr = addr

		return nil
	}
}

func WithGRPCDialOpts(opts ...grpc.DialOption) Opt {
	return func(c *Connection) error {
		if len(opts) < 1 {
			return nil
		}

		if c.DialOpts == nil {
			c.DialOpts = opts
			return nil
		}

		c.DialOpts = append(c.DialOpts, opts...)

		return nil
	}
}

// WithLogger sets client logger.
func WithLogger(log hclog.Logger) Opt {
	return func(c *Connection) error {
		c.Log = log

		return nil
	}
}

// WithLogLevel sets client log level.
func WithLogLevel(lvl pb.UI_Settings_Level) Opt {
	return func(c *Connection) error {
		c.Level = lvl

		return nil
	}
}

// Trace writes an hclog.Logger style message at a "trace" level.
func (c *Connection) Trace(msg string, args ...any) {
	if c.Level == pb.UI_Settings_LEVEL_TRACE {
		c.Log.Debug(msg, args...)
	}
}

// Connect takes a context and options and returns a new connection.
func Connect(ctx context.Context, opts ...Opt) (*Connection, error) {
	c := &Connection{
		DialOpts: []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithKeepaliveParams(
				keepalive.ClientParameters{
					Time:                30 * time.Second,
					PermitWithoutStream: true,
				},
			),
		},
		Log: hclog.NewNullLogger().Named("client"),
	}

	var err error
	for _, opt := range opts {
		err = opt(c)
		if err != nil {
			return nil, err
		}
	}

	if c.Addr == nil {
		return nil, errors.New("you must supply a server address")
	}

	c.Trace("connecting to server", "addr", c.Addr.String())
	conn, err := grpc.NewClient(c.Addr.String(), c.DialOpts...)
	if err != nil {
		return nil, err
	}
	c.Client = pb.NewEnosServiceClient(conn)

	return c, nil
}
