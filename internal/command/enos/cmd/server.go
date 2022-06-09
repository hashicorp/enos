package cmd

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/hashicorp/enos/internal/client"
	"github.com/hashicorp/enos/internal/server"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	"github.com/hashicorp/go-hclog"
)

// startGRPCServer starts the enos gRPC server and returns the instance and client to
// the server.
func startGRPCServer(ctx context.Context, timeout time.Duration) (*server.ServiceV1, pb.EnosServiceClient, error) {
	url, err := url.Parse(rootState.listenGRPC)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing listen-grpc value: %w", err)
	}

	log := hclog.New(&hclog.LoggerOptions{
		Name:  "enos",
		Level: hclog.LevelFromString(rootState.logLevelServer),
	})

	svr, err := server.New(
		server.WithGRPCListenURL(url),
		server.WithLogger(log),
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
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
			defer cancel()
			var enosClient pb.EnosServiceClient
			enosClient, err = client.Connect(ctx,
				client.WithGRPCListenURL(url),
				client.WithLogger(log),
			)
			if err != nil {
				log.Debug("waiting for connection to server: %w", err)
			} else {
				return svr, enosClient, nil
			}
		}
	}
}
