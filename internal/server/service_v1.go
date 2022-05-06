package server

import (
	"context"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	"github.com/hashicorp/go-hclog"
)

var _ pb.EnosServiceServer = (*ServiceV1)(nil)

// ServiceV1 is the enos.v1.ServerService
type ServiceV1 struct {
	log hclog.Logger

	grpcListenAddr net.Addr
	gprcListener   net.Listener
	gprcServer     *grpc.Server
}

// Opt is a functional option
type Opt func(*ServiceV1) error

// WithGRPCListenURL configures the gRPC listener address from a given URL
func WithGRPCListenURL(url *url.URL) Opt {
	return func(s *ServiceV1) error {
		var err error
		s.grpcListenAddr, err = ListenAddr(url)
		return err
	}
}

// WithLogger configures the logger
func WithLogger(log hclog.Logger) Opt {
	return func(s *ServiceV1) error {
		s.log = log
		return nil
	}
}

// ListenAddr returns a server listen address from a URL
func ListenAddr(url *url.URL) (net.Addr, error) {
	switch url.Scheme {
	case "unix", "unixpacket":
		return net.ResolveUnixAddr(url.Scheme, url.Host)
	default:
		addr := url.Host
		if idx := strings.IndexByte(addr, ':'); idx < 0 {
			addr += ":3205"
		}
		return net.ResolveTCPAddr("tcp", addr)
	}
}

// New takes options and returns an instance of ServiceV1
func New(opts ...Opt) (*ServiceV1, error) {
	svc := &ServiceV1{
		log: hclog.NewNullLogger(),
	}

	var err error
	for _, opt := range opts {
		err = opt(svc)
		if err != nil {
			return nil, err
		}
	}

	grpcLogger := svc.log.Named("grpc")
	grpcOpts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			logUnaryInterceptor(grpcLogger, false),
		),
		grpc.ChainStreamInterceptor(
			logStreamInterceptor(grpcLogger, false),
		),
		grpc.KeepaliveEnforcementPolicy(
			keepalive.EnforcementPolicy{
				MinTime:             20 * time.Second,
				PermitWithoutStream: true,
			},
		),
	}
	svc.gprcServer = grpc.NewServer(grpcOpts...)

	return svc, nil
}

// Start takes a context and starts the server. It will block until an error
// is encountered.
func (s *ServiceV1) Start(ctx context.Context) error {
	log := s.log.Named("server")

	// Only interrupt and kill are guaranteed on all OSes. We'll pipe through
	// unix signals we care about until such a time as we cannot.
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, os.Kill, syscall.SIGHUP, syscall.SIGTERM)
	defer cancel()

	wg := sync.WaitGroup{}

	errC := make(chan error, 1)
	wg.Add(1)
	go func() {
		errC <- s.start(ctx)
		wg.Done()
	}()
	defer wg.Wait()

	select {
	case err := <-errC:
		if err != nil {
			log.Error("server encountered an error", "error", err)
		}
		return err
	case <-ctx.Done():
		s.Stop()
		log.Error(ctx.Err().Error())
		return ctx.Err()
	}
}

// Start starts the service
func (s *ServiceV1) start(ctx context.Context) error {
	// Reflection makes it easy to see what methods are on a server via
	// grpcurl.
	reflection.Register(s.gprcServer)

	// Register ourselves with the instance of the gRPC server
	pb.RegisterEnosServiceServer(s.gprcServer, s)

	// Start our listener
	s.log.Named("server").Info("starting gRPC server",
		"network", s.grpcListenAddr.Network(),
		"addr", s.grpcListenAddr.String(),
	)
	var err error
	s.gprcListener, err = net.Listen(s.grpcListenAddr.Network(), s.grpcListenAddr.String())
	if err != nil {
		return status.Error(codes.FailedPrecondition, err.Error())
	}

	return s.gprcServer.Serve(s.gprcListener)
}

// Stop stops the service
func (s *ServiceV1) Stop() {
	log := s.log.Named("server")

	stopC := make(chan struct{})
	go func() {
		defer close(stopC)
		log.Debug("Attemping graceful stop")
		s.gprcServer.GracefulStop()
	}()

	select {
	case <-stopC:
		log.Debug("Server gracefully stopped")
	case <-time.After(5 * time.Second):
		log.Debug("Forcing stop because 5 second deadline has elapsed")
		s.gprcServer.Stop()
	}
}
