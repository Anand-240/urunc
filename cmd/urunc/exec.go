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
	"context"
	"errors"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

// ErrExecNotSupported is returned whenever "urunc exec" is invoked. urunc
// containers run as unikernels/VMs: there is no general-purpose process
// namespace to attach a new process to once the unikernel has started, so
// unlike runc, urunc cannot execute an additional process inside a running
// container.
var ErrExecNotSupported = errors.New("exec is not supported by urunc: urunc containers run as " +
	"unikernels/VMs and do not support attaching a new process after container start")

// execNotSupportedExitCode is the exit code urunc uses to report that exec
// is unsupported. It intentionally matches runc's own convention of using a
// dedicated exit code (255, see runc's exec.go) for exec failures, instead
// of falling through to the generic exit code 1 used for other urunc CLI
// errors. This lets callers such as containerd/go-runc, which invoke this
// command as `urunc exec --process <spec.json> [flags] <container-id>`,
// distinguish "exec is not supported" from a generic urunc failure, rather
// than retrying indefinitely as if it were transient.
const execNotSupportedExitCode = 255

// execCommand intentionally implements just enough of the runc-compatible
// "exec" interface required by containerd-shim-runc-v2/go-runc to be
// invoked without a CLI parsing failure:
//
//	urunc exec --process <spec.json> [--console-socket <path>] [--detach] [--pid-file <path>] <container-id>
//
// It does not attempt to actually execute a process inside a running
// unikernel: that would require attaching to a VM/unikernel's execution
// environment, which has no equivalent of a container's process namespace
// to exec into. Instead, it fails fast with a clear, distinct error so
// callers can tell "not supported" apart from "urunc is broken" and stop
// retrying instead of hanging indefinitely.
//
// See https://github.com/urunc-dev/urunc/issues/882 for the motivating
// failure mode: Argo Workflows relies on `kubectl exec` into a sidecar
// container to signal step completion, and because Kubernetes RuntimeClass
// is pod-scoped rather than per-container, that exec call was silently
// routed through urunc's shim and retried forever with an opaque failure.
var execCommand = &cli.Command{
	Name:      "exec",
	Usage:     "not supported: urunc unikernels cannot exec an additional process after start",
	ArgsUsage: `<container-id>`,
	Description: `The exec command is part of the OCI runtime CLI interface expected by
containerd-shim-runc-v2/go-runc, but urunc does not support it: a running
unikernel/VM has no general-purpose process namespace to attach a new
process to.

This command exists so that callers relying on the standard OCI runtime CLI
interface (e.g. "kubectl exec" against a urunc-scheduled pod) get a clear,
immediate "not supported" error instead of urunc's CLI parser failing on an
undefined command and callers retrying forever.`,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "process",
			Usage: "path to the process.json describing the process to exec (unused: exec is not supported)",
		},
		&cli.StringFlag{
			Name:  "console-socket",
			Usage: "path to an AF_UNIX socket for the console pty (unused: exec is not supported)",
		},
		&cli.BoolFlag{
			Name:  "detach",
			Usage: "detach from the container's process (unused: exec is not supported)",
		},
		&cli.StringFlag{
			Name:  "pid-file",
			Usage: "file to write the process id to (unused: exec is not supported)",
		},
	},
	Action: func(_ context.Context, cmd *cli.Command) error {
		logrus.WithField("command", "EXEC").WithField("args", os.Args).Debug("urunc INVOKED")

		if err := checkArgs(cmd, 1, minArgs); err != nil {
			return err
		}

		containerID := cmd.Args().First()
		if err := validateID(containerID); err != nil {
			return err
		}

		logrus.WithField("container", containerID).Warn(ErrExecNotSupported)

		// Exit with a dedicated code (see execNotSupportedExitCode) rather
		// than returning the error to be handled by main's generic,
		// exit-code-1 error path.
		fatalWithCode(ErrExecNotSupported, execNotSupportedExitCode)
		return nil // unreachable: fatalWithCode calls os.Exit
	},
}
