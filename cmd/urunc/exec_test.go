// Copyright (c) 2023-2026, Nubificus LTD
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// execHelperEnvVar re-invokes the test binary as the real urunc CLI entry
// point. execCommand's Action calls os.Exit (matching runc's own convention
// for exec failures), so it can't be exercised in-process without killing
// the test run; running it as a subprocess is the standard way to test
// os.Exit paths.
const execHelperEnvVar = "URUNC_TEST_EXEC_HELPER"

// TestExecCommandRegistered runs the real CLI entry point with "urunc exec"
// as a subprocess and checks that urfave/cli recognizes it as a known
// command rather than falling back to the "No help topic for 'exec'"
// unknown-command error that motivated this stub in the first place (see
// https://github.com/urunc-dev/urunc/issues/882 and the logs quoted in
// https://github.com/urunc-dev/urunc/issues/135).
func TestExecCommandRegistered(t *testing.T) {
	assert.NotNil(t, execCommand)
	assert.Equal(t, "exec", execCommand.Name)

	if os.Getenv(execHelperEnvVar) == "1" {
		os.Args = []string{"urunc", "exec", "some-container-id"}
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestExecCommandRegistered") //#nosec G204 -- os.Args[0] is this test binary re-invoking itself, not external input
	cmd.Env = append(os.Environ(), execHelperEnvVar+"=1")
	out, _ := cmd.CombinedOutput()

	assert.NotContains(t, strings.ToLower(string(out)), "no help topic")
}

// TestExecCommandFlagsAcceptGoRuncShape asserts that execCommand declares
// the flags containerd-shim-runc-v2/go-runc actually pass when invoking
// exec (`exec --process <spec.json> [--console-socket ...] [--detach]
// [--pid-file ...] <container-id>`), so parsing doesn't fail the way it did
// before this stub existed (see the "-info" parsing failure referenced in
// https://github.com/urunc-dev/urunc/issues/882).
func TestExecCommandFlagsAcceptGoRuncShape(t *testing.T) {
	names := make(map[string]bool)
	for _, f := range execCommand.Flags {
		for _, n := range f.Names() {
			names[n] = true
		}
	}
	for _, want := range []string{"process", "console-socket", "detach", "pid-file"} {
		assert.True(t, names[want], "expected exec command to define flag %q", want)
	}
}

// TestExecCommandFailsFast runs "urunc exec <container-id>" in a subprocess
// and checks that it exits immediately with the dedicated "not supported"
// exit code and a clear error message, instead of hanging or returning the
// generic exit code 1 used for other urunc CLI errors.
func TestExecCommandFailsFast(t *testing.T) {
	if os.Getenv(execHelperEnvVar) == "1" {
		os.Args = []string{"urunc", "exec", "some-container-id"}
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestExecCommandFailsFast") //#nosec G204 -- os.Args[0] is this test binary re-invoking itself, not external input
	cmd.Env = append(os.Environ(), execHelperEnvVar+"=1")
	out, runErr := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	if assert.ErrorAs(t, runErr, &exitErr, "expected urunc exec to exit with a non-zero status") {
		assert.Equal(t, execNotSupportedExitCode, exitErr.ExitCode(),
			"expected the dedicated exec-not-supported exit code, not the generic exit code 1")
	}
	assert.Contains(t, string(out), "exec is not supported by urunc")
}

// TestExecCommandDoesNotPanicOnGoRuncArgs exercises the exact argument shape
// go-runc/containerd-shim-runc-v2 use when invoking exec, to confirm CLI
// parsing accepts it cleanly rather than crashing or reporting an unknown
// flag.
func TestExecCommandDoesNotPanicOnGoRuncArgs(t *testing.T) {
	if os.Getenv(execHelperEnvVar) == "1" {
		os.Args = []string{
			"urunc", "exec",
			"--process", "/tmp/does-not-matter.json",
			"--console-socket", "/tmp/does-not-matter.sock",
			"--pid-file", "/tmp/does-not-matter.pid",
			"some-container-id",
		}
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestExecCommandDoesNotPanicOnGoRuncArgs") //#nosec G204 -- os.Args[0] is this test binary re-invoking itself, not external input
	cmd.Env = append(os.Environ(), execHelperEnvVar+"=1")
	out, runErr := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	if assert.ErrorAs(t, runErr, &exitErr, "expected urunc exec to exit with a non-zero status") {
		assert.Equal(t, execNotSupportedExitCode, exitErr.ExitCode())
	}
	lower := strings.ToLower(string(out))
	assert.NotContains(t, lower, "flag provided but not defined")
	assert.NotContains(t, lower, "panic")
}
