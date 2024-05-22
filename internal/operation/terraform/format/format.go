// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package format provides functions for formatting Terraform and cty as
// text. The implemenation comes from a modified version of the repl package
// in Terraform.
package format

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// TerraformOutput takes a terraform executor output metadata and returns it as
// human friendly formatted string.
func TerraformOutput(out *pb.Terraform_Command_Output_Response_Meta, indent int) (string, error) {
	if out.GetSensitive() {
		return "(sensitive)", nil
	}

	typ, err := ctyjson.UnmarshalType(out.GetType())
	if err != nil {
		return "", err
	}

	val, err := ctyjson.Unmarshal(out.GetValue(), typ)
	if err != nil {
		return "", err
	}

	return Value(val, indent), nil
}

// Value formats a value in a way that resembles Terraform language syntax
// and uses the type conversion functions where necessary to indicate exactly
// what type it is given, so that equality test failures can be quickly
// understood.
func Value(v cty.Value, indent int) string {
	if !v.IsKnown() {
		return "(unknown)"
	}
	if v.IsNull() {
		ty := v.Type()
		switch {
		case ty == cty.DynamicPseudoType:
			return "null"
		case ty == cty.String:
			return "tostring(null)"
		case ty == cty.Number:
			return "tonumber(null)"
		case ty == cty.Bool:
			return "tobool(null)"
		case ty.IsListType():
			return fmt.Sprintf("tolist(null) /* of %s */", ty.ElementType().FriendlyName())
		case ty.IsSetType():
			return fmt.Sprintf("toset(null) /* of %s */", ty.ElementType().FriendlyName())
		case ty.IsMapType():
			return fmt.Sprintf("tomap(null) /* of %s */", ty.ElementType().FriendlyName())
		default:
			return fmt.Sprintf("null /* %s */", ty.FriendlyName())
		}
	}

	ty := v.Type()
	switch {
	case ty.IsPrimitiveType():
		switch ty {
		case cty.String:
			if formatted, isMultiline := formatMultilineString(v, indent); isMultiline {
				return formatted
			}

			return strconv.Quote(v.AsString())
		case cty.Number:
			bf := v.AsBigFloat()

			return bf.Text('f', -1)
		case cty.Bool:
			if v.True() {
				return "true"
			}

			return "false"
		}
	case ty.IsObjectType():
		return formatMappingValue(v, indent)
	case ty.IsTupleType():
		return formatSequenceValue(v, indent)
	case ty.IsListType():
		return fmt.Sprintf("tolist(%s)", formatSequenceValue(v, indent))
	case ty.IsSetType():
		return fmt.Sprintf("toset(%s)", formatSequenceValue(v, indent))
	case ty.IsMapType():
		return fmt.Sprintf("tomap(%s)", formatMappingValue(v, indent))
	}

	// Should never get here because there are no other types
	return fmt.Sprintf("%#v", v)
}

func formatMultilineString(v cty.Value, indent int) (string, bool) {
	str := v.AsString()
	lines := strings.Split(str, "\n")
	if len(lines) < 2 {
		return "", false
	}

	// If the value is indented, we use the indented form of heredoc for readability.
	operator := "<<"
	if indent > 0 {
		operator = "<<-"
	}

	// Default delimiter is "End Of Text" by convention
	delimiter := "EOT"

OUTER:
	for {
		// Check if any of the lines are in conflict with the delimiter. The
		// parser allows leading and trailing whitespace, so we must remove it
		// before comparison.
		for _, line := range lines {
			// If the delimiter matches a line, extend it and start again
			if strings.TrimSpace(line) == delimiter {
				delimiter = delimiter + "_"

				continue OUTER
			}
		}

		// None of the lines match the delimiter, so we're ready
		break
	}

	// Write the heredoc, with indentation as appropriate.
	var buf strings.Builder

	buf.WriteString(operator)
	buf.WriteString(delimiter)
	for _, line := range lines {
		buf.WriteByte('\n')
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(line)
	}
	buf.WriteByte('\n')
	buf.WriteString(strings.Repeat(" ", indent))
	buf.WriteString(delimiter)

	return buf.String(), true
}

func formatMappingValue(v cty.Value, indent int) string {
	var buf strings.Builder
	count := 0
	buf.WriteByte('{')
	indent += 2
	for it := v.ElementIterator(); it.Next(); {
		count++
		k, v := it.Element()
		buf.WriteByte('\n')
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(Value(k, indent))
		buf.WriteString(" = ")
		buf.WriteString(Value(v, indent))
	}
	indent -= 2
	if count > 0 {
		buf.WriteByte('\n')
		buf.WriteString(strings.Repeat(" ", indent))
	}
	buf.WriteByte('}')

	return buf.String()
}

func formatSequenceValue(v cty.Value, indent int) string {
	var buf strings.Builder
	count := 0
	buf.WriteByte('[')
	indent += 2
	for it := v.ElementIterator(); it.Next(); {
		count++
		_, v := it.Element()
		buf.WriteByte('\n')
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(Value(v, indent))
		buf.WriteByte(',')
	}
	indent -= 2
	if count > 0 {
		buf.WriteByte('\n')
		buf.WriteString(strings.Repeat(" ", indent))
	}
	buf.WriteByte(']')

	return buf.String()
}
