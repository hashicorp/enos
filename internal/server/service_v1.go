package server

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/operation"
	"github.com/hashicorp/enos/internal/proto"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-multierror"
)

var _ pb.EnosServiceServer = (*ServiceV1)(nil)

// ServiceV1 is the enos.v1.ServerService
type ServiceV1 struct {
	log hclog.Logger

	grpcListenAddr net.Addr
	grpcListener   net.Listener
	grpcServer     *grpc.Server

	operator operation.Operator
}

// ServiceConfig is the running service config.
type ServiceConfig struct {
	ListenAddr net.Addr
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

// WithOperator configures the servers operation operator
func WithOperator(op operation.Operator) Opt {
	return func(s *ServiceV1) error {
		s.operator = op
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
		log:      hclog.NewNullLogger(),
		operator: operation.NewLocalOperator(),
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
	svc.grpcServer = grpc.NewServer(grpcOpts...)

	return svc, nil
}

// Start takes a context and starts the server. It returns any immediate errors and a service config.
// Fatal errors encountered will automatically stop the server.
func (s *ServiceV1) Start(ctx context.Context) (*ServiceConfig, error) {
	if s.grpcListenAddr == nil {
		return nil, fmt.Errorf("unable to start gRPC service: you must provider a listen address")
	}

	s.log.Info("starting gRPC server",
		"network", s.grpcListenAddr.Network(),
		"addr", s.grpcListenAddr.String(),
	)

	// Only interrupt and kill are guaranteed on all OSes. We'll pipe through unix signals we care
	// about until such a time as we cannot.
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt, os.Kill, syscall.SIGHUP, syscall.SIGTERM)

	err := s.startListener(ctx)
	if err != nil {
		return nil, err
	}

	err = s.startOperator(ctx)
	if err != nil {
		return nil, err
	}

	serve := func() {
		wg := sync.WaitGroup{}

		errC := make(chan error, 1)
		wg.Add(1)
		go func() {
			errC <- s.serve(ctx)
			wg.Done()
		}()

		defer wg.Wait()

		select {
		case err := <-errC:
			if err != nil {
				s.log.Error("server encountered an error", "error", err)
			}
			return
		case <-ctx.Done():
			err := multierror.Append(ctx.Err(), s.Stop()).ErrorOrNil()
			if err != nil {
				s.log.Error(err.Error())
			}
			return
		}
	}

	go serve()

	return &ServiceConfig{
		ListenAddr: s.grpcListener.Addr(),
	}, nil
}

// startListener starts the gRPC server listener
func (s *ServiceV1) startListener(ctx context.Context) error {
	// Reflection makes it easy to see what methods are on a server via grpcurl.
	reflection.Register(s.grpcServer)

	// Register ourselves with the instance of the gRPC server
	pb.RegisterEnosServiceServer(s.grpcServer, s)

	s.log.Debug("starting gRPC server listener",
		"network", s.grpcListenAddr.Network(),
		"addr", s.grpcListenAddr.String(),
	)
	var err error
	s.grpcListener, err = net.Listen(s.grpcListenAddr.Network(), s.grpcListenAddr.String())
	if err != nil {
		s.log.Error("failed to start gRPC server listener",
			"network", s.grpcListenAddr.Network(),
			"addr", s.grpcListenAddr.String(),
			"error", err,
		)
		return err
	}

	s.log.Debug("gRPC server listener is listening",
		"network", s.grpcListenAddr.Network(),
		"configured_addr", s.grpcListenAddr.String(),
		"resolved_addr", s.grpcListener.Addr().String(),
	)

	return nil
}

// startOperator starts the service operator
func (s *ServiceV1) startOperator(ctx context.Context) error {
	s.log.Debug("starting service operator")

	err := s.operator.Start(ctx)
	if err != nil {
		s.log.Error("failed to start service operator",
			"error", err,
		)
		return err
	}

	s.log.Debug("service operator running")
	return nil
}

// serve services requests. It will block until an error is encountered.
func (s *ServiceV1) serve(ctx context.Context) error {
	s.log.Debug("serving gRPC requests",
		"network", s.grpcListenAddr.Network(),
		"addr", s.grpcListener.Addr(),
	)

	return s.grpcServer.Serve(s.grpcListener)
}

// Stop stops the service
func (s *ServiceV1) Stop() error {
	var wg sync.WaitGroup
	var err error
	stopC := make(chan struct{})

	go func() {
		defer close(stopC)
		s.log.Info("Attemping graceful gRPC server stop")
		s.grpcServer.GracefulStop()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if s.operator != nil {
			s.log.Info("Stopping operator")
			err = s.operator.Stop()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-stopC:
			s.log.Info("Server gracefully stopped")
		case <-time.After(5 * time.Second):
			s.log.Info("Forcing stop because 5 second deadline has elapsed")
			s.grpcServer.Stop()
		}
	}()

	wg.Wait()

	return err
}

// dispatch takes a workspace, filter, and base operation request. It decodes,
// the flightplan, filters the scenarios, and dispatches an operation for each
// scenario that matches the filter. It returns the decoder response, a slice
// of operation references, and any diagnostics if dispatching isn't possible.
// The base operation must include a valid operation type.
func (s *ServiceV1) dispatch(
	f *pb.Scenario_Filter,
	baseReq *pb.Operation_Request,
) (
	[]*pb.Diagnostic,
	*pb.DecodeResponse,
	[]*pb.Ref_Operation,
) {
	diags := []*pb.Diagnostic{}
	refs := []*pb.Ref_Operation{}

	ws := baseReq.GetWorkspace()
	if ws == nil {
		diags = append(diags, diagnostics.FromErr(fmt.Errorf(
			"unable to dispatch operations for requests without the required workspace",
		))...)
	}

	fp, decRes := decodeFlightPlan(
		ws.GetFlightplan(),
		flightplan.DecodeModeFull,
		f,
	)

	if baseReq.GetValue() == nil {
		diags = append(diags, diagnostics.FromErr(fmt.Errorf(
			"failed to dispatch operation because operation request value has not been set",
		))...)
	}

	if len(fp.Scenarios) == 0 {
		filter, err := flightplan.NewScenarioFilter(
			flightplan.WithScenarioFilterDecode(f),
		)
		if err != nil {
			diags = append(diags, diagnostics.FromErr(err)...)
		} else {
			diags = append(diags, diagnostics.FromErr(fmt.Errorf(
				"no scenarios found matching filter '%s'", filter.String(),
			))...)
		}
	}

	if diagnostics.HasFailed(
		ws.GetTfExecCfg().GetFailOnWarnings(),
		diagnostics.Concat(diags, decRes.GetDiagnostics()),
	) {
		return diags, decRes, refs
	}

	for _, scenario := range fp.Scenarios {
		req := &pb.Operation_Request{}
		err := proto.Copy(baseReq, req)
		if err != nil {
			diags = append(diags, diagnostics.FromErr(err)...)
			continue
		}

		req.Scenario = scenario.Ref()
		ref, moreDiags := s.operator.Dispatch(req)
		diags = append(diags, moreDiags...)
		if ref != nil {
			refs = append(refs, ref)
		}
	}

	return diags, decRes, refs
}
