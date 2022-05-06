package client

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/hashicorp/enos/internal/server"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	"github.com/hashicorp/go-hclog"
)

// Config is the client configuration
type Config struct {
	Addr net.Addr
	Log  hclog.Logger
}

// Opt is a client configuration option
type Opt func(*Config) error

// WithGRPCListenURL sets the listener address from a URL
func WithGRPCListenURL(url *url.URL) Opt {
	return func(c *Config) error {
		var err error
		c.Addr, err = server.ListenAddr(url)
		return err
	}
}

// WithLogger sets client logger
func WithLogger(log hclog.Logger) Opt {
	return func(c *Config) error {
		c.Log = log
		return nil
	}
}

// Connect takes a context and options and returns a new enos.v1 client
func Connect(ctx context.Context, opts ...Opt) (pb.EnosServiceClient, error) {
	c := &Config{
		Log: hclog.NewNullLogger(),
	}

	var err error
	for _, opt := range opts {
		err = opt(c)
		if err != nil {
			return nil, err
		}
	}

	if c.Addr == nil {
		return nil, fmt.Errorf("you must supply a server address")
	}

	grpcOpts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(
			keepalive.ClientParameters{
				Time:                30 * time.Second,
				PermitWithoutStream: true,
			},
		),
	}

	c.Log.Named("client").Debug("connecting to server", "addr", c.Addr.String())
	conn, err := grpc.DialContext(ctx, c.Addr.String(), grpcOpts...)
	if err != nil {
		return nil, err
	}

	return pb.NewEnosServiceClient(conn), nil
}
