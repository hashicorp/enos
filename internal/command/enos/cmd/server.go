package cmd

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"google.golang.org/grpc"

	"github.com/hashicorp/enos/internal/client"
	"github.com/hashicorp/enos/internal/operation"
	"github.com/hashicorp/enos/internal/server"
	"github.com/hashicorp/enos/internal/state"
	"github.com/hashicorp/go-hclog"
)

// startServer starts the enos gRPC server and returns the instance and client to
// the server.
func startServer(
	ctx context.Context,
	timeout time.Duration,
) (
	*server.ServiceV1,
	*client.Connection,
	error,
) {
	listenURL, err := url.Parse(rootState.grpcListenAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing grpc-listen value: %w", err)
	}

	svrLog := hclog.New(&hclog.LoggerOptions{
		Name:  "enos",
		Level: hclog.LevelFromString(rootState.logLevelServer),
	}).Named("server")

	// hclog doesn't support a trace level but we want it for client side. This
	// allows us to have pretty debug logging but manually handle trace logging
	// if we want.
	cll := strings.ToLower(rootState.logLevel)
	switch cll {
	case "t", "a", "trace":
		cll = "debug"
	default:
	}
	clientLog := hclog.New(&hclog.LoggerOptions{
		Name:  "enos",
		Level: hclog.LevelFromString(cll),
	}).Named("client")

	svr, err := server.New(
		server.WithGRPCListenURL(listenURL),
		server.WithGRPCServerOptions(
			grpc.MaxRecvMsgSize(rootState.grpcMaxRecv),
			grpc.MaxSendMsgSize(rootState.grpcMaxSend),
		),
		server.WithLogger(svrLog),
		server.WithOperator(
			operation.NewLocalOperator(
				operation.WithLocalOperatorLog(svrLog.Named("operator")),
				operation.WithLocalOperatorState(state.NewInMemoryState()),
				operation.WithLocalOperatorConfig(rootState.operatorConfig),
			),
		),
	)
	if err != nil {
		return nil, nil, err
	}

	cfg, err := svr.Start(ctx)
	if err != nil {
		return nil, nil, err
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Millisecond)
	for {
		select {
		case <-waitCtx.Done():
			return nil, nil, fmt.Errorf("waiting for server to start on address %s:  %w", cfg.ListenAddr.String(), err)
		default:
		}

		select {
		case <-waitCtx.Done():
			return nil, nil, fmt.Errorf("waiting for server to start on address %s:  %w", cfg.ListenAddr.String(), err)
		case <-ticker.C:
			connCtx, cancel := context.WithTimeout(
				ctx,
				50*time.Millisecond,
			)
			defer cancel()
			var enosConnection *client.Connection
			enosConnection, err = client.Connect(connCtx,
				client.WithGRPCDialOpts(
					grpc.WithDefaultCallOptions(
						grpc.MaxCallRecvMsgSize(rootState.grpcMaxRecv),
						grpc.MaxCallSendMsgSize(rootState.grpcMaxSend),
					),
				),
				client.WithGRPCListenAddr(cfg.ListenAddr),
				client.WithLogger(clientLog),
			)
			if err == nil {
				return svr, enosConnection, nil
			}
		}
	}
}
