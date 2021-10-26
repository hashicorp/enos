package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "enos",
	Short: "Enos is your friendly neighborhood test runner",
	Long:  "Enos is a one stop shop for defining and executing complex test scenarios",
}

// Execute executes enos
func Execute() {
	rootCmd.AddCommand(newVersionCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
