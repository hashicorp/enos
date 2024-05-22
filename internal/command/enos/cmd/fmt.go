// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

var fmtCfg = &pb.FormatRequest_Config{}

func newFmtCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fmt [ARGS] [PATH]",
		Short: "Format Enos configuration",
		Long:  "Format Enos configuration or variables files. When given a path to a file Enos will format it. When given a path to a directory it will search for files that match enos.hcl, enos.vars.hcl, or enos-*.hcl. If no path is given it will perform a file search from the current working directly. If no files are found or - is passed as the path it will assume STDIN is the source to be formatted.",
		RunE:  runFmtCmd,
		Args:  cobra.MaximumNArgs(1),
	}

	cmd.PersistentFlags().BoolVarP(&fmtCfg.Write, "write", "w", true, "Write changes to files. Always disabled if using STDIN or --check")
	cmd.PersistentFlags().BoolVarP(&fmtCfg.List, "list", "l", true, "List changed files. Always disabled if using STDIN")
	cmd.PersistentFlags().BoolVarP(&fmtCfg.Check, "check", "c", false, "Check if the input is formatted. Exit will be 0 for success, 1 for an error, 3 for success but files would be changed")
	cmd.PersistentFlags().BoolVarP(&fmtCfg.Diff, "diff", "d", false, "Display the unified diff for files that change")
	cmd.PersistentFlags().BoolVarP(&fmtCfg.Recursive, "recursive", "r", false, "Recursively scan the directory and format the files")

	return cmd
}

// runFmtCmd is the function that formats the enos configuration.
func runFmtCmd(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true // We'll handle it from here cobra

	readEnosFiles := func(path string) ([]*pb.FormatRequest_File, []*pb.Diagnostic) {
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

		// We have a path that points to a single file
		if !info.IsDir() {
			content, err := io.ReadAll(f)
			if err != nil {
				return nil, diagnostics.FromErr(err)
			}

			return []*pb.FormatRequest_File{
				{Path: path, Body: content},
			}, nil
		}

		files := []*pb.FormatRequest_File{}
		readRawFiles := func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			fpFiles, err := flightplan.FindRawFiles(path, flightplan.FlightPlanFileNamePattern)
			if err != nil {
				return err
			}
			for path, bytes := range fpFiles {
				files = append(files, &pb.FormatRequest_File{
					Path: path,
					Body: bytes,
				})
			}

			varsFiles, err := flightplan.FindRawFiles(path, flightplan.VariablesNamePattern)
			if err != nil {
				return err
			}

			for path, bytes := range varsFiles {
				files = append(files, &pb.FormatRequest_File{
					Path: path,
					Body: bytes,
				})
			}

			return nil
		}

		if fmtCfg.GetRecursive() {
			err = filepath.Walk(path, readRawFiles)
		} else {
			err = readRawFiles(path, nil, nil)
		}
		if err != nil {
			return nil, diagnostics.FromErr(err)
		}

		return files, nil
	}

	var err error
	req := &pb.FormatRequest{Config: fmtCfg, Files: []*pb.FormatRequest_File{}}
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

	// Scan STDIN for content if we've been told to use STDIN either implicitly of explicitly.
	if (argP == "-" || argP == "") && len(req.GetFiles()) == 0 {
		bytes, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			res.Diagnostics = diagnostics.FromErr(err)
			return ui.ShowFormat(fmtCfg, res)
		}
		req.Files = []*pb.FormatRequest_File{
			{Path: "STDIN", Body: bytes},
		}
	}

	res, err = rootState.enosConnection.Client.Format(ctx, req)
	if err != nil {
		return err
	}

	return ui.ShowFormat(fmtCfg, res)
}
