// Copyright 2018-2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
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
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	models "github.com/firecracker-microvm/firecracker-go-sdk/client/models"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	client "github.com/firecracker-microvm/firecracker-go-sdk/client"
	ops "github.com/firecracker-microvm/firecracker-go-sdk/client/operations"
	"github.com/go-openapi/strfmt"
	"github.com/sirupsen/logrus"
)

const skipIntegrationTest = "SKIP_INTEG_TEST"

var kernelImagePath = "./vmlinux"

func init() {
	if v := os.Getenv("KERNELIMAGE"); len(v) != 0 {
		kernelImagePath = v
	}
}

func TestFireCTL(t *testing.T) {
	if len(os.Getenv(skipIntegrationTest)) > 0 {
		t.Skip()
	}

	const firectlName = "./firectl"
	if _, err := os.Stat(firectlName); os.IsNotExist(err) {
		t.Fatalf("Missing firectl binary, %s", firectlName)
	}

	if _, err := os.Stat(kernelImagePath); os.IsNotExist(err) {
		t.Fatalf("Missing kernel image, %s", kernelImagePath)
	}

	const rootDrivePath = "/dev/null"
	if _, err := os.Stat(rootDrivePath); os.IsNotExist(err) {
		t.Fatalf("Missing root drive, %s", rootDrivePath)
	}

	ctx := context.Background()
	ctlCtx, cancelFn := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancelFn()

	const socketpath = "./integration_test.sock"
	firectlArgs := []string{
		"-s",
		socketpath,
		"--kernel",
		kernelImagePath,
		"--root-drive",
		rootDrivePath,
	}
	cmd := exec.CommandContext(ctlCtx, firectlName, firectlArgs...)
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to run command: %v", err)
	}

	defer func() {
		if err := os.Remove(socketpath); err != nil {
			t.Log("Failed to remove socket path file")
		}

		// We signal SIGQUIT to the Firecracker process due to the application
		// using SIGQUIT to shutdown with
		if err := cmd.Process.Signal(syscall.SIGQUIT); err != nil {
			t.Log("Failed to kill firectl process")
		}
	}()

	client := client.NewHTTPClient(strfmt.NewFormats())
	transport := firecracker.NewUnixSocketTransport(socketpath, logrus.NewEntry(logrus.New()), false)
	client.SetTransport(transport)

	valid := false
	interval := 50 * time.Millisecond
	payload := &models.InstanceInfo{}
	for i := 0; i < 10; i++ {
		resp, err := client.Operations.DescribeInstance(ops.NewDescribeInstanceParams())
		if err != nil {
			t.Log("Failed to communicate over socket file:", err)

			time.Sleep(interval)
			continue
		}

		payload = resp.Payload
		if len(*resp.Payload.State) != 0 &&
			*payload.State != models.InstanceInfoStateUninitialized {
			valid = true
			break
		}

		time.Sleep(interval)
	}

	if !valid {
		t.Errorf("VM failed to initialize with last state of %q. Can firecracker successfully launch a VM?", *payload.State)
	}
}
