package cmd

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

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
	url, err := url.Parse(rootState.listenGRPC)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing listen-grpc value: %w", err)
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
		server.WithGRPCListenURL(url),
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

	go func() {
		_ = svr.Start(ctx)
	}()

	waitCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Millisecond)
	for {
		select {
		case <-waitCtx.Done():
			return nil, nil, fmt.Errorf("waiting for server to start: %w", err)
		default:
		}

		select {
		case <-waitCtx.Done():
			return nil, nil, fmt.Errorf("waiting for server to start: %w", err)
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(
				context.Background(),
				1*time.Millisecond,
			)
			defer cancel()
			var enosConnection *client.Connection
			enosConnection, err = client.Connect(ctx,
				client.WithGRPCListenURL(url),
				client.WithLogger(clientLog),
			)
			if err == nil {
				return svr, enosConnection, nil
			}
		}
	}
}
