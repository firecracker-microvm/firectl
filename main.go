// Copyright 2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//	http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package main

import (
	"context"
	"fmt"
	"os"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	flags "github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
)

const (
	terminalProgram = "xterm"
	// consoleXterm indicates that the machine's console should be presented in an xterm
	consoleXterm = "xterm"
	// consoleStdio indicates that the machine's console should re-use the parent's stdio streams
	consoleStdio = "stdio"
	// consoleFile inddicates that the machine's console should be presented in files rather than stdout/stderr
	consoleFile = "file"
	// consoleNone indicates that the machine's console IO should be discarded
	consoleNone = "none"

	// executableMask is the mask needed to check whether or not a file's
	// permissions are executable.
	executableMask = 0111
)

func main() {
	opts := options{}
	p := flags.NewParser(&opts, flags.Default)
	// if no args just print help
	if len(os.Args) == 1 {
		p.WriteHelp(os.Stderr)
		os.Exit(0)
	}
	_, err := p.ParseArgs(os.Args)
	if err != nil {
		// ErrHelp indicates that the help message was printed so we
		// can exit
		if val, ok := err.(*flags.Error); ok && val.Type == flags.ErrHelp {
			os.Exit(0)
		}
		p.WriteHelp(os.Stderr)
		os.Exit(1)
	}

	defer opts.Close()

	if err := runVMM(context.Background(), opts); err != nil {
		log.Fatalf(err.Error())
	}
}

// Run a vmm with a given set of options
func runVMM(ctx context.Context, opts options) error {
	// convert options to a firecracker config
	fcCfg, err := opts.getFirecrackerConfig()
	if err != nil {
		log.Errorf("Error: %s", err)
		return err
	}
	logger := log.New()

	if opts.Debug {
		log.SetLevel(log.DebugLevel)
		logger.SetLevel(log.DebugLevel)
	}

	vmmCtx, vmmCancel := context.WithCancel(ctx)
	defer vmmCancel()

	machineOpts := []firecracker.Opt{
		firecracker.WithLogger(log.NewEntry(logger)),
	}

	if len(opts.FcBinary) != 0 {
		finfo, err := os.Stat(opts.FcBinary)
		if os.IsNotExist(err) {
			return fmt.Errorf("Binary %q does not exist: %v", opts.FcBinary, err)
		}

		if err != nil {
			return fmt.Errorf("Failed to stat binary, %q: %v", opts.FcBinary, err)
		}

		if finfo.IsDir() {
			return fmt.Errorf("Binary, %q, is a directory", opts.FcBinary)
		} else if finfo.Mode()&executableMask == 0 {
			return fmt.Errorf("Binary, %q, is not executable. Check permissions of binary", opts.FcBinary)
		}

		cmd := firecracker.VMCommandBuilder{}.
			WithBin(opts.FcBinary).
			WithSocketPath(fcCfg.SocketPath).
			WithStdin(os.Stdin).
			WithStdout(os.Stdout).
			WithStderr(os.Stderr).
			Build(ctx)

		machineOpts = append(machineOpts, firecracker.WithProcessRunner(cmd))
	}

	m, err := firecracker.NewMachine(vmmCtx, fcCfg, machineOpts...)
	if err != nil {
		return fmt.Errorf("Failed creating machine: %s", err)
	}

	if opts.validMetadata != nil {
		m.EnableMetadata(opts.validMetadata)
	}

	if err := m.Start(vmmCtx); err != nil {
		return fmt.Errorf("Failed to start machine: %v", err)
	}
	defer m.StopVMM()

	// wait for the VMM to exit
	if err := m.Wait(vmmCtx); err != nil {
		return fmt.Errorf("Wait returned an error %s", err)
	}
	log.Printf("Start machine was happy")
	return nil
}
