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
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	models "github.com/firecracker-microvm/firecracker-go-sdk/client/models"
)

func TestGetFirecrackerConfig(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "firectl-test-drive-path")
	if err != nil {
		t.Error(err)
	}
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()
	cases := []struct {
		name        string
		opts        *options
		expectedErr func(error) (bool, error)
		outConfig   firecracker.Config
	}{
		{
			name: "Invalid metadata",
			opts: &options{
				FcMetadata: "{ invalid:json",
			},
			expectedErr: func(e error) (bool, error) {
				return strings.HasPrefix(e.Error(), errInvalidMetadata.Error()), errInvalidMetadata
			},
			outConfig: firecracker.Config{},
		},
		{
			name: "Invalid network config",
			opts: &options{
				FcNicConfig: []string{"no-slash"},
			},
			expectedErr: func(e error) (bool, error) {
				return e == errInvalidNicConfig, errInvalidNicConfig
			},
			outConfig: firecracker.Config{},
		},
		{
			name: "Invalid drives",
			opts: &options{
				FcNicConfig:        []string{"a/b"},
				FcAdditionalDrives: []string{"/no-suffix"},
			},
			expectedErr: func(e error) (bool, error) {
				return e == errInvalidDriveSpecificationNoSuffix, errInvalidDriveSpecificationNoSuffix
			},
			outConfig: firecracker.Config{},
		},
		{
			name: "Invalid vsock addr",
			opts: &options{
				FcNicConfig:        []string{"a/b"},
				FcAdditionalDrives: []string{tempFile.Name() + ":ro"},
				FcVsockDevices:     []string{"noCID"},
			},
			expectedErr: func(e error) (bool, error) {
				return e == errUnableToParseVsockDevices, errUnableToParseVsockDevices
			},
			outConfig: firecracker.Config{},
		},
		{
			name: "Invalid fifo config",
			opts: &options{
				FcNicConfig:        []string{"a/b"},
				FcAdditionalDrives: []string{tempFile.Name() + ":ro"},
				FcVsockDevices:     []string{"a:3"},
				FcFifoLogFile:      tempFile.Name(),
				createFifoFileLogs: func(_ string) (*os.File, error) {
					return nil, errUnableToCreateFifoLogFile
				},
			},
			expectedErr: func(e error) (bool, error) {
				return e != nil && strings.HasPrefix(e.Error(), errUnableToCreateFifoLogFile.Error()),
					errUnableToCreateFifoLogFile
			},
			outConfig: firecracker.Config{},
		},
		{
			name: "socket path provided",
			opts: &options{
				FcSocketPath: "/some/path/here",
			},
			expectedErr: func(e error) (bool, error) {
				return e == nil, nil
			},
			outConfig: firecracker.Config{
				SocketPath: "/some/path/here",
				Drives: []models.Drive{
					models.Drive{
						DriveID:      firecracker.String("1"),
						PathOnHost:   firecracker.String(""),
						IsRootDevice: firecracker.Bool(true),
						IsReadOnly:   firecracker.Bool(false),
					},
				},
				MachineCfg: models.MachineConfiguration{
					VcpuCount:  firecracker.Int64(0),
					MemSizeMib: firecracker.Int64(0),
					HtEnabled:  firecracker.Bool(true),
				},
			},
		},
		{
			name: "Valid config",
			opts: &options{
				FcSocketPath: "valid/path",
			},
			expectedErr: func(e error) (bool, error) {
				return e == nil, nil
			},
			outConfig: firecracker.Config{
				SocketPath: "valid/path",
				Drives: []models.Drive{
					models.Drive{
						DriveID:      firecracker.String("1"),
						PathOnHost:   firecracker.String(""),
						IsRootDevice: firecracker.Bool(true),
						IsReadOnly:   firecracker.Bool(false),
					},
				},
				MachineCfg: models.MachineConfiguration{
					VcpuCount:  firecracker.Int64(0),
					MemSizeMib: firecracker.Int64(0),
					HtEnabled:  firecracker.Bool(true),
				},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg, err := c.opts.getFirecrackerConfig()
			if ok, expected := c.expectedErr(err); !ok {
				t.Errorf("expected %s but got %s", expected, err)
			}
			if !reflect.DeepEqual(c.outConfig, cfg) {
				t.Errorf("expected %+v but got %+v", c.outConfig, cfg)
			}
		})

	}
}

