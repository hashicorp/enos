package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/version"
)

func newVersionCmd() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Enos version",
		Long:  "Enos version",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !versionArgs.all {
				fmt.Printf("%s\n", version.Version)
			} else {
				fmt.Printf("Enos version: %s sha: %s\n", version.Version, version.GitSHA)
			}

			return nil
		},
	}

	versionCmd.PersistentFlags().BoolVar(&versionArgs.all, "all", false, "display all version information")

	return versionCmd
}

var versionArgs struct {
	all bool
}
