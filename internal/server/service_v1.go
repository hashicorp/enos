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
	gprcListener   net.Listener
	gprcServer     *grpc.Server

	operator operation.Operator
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
	svc.gprcServer = grpc.NewServer(grpcOpts...)

	return svc, nil
}

// Start takes a context and starts the server. It will block until an error
// is encountered.
func (s *ServiceV1) Start(ctx context.Context) error {
	// Only interrupt and kill are guaranteed on all OSes. We'll pipe through
	// unix signals we care about until such a time as we cannot.
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, os.Kill, syscall.SIGHUP, syscall.SIGTERM)
	defer cancel()

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
		return err
	case <-ctx.Done():
		err := multierror.Append(ctx.Err(), s.Stop()).ErrorOrNil()
		if err != nil {
			s.log.Error(err.Error())
		}
		return err
	}
}

// Serve start the gRPC server and operator
func (s *ServiceV1) serve(ctx context.Context) error {
	// Reflection makes it easy to see what methods are on a server via
	// grpcurl.
	reflection.Register(s.gprcServer)

	// Register ourselves with the instance of the gRPC server
	pb.RegisterEnosServiceServer(s.gprcServer, s)

	// Start our listener
	s.log.Info("starting gRPC server",
		"network", s.grpcListenAddr.Network(),
		"addr", s.grpcListenAddr.String(),
	)
	var err error
	s.gprcListener, err = net.Listen(s.grpcListenAddr.Network(), s.grpcListenAddr.String())
	if err != nil {
		return err
	}

	err = s.operator.Start(ctx)
	if err != nil {
		return err
	}

	return s.gprcServer.Serve(s.gprcListener)
}

// Stop stops the service
func (s *ServiceV1) Stop() error {
	var wg sync.WaitGroup
	var err error
	stopC := make(chan struct{})

	go func() {
		defer close(stopC)
		s.log.Info("Attemping graceful gRPC server stop")
		s.gprcServer.GracefulStop()
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
			s.gprcServer.Stop()
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

	scenarios, decRes := decodeAndFilter(
		ws.GetFlightplan(),
		f,
	)

	if baseReq.GetValue() == nil {
		diags = append(diags, diagnostics.FromErr(fmt.Errorf(
			"failed to dispatch operation because operation request value has not been set",
		))...)
	}

	if len(scenarios) == 0 {
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

	for _, scenario := range scenarios {
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