func TestParseBlockDevices(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "firectl-test-drive-path")
	if err != nil {
		t.Error(err)
	}
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()
	validDrive := models.Drive{
		DriveID:      firecracker.String("2"),
		PathOnHost:   firecracker.String(tempFile.Name()),
		IsReadOnly:   firecracker.Bool(false),
		IsRootDevice: firecracker.Bool(false),
	}
	cases := []struct {
		name        string
		in          []string
		outDrives   []models.Drive
		expectedErr func(error) bool
	}{
		{
			name:      "No drive suffix",
			in:        []string{"/path"},
			outDrives: nil,
			expectedErr: func(a error) bool {
				return a == errInvalidDriveSpecificationNoSuffix
			},
		},
		{
			name:      "No drive path",
			in:        []string{":rw"},
			outDrives: nil,
			expectedErr: func(a error) bool {
				return a == errInvalidDriveSpecificationNoPath
			},
		},
		{
			name:        "non-existant drive path",
			in:          []string{"/does/not/exist:ro"},
			outDrives:   nil,
			expectedErr: os.IsNotExist,
		},
		{
			name:      "valid drive path + suffix",
			in:        []string{tempFile.Name() + ":rw"},
			outDrives: []models.Drive{validDrive},
			expectedErr: func(a error) bool {
				return a == nil
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			drives, err := parseBlockDevices(c.in)
			if !reflect.DeepEqual(c.outDrives, drives) {
				t.Errorf("expected %v but got %v for %s",
					c.outDrives,
					drives,
					c.in)
			}
			if !c.expectedErr(err) {
				t.Errorf("did not get the expected err but received %s for %s",
					err,
					c.in)
			}
		})
	}
}

func TestParseNicConfig(t *testing.T) {
	cases := []struct {
		name      string
		in        string
		outDevice string
		outMac    string
		outError  error
	}{
		{
			name:      "valid nic config",
			in:        "a/b",
			outDevice: "a",
			outMac:    "b",
			outError:  nil,
		},
		{
			name:      "no macaddr",
			in:        "a/",
			outDevice: "",
			outMac:    "",
			outError:  errInvalidNicConfig,
		},
		{
			name:      "no separater",
			in:        "ab",
			outDevice: "",
			outMac:    "",
			outError:  errInvalidNicConfig,
		},
		{
			name:      "empty nic config",
			in:        "",
			outDevice: "",
			outMac:    "",
			outError:  errInvalidNicConfig,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			device, macaddr, err := parseNicConfig(c.in)
			if device != c.outDevice {
				t.Errorf("expected device %s but got %s for input %s",
					c.outDevice,
					device,
					c.in)
			}
			if macaddr != c.outMac {
				t.Errorf("expected macaddr %s but got %s for input %s",
					c.outMac,
					macaddr,
					c.in)
			}
			if err != c.outError {
				t.Errorf("expected error %s but got %s for input %s",
					c.outError,
					err,
					c.in)
			}
		})
	}
}

func TestParseVsocks(t *testing.T) {
	cases := []struct {
		name        string
		in          []string
		outDevices  []firecracker.VsockDevice
		expectedErr func(a error) bool
	}{
		{
			name: "valid input",
			in:   []string{"a:3"},
			outDevices: []firecracker.VsockDevice{
				firecracker.VsockDevice{
					Path: "a",
					CID:  uint32(3),
				},
			},
			expectedErr: func(a error) bool {
				return a == nil
			},
		},
		{
			name:       "no CID",
			in:         []string{"a3:"},
			outDevices: []firecracker.VsockDevice{},
			expectedErr: func(a error) bool {
				return a == errUnableToParseVsockDevices
			},
		},
		{
			name:       "empty vsock",
			in:         []string{""},
			outDevices: []firecracker.VsockDevice{},
			expectedErr: func(a error) bool {
				return a == errUnableToParseVsockDevices
			},
		},
		{
			name:       "non-number CID",
			in:         []string{"a:b"},
			outDevices: []firecracker.VsockDevice{},
			expectedErr: func(a error) bool {
				return a == errUnableToParseVsockCID
			},
		},
		{
			name:       "no separator",
			in:         []string{"ae"},
			outDevices: []firecracker.VsockDevice{},
			expectedErr: func(a error) bool {
				return a == errUnableToParseVsockDevices
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			devices, err := parseVsocks(c.in)
			if !reflect.DeepEqual(devices, c.outDevices) {
				t.Errorf("expected %v but got %v for %s",
					c.outDevices,
					devices,
					c.in)
			}
			if !c.expectedErr(err) {
				t.Errorf("did not expect err: %s", err)
			}
		})
	}
}

