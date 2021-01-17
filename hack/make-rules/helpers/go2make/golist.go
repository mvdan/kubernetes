package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

type Package struct {
	Dir        string
	ImportPath string
	Name       string
	Doc        string
	Target     string
	Goroot     bool
	Standard   bool
	Export     string
	DepOnly    bool

	GoFiles []string

	Imports []string
	Deps    []string

	Error *PackageError
}

type PackageError struct {
	ImportStack []string
	Pos         string
	Err         string
}

// listPackages is a wrapper for 'go list -json -e', which can take arbitrary
// environment variables and arguments as input. The working directory can be
// fed by adding $PWD to env; otherwise, it will default to the current
// directory.
//
// Since -e is used, the returned error will only be non-nil if a JSON result
// could not be obtained. Such examples are if the Go command is not installed,
// or if invalid flags are used as arguments.
//
// Errors encountered when loading packages will be returned for each package,
// in the form of PackageError. See 'go help list'.
func listPackages(ctx context.Context, dir string, env []string, args ...string) (pkgs []*Package, finalErr error) {
	goArgs := append([]string{"list", "-json", "-e"}, args...)
	cmd := exec.CommandContext(ctx, "go", goArgs...)
	cmd.Env = env
	cmd.Dir = dir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf
	defer func() {
		if stderrBuf.Len() > 0 {
			// TODO: wrap? but the format is backwards, given that
			// stderr is likely multi-line
			finalErr = fmt.Errorf("%v\n%s", finalErr, stderrBuf.Bytes())
		}
	}()

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	dec := json.NewDecoder(stdout)
	for dec.More() {
		var pkg Package
		if err := dec.Decode(&pkg); err != nil {
			return nil, err
		}
		pkgs = append(pkgs, &pkg)
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}
	return pkgs, nil
}
