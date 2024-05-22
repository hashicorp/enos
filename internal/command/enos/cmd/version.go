// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

func newVersionCmd() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Enos version",
		Long:  "Enos version",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			res, err := rootState.enosConnection.Client.GetVersion(
				ctx, &pb.GetVersionRequest{},
			)
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