func TestHandleFifos(t *testing.T) {
	validateTrue := func(options) bool { return true }
	cases := []struct {
		name         string
		opt          options
		outWriterNil bool
		expectedErr  func(error) (bool, error)
		numClosers   int
		validate     func(options) bool
	}{
		{
			name: "both FcFifoLogFile and FcLogFifo set",
			opt: options{
				FcFifoLogFile: "a",
				FcLogFifo:     "b",
			},
			outWriterNil: true,
			expectedErr: func(e error) (bool, error) {
				return e == errConflictingLogOpts, errConflictingLogOpts
			},
			numClosers: 0,
			validate:   validateTrue,
		},
		{
			name: "set FcFifoLogFile causing createFifoFileLogs to fail",
			opt: options{
				FcFifoLogFile: "fail-here",
				createFifoFileLogs: func(_ string) (*os.File, error) {
					return nil, errUnableToCreateFifoLogFile
				},
			},
			outWriterNil: true,
			expectedErr: func(a error) (bool, error) {
				if a == nil {
					return false,
						errUnableToCreateFifoLogFile
				}
				return strings.HasPrefix(a.Error(),
						errUnableToCreateFifoLogFile.Error()),
					errUnableToCreateFifoLogFile
			},
			numClosers: 0,
			validate:   validateTrue,
		},
		{
			name: "set FcLogFifo but not FcMetricsFifo",
			opt: options{
				FcLogFifo: "testing",
			},
			outWriterNil: true,
			expectedErr: func(e error) (bool, error) {
				return e == nil, nil
			},
			numClosers: 1,
			validate: func(opt options) bool {
				return strings.HasSuffix(opt.FcMetricsFifo, "fc_metrics_fifo")
			},
		},
		{
			name: "set FcMetricsFifo but not FcLogFifo",
			opt: options{
				FcMetricsFifo: "test",
			},
			outWriterNil: true,
			expectedErr: func(e error) (bool, error) {
				return e == nil, nil
			},
			numClosers: 1,
			validate: func(opt options) bool {
				return strings.HasSuffix(opt.FcLogFifo, "fc_fifo")
			},
		},
		{
			name: "set FcFifoLogFile with valid value",
			opt: options{
				FcFifoLogFile:      "value",
				createFifoFileLogs: createFifoFileLogs,
			},
			outWriterNil: false,
			expectedErr: func(e error) (bool, error) {
				return e == nil, nil
			},
			numClosers: 2,
			validate: func(opt options) bool {
				// remove fcfifoLogFile that is created
				os.Remove(opt.FcFifoLogFile)
				return strings.HasSuffix(opt.FcLogFifo, "fc_fifo") &&
					strings.HasSuffix(opt.FcMetricsFifo, "fc_metrics_fifo")
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w, e := c.opt.handleFifos()
			if (w == nil && !c.outWriterNil) || (w != nil && c.outWriterNil) {
				t.Errorf("expected writer to be %v but writer was %v",
					c.outWriterNil,
					w == nil)
			}
			if ok, expected := c.expectedErr(e); !ok {
				t.Errorf("expected %s but got %s", expected, e)
			}
			if len(c.opt.closers) != c.numClosers {
				t.Errorf("expected to have %d closers but had %d",
					c.numClosers,
					len(c.opt.closers))
			}
			if !c.validate(c.opt) {
				t.Errorf("options did not validate")
			}
			c.opt.Close()
		})
	}
}

