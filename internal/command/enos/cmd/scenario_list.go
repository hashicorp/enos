package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/ui"
)

func newScenarioListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [DIRECTORY]",
		Short: "List Enos quality requirement scenarios",
		Long:  "List Enos quality requirement scenarios",
		RunE:  runScenarioListCmd,
	}
}

func runScenarioListCmd(cmd *cobra.Command, args []string) error {
	path, err := os.Getwd()
	if err != nil {
		return err
	}

	if len(args) > 0 {
		ep, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}

		path = ep
	}

	decoder := flightplan.NewDecoder(
		flightplan.WithDecoderDirectory(path),
	)

	diags := decoder.Parse()
	if diags.HasErrors() {
		return diags
	}

	fp, diags := decoder.Decode()
	if diags.HasErrors() {
		return diags
	}

	if len(fp.Scenarios) != 0 {
		header := []string{"scenario"}
		rows := [][]string{{""}} // add a padding row
		for _, scenario := range fp.Scenarios {
			rows = append(rows, []string{scenario.Name})
		}

		ui.RenderTable(os.Stdout, header, rows)
	}

	// Print warnings
	if len(diags) > 0 {
		for _, diag := range diags {
			fmt.Fprintln(os.Stderr, diag)
		}
	}

	return nil
}
