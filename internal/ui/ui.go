package ui

import (
	"io"

	"github.com/olekukonko/tablewriter"
)

// RenderTable does a basic render of table data to the desired writer
func RenderTable(w io.Writer, header []string, rows [][]string) {
	table := tablewriter.NewWriter(w)

	table.SetHeader(header)

	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetNoWhiteSpace(true)
	table.AppendBulk(rows)

	table.Render()
}
