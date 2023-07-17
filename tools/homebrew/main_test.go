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

	p, err := filepath.Abs("./support/enos_0.0.1_SHA256SUMS")
	require.NoError(t, err)

	sums, err := readMetadata(p)
	require.NoError(t, err)

	expected := &metadata{
		DarwinAMD64SHA: "788eda2be1887fa13b2aca2a5bcad4535278310946c8f6f68fa561e72f7a351b",
		DarwinARM64SHA: "15d82aa03f5585966bc747e04ee31c025391dc1c80b3ba6419c95f6b764eebbd",
		LinuxAMD64SHA:  "dc1f9597b024b59bf444d894c633c3fe796b7d57d74d7983ad57c4d7d37a516d",
		LinuxARM64SHA:  "2a1677b83d6ec24038ef949420afa79269a405e35396aefec432412233cfc251",
		Version:        "0.0.1",
		VersionTag:     "v0.0.1",
	}
	require.Equal(t, expected, sums)
}

// Ensure that the template renders correctly.
func Test_renderHomebrewFormulaTemplate(t *testing.T) {
	t.Parallel()

	p, err := filepath.Abs("./support/enos_0.0.1_SHA256SUMS")
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
