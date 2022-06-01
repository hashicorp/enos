package cmd

import (
	"context"
	"net/url"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/enos/internal/client"
	"github.com/hashicorp/enos/internal/server"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	"github.com/hashicorp/go-hclog"
)

// startGRPCServer starts the enos gRPC server and returns the instance and client to
// the server.
func startGRPCServer(ctx context.Context, timeout time.Duration) (*server.ServiceV1, pb.EnosServiceClient, error) {
	url, err := url.Parse(rootArgs.listenGRPC)
	if err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "parsing listen-grpc value: %s", err.Error())
	}

	log := hclog.New(&hclog.LoggerOptions{
		Name:  "enos",
		Level: hclog.LevelFromString(rootArgs.logLevelS),
	})

	svr, err := server.New(
		server.WithGRPCListenURL(url),
		server.WithLogger(log),
	)
	if err != nil {
		return nil, nil, status.Error(codes.Internal, err.Error())
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
			return nil, nil, status.Errorf(codes.DeadlineExceeded, "waiting for server to start: %s", err)
		default:
		}

		select {
		case <-waitCtx.Done():
			return nil, nil, status.Errorf(codes.DeadlineExceeded, "waiting for server to start: %s", err)
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
