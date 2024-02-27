// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd.AddCommand(newCreateFormulaCommand())

	err := rootCmd.Execute()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

//go:embed support/enos.rb.tmpl
var formulaTemplate embed.FS

var t = template.Must(template.ParseFS(formulaTemplate, "support/enos.rb.tmpl"))

var rootCmd = cobra.Command{
	Use:   "homebrew",
	Short: "Create an Enos Homebrew Formula for a release",
}

var createFormulaConfigs struct {
	path       string
	version    string
	versionTag string
	outPath    string
}

type metadata struct {
	DarwinAMD64SHA string
	DarwinARM64SHA string
	LinuxAMD64SHA  string
	LinuxARM64SHA  string
	Version        string
	VersionTag     string
}

type errInvalidFilenameReason string

var (
	errInvalidFilenameReasonArch    = errInvalidFilenameReason("Unknown architecture")
	errInvalidFilenameReasonPlaform = errInvalidFilenameReason("Unknown platform")
)

type errInvalidFilename struct {
	reason errInvalidFilenameReason
}

func (e *errInvalidFilename) Error() string {
	msg := "Filename should be in format <product>_<version>_<linux|darwin>_<amd64|arm64>.zip"
	if e.reason != "" {
		return fmt.Sprintf("%s. %s", e.reason, msg)
	}

	return msg
}

func newCreateFormulaCommand() *cobra.Command {
	createFormula := &cobra.Command{
		Use:  "create",
		RunE: createFormula,
	}

	createFormula.PersistentFlags().StringVarP(&createFormulaConfigs.path, "path", "p", "", "the path to the SHA265SUMS file")
	createFormula.PersistentFlags().StringVarP(&createFormulaConfigs.outPath, "outpath", "o", "", "the path to the output file")

	return createFormula
}

// Get the version, the version tag, and the SHASUMS of each asset from the SHASUMS file.
func readMetadata(path string) (*metadata, error) {
	metadata := &metadata{}
	input, err := os.Open(path)
	if err != nil {
		return metadata, err
	}
	defer input.Close()

	counter := 0
	var tempSha string
	var tempPlatformArch string

	// Get SHASUMS and version from file contents
	scanner := bufio.NewScanner(input)
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		counter++
		if strings.Contains(scanner.Text(), ".zip") {
			tempPlatformArch = scanner.Text()
		} else {
			tempSha = scanner.Text()
		}

		if counter%2 == 0 {
			parts := strings.Split(tempPlatformArch, "_")
			metadata.Version = parts[1]
			switch parts[2] {
			case "darwin":
				switch parts[3] {
				case "arm64.zip":
					metadata.DarwinARM64SHA = tempSha
				case "amd64.zip":
					metadata.DarwinAMD64SHA = tempSha
				default:
					return metadata, &errInvalidFilename{errInvalidFilenameReasonArch}
				}
			case "linux":
				switch parts[3] {
				case "arm64.zip":
					metadata.LinuxARM64SHA = tempSha
				case "amd64.zip":
					metadata.LinuxAMD64SHA = tempSha
				default:
					return metadata, &errInvalidFilename{errInvalidFilenameReasonArch}
				}
			default:
				return metadata, &errInvalidFilename{errInvalidFilenameReasonPlaform}
			}
		}
	}

	// Get the version tag
	metadata.VersionTag = "v" + metadata.Version

	return metadata, nil
}

// Execute the template with the metadata values to `dest`.
func renderHomebrewFormulaTemplate(dest io.Writer, metadataPath string) error {
	metadata, err := readMetadata(metadataPath)
	if err != nil {
		return fmt.Errorf("reading metadata: %s", err.Error())
	}

	// Execute the template using metadata values from the SHASUMS file
	err = t.Execute(dest, metadata)
	if err != nil {
		return fmt.Errorf("executing template: %s", err.Error())
	}

	return nil
}

// Write the executed template to an output file.
func createFormula(cmd *cobra.Command, args []string) error {
	buf := bytes.Buffer{}

	// Create a new formula template and fill in the values from the SHASUMS input file
	err := renderHomebrewFormulaTemplate(&buf, createFormulaConfigs.path)
	if err != nil {
		fmt.Printf("rendering Homebrew formula template: %s\n", err.Error())
	}

	// Create an output file
	output, err := os.Create(createFormulaConfigs.outPath)
	if err != nil {
		fmt.Printf("creating file: %s\n", err.Error())
	}

	// Write the template to the output file
	_, err = buf.WriteTo(output)
	if err != nil {
		fmt.Printf("writing updated template to output file: %s\n", err.Error())
	}

	return nil
}
