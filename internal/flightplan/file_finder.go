// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
)

// FlightPlanFileNamePattern is what file names match valid enos configuration files.
var (
	FlightPlanFileNamePattern = regexp.MustCompile(`^enos[-\w]*?\.hcl$`)
	VariablesNamePattern      = regexp.MustCompile(`^enos[-\w]*?\.vars\.hcl$`)
)

// RawFiles are a map of flightplan configuration files and their contents.
type RawFiles map[string][]byte

// FindRawFiles scans a directory for files matching the given pattern and
// returns the loaded raw files.
func FindRawFiles(dir string, pattern *regexp.Regexp) (RawFiles, error) {
	var err error
	files := RawFiles{}

	err = filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("scanning for Enos configuration: %w", err)
		}

		// We're only going a single level deep for now so we can ingnore directories
		if info.IsDir() {
			// Always skip the directory unless it's the root we're walking
			if path != dir {
				return filepath.SkipDir
			}
		}

		if !pattern.MatchString(info.Name()) {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		bytes, err := io.ReadAll(f)
		if err != nil {
			return err
		}
		files[path] = bytes

		return nil
	})

	return files, err
}

// LoadRawFiles takes a slice of paths and returns the loaded raw files.
func LoadRawFiles(paths []string) (RawFiles, error) {
	rawFiles := RawFiles{}
	var err error

	for _, path := range paths {
		if !filepath.IsAbs(path) {
			path, err = filepath.Abs(path)
			if err != nil {
				return nil, err
			}
		}

		path, err := filepath.EvalSymlinks(path)
		if err != nil {
			return nil, err
		}

		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		bytes, err := io.ReadAll(f)
		if err != nil {
			return nil, err
		}
		rawFiles[path] = bytes
	}

	return rawFiles, nil
}
