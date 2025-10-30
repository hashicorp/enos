// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package basic

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/ui/status"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

func indent(width int, in string) string {
	return strings.Repeat(" ", width) + in
}

func printLine(w io.Writer, offset int, in string) {
	fmt.Fprintf(w, "%s\n", indent(offset, in))
}

func printMultiLine(w io.Writer, offset int, in string) error {
	scanner := bufio.NewScanner(strings.NewReader(in))
	for scanner.Scan() {
		fmt.Fprintf(w, "%s\n", indent(offset, scanner.Text()))
	}

	return scanner.Err()
}

// ShowScenarioOutline shows an outline of scenarios.
func (v *View) ShowScenarioOutline(res *pb.OutlineScenariosResponse) error {
	b := new(strings.Builder)

	errToDiags := func(err error) error {
		if err == nil {
			return nil
		}
		res.Diagnostics = diagnostics.Concat(res.GetDiagnostics(), diagnostics.FromErr(err))

		return err
	}

	for i, out := range res.GetOutlines() {
		if i != 0 {
			b.WriteString("\n")
		}

		fmt.Fprintf(b, "%s\n", out.GetScenario().GetId().GetName())
		description := out.GetScenario().GetId().GetDescription()
		if description != "" {
			printLine(b, 2, "Description:")
			if err := errToDiags(printMultiLine(b, 4, description)); err != nil {
				return err
			}
		}

		for i, variant := range out.GetMatrix().GetVectors() {
			if i == 0 {
				printLine(b, 2, "Variants:")
			}
			// We assume a well formatted matrix from the service
			key := variant.GetElements()[0].GetKey()
			fmt.Fprint(b, indent(4, fmt.Sprintf("- %s: [", key)))
			for i := range variant.GetElements() {
				if i != 0 {
					fmt.Fprint(b, ", ")
				}
				fmt.Fprintf(b, "%s", variant.GetElements()[i].GetValue())
			}
			fmt.Fprint(b, "]\n")
		}

		for i, quality := range out.GetVerifies() {
			if i == 0 {
				printLine(b, 2, "Verifies:")
			}
			printLine(b, 4, fmt.Sprintf("- %s:", quality.GetName()))
			description := quality.GetDescription()
			if description != "" {
				if err := errToDiags(printMultiLine(b, 6, description)); err != nil {
					return err
				}
			}
		}

		for i, step := range out.GetSteps() {
			if i == 0 {
				printLine(b, 2, "Steps:")
			}
			// We assume a well formatted matrix from the service
			printLine(b, 4, "- "+step.GetName())
			description := step.GetDescription()
			if description != "" {
				printLine(b, 6, "Description:")
				if err := errToDiags(printMultiLine(b, 8, description)); err != nil {
					return err
				}
			}

			for i, quality := range step.GetVerifies() {
				if i == 0 {
					printLine(b, 6, "Verifies:")
				}
				printLine(b, 8, fmt.Sprintf("- %s:", quality.GetName()))
				description := quality.GetDescription()
				if description != "" {
					if err := errToDiags(printMultiLine(b, 10, description)); err != nil {
						return err
					}
				}
			}
		}
	}

	output := b.String()
	if output != "" {
		v.ui.Output(output)
	}
	v.WriteDiagnostics(res.GetDecode().GetDiagnostics())
	v.WriteDiagnostics(res.GetDiagnostics())

	return status.OutlineScenarios(v.settings.GetFailOnWarnings(), res)
}
