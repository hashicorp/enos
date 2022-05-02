package main

import (
	"bufio"
	"bytes"
	"errors"
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

const formulaTemplate = `require_relative "../Strategies/private_strategy"
class Enos < Formula
	desc "A tool for powering Software Quality as Code by writing Terraform-based quality requirement scenarios using a composable and shareable declarative language."
	homepage "https://github.com/hashicorp/enos"
	version "{{.Version}}"

	on_macos do
		if Hardware::CPU.arm?
			url "https://github.com/hashicorp/enos/releases/download/{{.VersionTag}}/enos_{{.Version}}_darwin_arm64.zip", :using => GitHubPrivateRepositoryReleaseDownloadStrategy
			sha256 "{{.DarwinARM64SHA}}"

			def install
				bin.install "enos"
			end
		end
		if Hardware::CPU.intel?
			url "https://github.com/hashicorp/enos/releases/download/{{.VersionTag}}/enos_{{.Version}}_darwin_amd64.zip", :using => GitHubPrivateRepositoryReleaseDownloadStrategy
			sha256 "{{.DarwinAMD64SHA}}"

			def install
				bin.install "enos"
			end
		end
	end

	on_linux do
		if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
			url "https://github.com/hashicorp/enos/releases/download/{{.VersionTag}}/enos_{{.Version}}_linux_arm64.zip", :using => GitHubPrivateRepositoryReleaseDownloadStrategy
			sha256 "{{.LinuxARM64SHA}}"

			def install
				bin.install "enos"
			end
		end
		if Hardware::CPU.intel?
			url "https://github.com/hashicorp/enos/releases/download/{{.VersionTag}}/enos_{{.Version}}_linux_amd64.zip", :using => GitHubPrivateRepositoryReleaseDownloadStrategy
			sha256 "{{.LinuxAMD64SHA}}"

			def install
				bin.install "enos"
			end
		end
	end
end
`

// Create a new template
var t = template.Must(template.New("formula").Parse(formulaTemplate))

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

func newCreateFormulaCommand() *cobra.Command {
	createFormula := &cobra.Command{
		Use:  "create",
		RunE: createFormula,
	}

	createFormula.PersistentFlags().StringVarP(&createFormulaConfigs.path, "path", "p", "", "the path to the SHA265SUMS file")
	createFormula.PersistentFlags().StringVarP(&createFormulaConfigs.outPath, "outpath", "o", "", "the path to the output file")

	return createFormula
}

// Get the the version, the version tag, and the SHASUMS of each asset from the SHASUMS file
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
					return metadata, errors.New("Unknown arch. Filename should be in format <product>_<version>_<linux|darwin>_<amd64|arm64>.zip")
				}
			case "linux":
				switch parts[3] {
				case "arm64.zip":
					metadata.LinuxARM64SHA = tempSha
				case "amd64.zip":
					metadata.LinuxAMD64SHA = tempSha
				default:
					return metadata, errors.New("Unknown arch. Filename should be in format <product>_<version>_<linux|darwin>_<amd64|arm64>.zip")
				}
			default:
				return metadata, errors.New("Unknown platform. Filename should be in format <product>_<version>_<linux|darwin>_<amd64|arm64>.zip")
			}
		}
	}

	// Get the version tag
	metadata.VersionTag = "v" + metadata.Version

	return metadata, nil
}

// Execute the template with the metadata values to `dest`
func renderHomebrewFormulaTemplate(dest io.Writer, metadataPath string) error {
	metadata, err := readMetadata(metadataPath)
	if err != nil {
		fmt.Println("reading metadata:", err)
	}

	// Execute the template using metadata values from the SHASUMS file
	err = t.Execute(dest, metadata)
	if err != nil {
		fmt.Println("executing template:", err)
	}
	return nil
}

// Write the executed template to an output file
func createFormula(cmd *cobra.Command, args []string) error {
	buf := bytes.Buffer{}

	// Create a new formula template and fill in the values from the SHASUMS input file
	err := renderHomebrewFormulaTemplate(&buf, createFormulaConfigs.path)
	if err != nil {
		fmt.Println("rendering Homebrew formula template:", err)
	}

	// Create an output file
	output, err := os.Create(createFormulaConfigs.outPath)
	if err != nil {
		fmt.Println("creating file: ", err)
	}

	// Write the template to the output file
	_, err = buf.WriteTo(output)
	if err != nil {
		fmt.Println("writing updated template to output file: ", err)
	}

	return nil
}
