package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

func newVersionCmd() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Enos version",
		Long:  "Enos version",
		RunE: func(cmd *cobra.Command, args []string) error {
			// We don't start the server automatically for version, only scenario
			// sub-commands
			svr, client, err := startGRPCServer(context.Background(), 5*time.Second)
			if err != nil {
				return err
			}
			defer svr.Stop()

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			res, err := client.GetVersion(ctx, &pb.GetVersionRequest{})
			if err != nil {
				return err
			}

			return ui.ShowVersion(versionArgs.all, res)
		},
	}

	versionCmd.PersistentFlags().BoolVar(&versionArgs.all, "all", false, "display all version information")

	return versionCmd
}

var versionArgs struct {
	all bool
}
