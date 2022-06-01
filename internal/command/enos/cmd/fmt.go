package cmd

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

var fmtCfg = &pb.FormatRequest_Config{}

func newFmtCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fmt [ARGS] [PATH]",
		Short: "Format Enos configuration",
		Long:  "Format Enos configuration or variables files. When given a path Enos will format the file or files in the given directory. If not path exists and no matching files are found it will assume STDIN as the source. When given the check flag it will return 3 if a diff exists.",
		RunE:  runFmtCmd,
		Args:  cobra.MaximumNArgs(1),
	}

	cmd.PersistentFlags().BoolVarP(&fmtCfg.Write, "write", "w", true, "Write changes to files. Always disabled if using STDIN or --check")
	cmd.PersistentFlags().BoolVarP(&fmtCfg.List, "list", "l", true, "List changed files. Always disabled if using STDIN")
	cmd.PersistentFlags().BoolVarP(&fmtCfg.Check, "check", "c", false, "Check if the input is formatted. Exit will be 0 for success, 1 for an error, 3 for success but changed files")
	cmd.PersistentFlags().BoolVarP(&fmtCfg.Diff, "diff", "d", false, "Display the unified diff for files that change")

	return cmd
}

// runFmtCmd is the function that formats the enos configuration.
func runFmtCmd(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true // We'll handle it from here cobra

	readEnosFiles := func(path string) (map[string][]byte, []*pb.Diagnostic) {
		var err error

		if path == "" {
			path, err = os.Getwd()
			if err != nil {
				return nil, diagnostics.FromErr(err)
			}
		}

		path, err = filepath.Abs(path)
		if err != nil {
			return nil, diagnostics.FromErr(err)
		}

		path, err = filepath.EvalSymlinks(path)
		if err != nil {
			return nil, diagnostics.FromErr(err)
		}

		f, err := os.Open(path)
		if err != nil {
			return nil, diagnostics.FromErr(err)
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			return nil, diagnostics.FromErr(err)
		}

		if info.IsDir() {
			fpFiles, err := flightplan.FindRawFiles(path, flightplan.FlightPlanFileNamePattern)
			if err != nil {
				return nil, diagnostics.FromErr(err)
			}

			varsFiles, err := flightplan.FindRawFiles(path, flightplan.VariablesNamePattern)
			if err != nil {
				return nil, diagnostics.FromErr(err)
			}

			files := fpFiles
			for path, bytes := range varsFiles {
				files[path] = bytes
			}

			return files, nil
		}

		content, err := io.ReadAll(f)
		if err != nil {
			return nil, diagnostics.FromErr(err)
		}

		return map[string][]byte{
			path: content,
		}, nil
	}

	var err error
	req := &pb.FormatRequest{Config: fmtCfg, Files: map[string][]byte{}}
	res := &pb.FormatResponse{}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	argP := ""
	if len(args) > 0 {
		argP = args[0]
	}

	// Scan the path for files
	if argP != "-" {
		req.Files, res.Diagnostics = readEnosFiles(argP)
		if diagnostics.HasErrors(res.GetDiagnostics()) {
			return ui.ShowFormat(fmtCfg, res)
		}
	}

	/// Scan STDIN for content if we've been told to use STDIN either implicitly
	// of explicitly.
	if (argP == "-" || argP == "") && len(req.Files) == 0 {
		bytes, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			res.Diagnostics = diagnostics.FromErr(err)
			return ui.ShowFormat(fmtCfg, res)
		}
		req.Files["STDIN"] = bytes
	}

	res, err = enosClient.Format(ctx, req)
	if err != nil {
		return err
	}

	return ui.ShowFormat(fmtCfg, res)
}