func TestGetFirecrackerNetworkingConfig(t *testing.T) {
	cases := []struct {
		name        string
		opt         options
		expectedErr func(error) (bool, error)
		expectedNic []firecracker.NetworkInterface
	}{
		{
			name: "empty FCNicConfig",
			opt:  options{},
			expectedErr: func(e error) (bool, error) {
				return e == nil, nil
			},
			expectedNic: nil,
		},
		{
			name: "non-empty but invalid FcNicConfig",
			opt: options{
				FcNicConfig: []string{"invalid"},
			},
			expectedErr: func(e error) (bool, error) {
				return e == errInvalidNicConfig, errInvalidNicConfig
			},
			expectedNic: nil,
		},
		{
			name: "valid FcNicConfig with MMDS set to true",
			opt: options{
				FcNicConfig:   []string{"valid/things"},
				validMetadata: 42,
			},
			expectedErr: func(e error) (bool, error) {
				return e == nil, nil
			},
			expectedNic: []firecracker.NetworkInterface{
				firecracker.NetworkInterface{
					StaticConfiguration: &firecracker.StaticNetworkConfiguration{
						MacAddress:  "things",
						HostDevName: "valid",
					},
					AllowMMDS: true,
				},
			},
		},
		{
			name: "valid FcNicConfig with MMDS set to false",
			opt: options{
				FcNicConfig: []string{"valid/things"},
			},
			expectedErr: func(e error) (bool, error) {
				return e == nil, nil
			},
			expectedNic: []firecracker.NetworkInterface{
				firecracker.NetworkInterface{
					StaticConfiguration: &firecracker.StaticNetworkConfiguration{
						MacAddress:  "things",
						HostDevName: "valid",
					},
					AllowMMDS: false,
				},
			},
		},
		{
			name: "Multiple valid FcNicConfig with MMDS set to false",
			opt: options{
				FcNicConfig: []string{"valid/things", "morevalid/morethings"},
			},
			expectedErr: func(e error) (bool, error) {
				return e == nil, nil
			},
			expectedNic: []firecracker.NetworkInterface{
				firecracker.NetworkInterface{
					StaticConfiguration: &firecracker.StaticNetworkConfiguration{
						MacAddress:  "things",
						HostDevName: "valid",
					},
					AllowMMDS: false,
				},
				firecracker.NetworkInterface{
					StaticConfiguration: &firecracker.StaticNetworkConfiguration{
						MacAddress:  "morethings",
						HostDevName: "morevalid",
					},
					AllowMMDS: false,
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			nic, err := c.opt.getNetwork()
			if ok, expected := c.expectedErr(err); !ok {
				t.Errorf("expected %s but got %s", expected, err)
			}
			if !reflect.DeepEqual(nic, c.expectedNic) {
				t.Errorf("expected %v but got %v", c.expectedNic, nic)
			}
		})
	}
}

func TestGetBlockDevices(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "firectl-test-drive-path")
	if err != nil {
		t.Error(err)
	}
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()
	cases := []struct {
		name           string
		opt            options
		expectedErr    func(e error) (bool, error)
		expectedDrives []models.Drive
	}{
		{
			name: "invalid FcAdditionalDrives value",
			opt: options{
				FcAdditionalDrives: []string{"ab"},
			},
			expectedErr: func(e error) (bool, error) {
				return e == errInvalidDriveSpecificationNoSuffix,
					errInvalidDriveSpecificationNoSuffix
			},
			expectedDrives: nil,
		},
		{
			name: "valid FcAdditionalDrives with valid Root drive",
			opt: options{
				FcAdditionalDrives: []string{tempFile.Name() + ":ro"},
				FcRootDrivePath:    tempFile.Name(),
				FcRootPartUUID:     "UUID",
			},
			expectedErr: func(e error) (bool, error) {
				return e == nil, nil
			},
			expectedDrives: []models.Drive{
				models.Drive{
					DriveID:      firecracker.String("2"),
					PathOnHost:   firecracker.String(tempFile.Name()),
					IsReadOnly:   firecracker.Bool(true),
					IsRootDevice: firecracker.Bool(false),
				},
				models.Drive{
					DriveID:      firecracker.String("1"),
					PathOnHost:   firecracker.String(tempFile.Name()),
					IsRootDevice: firecracker.Bool(true),
					IsReadOnly:   firecracker.Bool(false),
					Partuuid:     "UUID",
				},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			drives, err := c.opt.getBlockDevices()
			if ok, expected := c.expectedErr(err); !ok {
				t.Errorf("expected %s but got %s", expected, err)
			}
			if !reflect.DeepEqual(drives, c.expectedDrives) {
				t.Errorf("expected %v but got %v", c.expectedDrives, drives)
			}
		})
	}
}

func TestGetSocketPath(t *testing.T) {
	cases := []struct {
		name        string
		setup       func() func()
		expectedOut func(string) bool
	}{
		{
			name:  "verify sane path",
			setup: func() func() { return func() {} },
			expectedOut: func(o string) bool {
				return strings.Contains(o, "firecracker.sock")
			},
		},
		{
			name: "no home dir",
			setup: func() func() {
				existing := os.Getenv("HOME")
				os.Setenv("HOME", "")
				return func() { os.Setenv("HOME", existing) }
			},
			expectedOut: func(o string) bool {
				return strings.Contains(o, os.TempDir())
			},
		},
		{
			name: "no tempdir",
			setup: func() func() {
				existingHome := os.Getenv("HOME")
				os.Setenv("HOME", "")
				existingTmp := os.Getenv("TMPDIR")
				os.Setenv("TMPDIR", "")
				return func() {
					os.Setenv("HOME", existingHome)
					os.Setenv("TMPDIR", existingTmp)
				}
			},
			expectedOut: func(o string) bool {
				return strings.Contains(o, "/tmp")
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer c.setup()()
			out := getSocketPath()
			if !c.expectedOut(out) {
				t.Errorf("getSocketPath returned %v", out)
			}
		})
	}
}
