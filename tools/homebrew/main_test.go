// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Ensure that the SHASUMS file is parsed accurately.
func Test_decodeMetadata(t *testing.T) {
	t.Parallel()

	p, err := filepath.Abs("./support/enos_0.0.27_SHA256SUMS")
	require.NoError(t, err)

	sums, err := readMetadata(p)
	require.NoError(t, err)

	expected := &metadata{
		DarwinARM64SHA: "e10ffb026f933eef8d40b25dc864557b473da344346c5c6fd54ae28062dc4932",
		DarwinAMD64SHA: "155dc543097509fb7657867c468cc0a32ea62553eadfec34326f4633a577a043",
		LinuxARM64SHA:  "81ed9db7983b9dc5e78cb40607afb8227a002b714113c0768bcbbcfcf339f941",
		LinuxAMD64SHA:  "c4d4ae0d4de8315d18081a6719a634bbe263753e1f9d4a576c0b4650c2137af8",
		Version:        "0.0.27",
		VersionTag:     "v0.0.27",
	}
	require.Equal(t, expected, sums)
}

// Ensure that the template renders correctly.
func Test_renderHomebrewFormulaTemplate(t *testing.T) {
	t.Parallel()

	p, err := filepath.Abs("./support/enos_0.0.27_SHA256SUMS")
	require.NoError(t, err)

	buf := bytes.Buffer{}
	err = renderHomebrewFormulaTemplate(&buf, p)
	require.NoError(t, err)

	ep, err := filepath.Abs("./support/enos.rb")
	require.NoError(t, err)

	f, err := os.Open(ep)
	require.NoError(t, err)

	expected, err := io.ReadAll(f)
	require.NoError(t, err)

	require.Equal(t, string(expected), buf.String())
}
