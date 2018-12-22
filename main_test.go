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
	"testing"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	models "github.com/firecracker-microvm/firecracker-go-sdk/client/models"
)

func TestParseBlockDevices(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "firectl-test-drive-path")
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()
	validDrive := models.Drive{
		DriveID:      firecracker.String("2"),
		PathOnHost:   firecracker.String(tempFile.Name()),
		IsReadOnly:   firecracker.Bool(false),
		IsRootDevice: firecracker.Bool(false),
	}
	cases := []struct {
		in          []string
		outDrives   []models.Drive
		expectedErr func(error) bool
	}{
		// no suffix
		{
			in:        []string{"/path"},
			outDrives: nil,
			expectedErr: func(a error) bool {
				return a == invalidDriveSpecificationNoSuffix
			},
		},
		// no path
		{
			in:        []string{":rw"},
			outDrives: nil,
			expectedErr: func(a error) bool {
				return a == invalidDriveSpecificationNoPath
			},
		},
		// path does not exist
		{
			in:          []string{"/does/not/exist:ro"},
			outDrives:   nil,
			expectedErr: os.IsNotExist,
		},
		// valid
		{
			in:        []string{tempFile.Name() + ":rw"},
			outDrives: []models.Drive{validDrive},
			expectedErr: func(a error) bool {
				return a == nil
			},
		},
	}
	for _, c := range cases {
		drives, err := parseBlockDevices(c.in)
		if !reflect.DeepEqual(c.outDrives, drives) {
			t.Errorf("expected %v but got %v for %s", c.outDrives, drives, c.in)
		}
		if !c.expectedErr(err) {
			t.Errorf("did not get the expected err but received %s for %s", err, c.in)
		}
	}
}

func TestParseNicConfig(t *testing.T) {
	cases := []struct {
		in        string
		outDevice string
		outMac    string
		outError  error
	}{
		// valid input
		{
			in:        "a/b",
			outDevice: "a",
			outMac:    "b",
			outError:  nil,
		},
		// missing macaddr but has slash
		{
			in:        "a/",
			outDevice: "",
			outMac:    "",
			outError:  parseNicConfigError,
		},
		// no /
		{
			in:        "ab",
			outDevice: "",
			outMac:    "",
			outError:  parseNicConfigError,
		},
		// empty input
		{
			in:        "",
			outDevice: "",
			outMac:    "",
			outError:  parseNicConfigError,
		},
	}

	for _, c := range cases {
		device, macaddr, err := parseNicConfig(c.in)
		if device != c.outDevice {
			t.Errorf("expected device %s but got %s for input %s", c.outDevice, device, c.in)
		}
		if macaddr != c.outMac {
			t.Errorf("expected macaddr %s but got %s for input %s", c.outMac, macaddr, c.in)
		}
		if err != c.outError {
			t.Errorf("expected error %s but got %s for input %s", c.outError, err, c.in)
		}
	}
}

func TestParseVsocks(t *testing.T) {
	cases := []struct {
		in          []string
		outDevices  []firecracker.VsockDevice
		expectedErr func(a error) bool
	}{
		// valid input
		{
			in: []string{"a:3"},
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
		// not two fields
		{
			in:         []string{"a3:"},
			outDevices: []firecracker.VsockDevice{},
			expectedErr: func(a error) bool {
				return a == unableToParseVsockDevices
			},
		},
		// empty string
		{
			in:         []string{""},
			outDevices: []firecracker.VsockDevice{},
			expectedErr: func(a error) bool {
				return a == unableToParseVsockDevices
			},
		},
		// non-number
		{
			in:         []string{"a:b"},
			outDevices: []firecracker.VsockDevice{},
			expectedErr: func(a error) bool {
				return a == unableToParseVsockCID
			},
		},
		// no :
		{
			in:         []string{"ae"},
			outDevices: []firecracker.VsockDevice{},
			expectedErr: func(a error) bool {
				return a == unableToParseVsockDevices
			},
		},
	}
	for _, c := range cases {
		devices, err := parseVsocks(c.in)
		if !reflect.DeepEqual(devices, c.outDevices) {
			t.Errorf("expected %v but got %v for %s", c.outDevices, devices, c.in)
		}
		if !c.expectedErr(err) {
			t.Errorf("did not expect err: %s", err)
		}
	}
}
