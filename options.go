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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	models "github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func newOptions() *options {
	return &options{
		createFifoFileLogs: createFifoFileLogs,
	}
}

type options struct {
	FcBinary           string   `long:"firecracker-binary" description:"Path to firecracker binary"`
	FcKernelImage      string   `long:"kernel" description:"Path to the kernel image" default:"./vmlinux"`
	FcKernelCmdLine    string   `long:"kernel-opts" description:"Kernel commandline" default:"ro console=ttyS0 noapic reboot=k panic=1 pci=off nomodules"`
	FcRootDrivePath    string   `long:"root-drive" description:"Path to root disk image"`
	FcRootPartUUID     string   `long:"root-partition" description:"Root partition UUID"`
	FcAdditionalDrives []string `long:"add-drive" description:"Path to additional drive, suffixed with :ro or :rw, can be specified multiple times"`
	FcNicConfig        string   `long:"tap-device" description:"NIC info, specified as DEVICE/MAC"`
	FcVsockDevices     []string `long:"vsock-device" description:"Vsock interface, specified as PATH:CID. Multiple OK"`
	FcLogFifo          string   `long:"vmm-log-fifo" description:"FIFO for firecracker logs"`
	FcLogLevel         string   `long:"log-level" description:"vmm log level" default:"Debug"`
	FcMetricsFifo      string   `long:"metrics-fifo" description:"FIFO for firecracker metrics"`
	FcDisableHt        bool     `long:"disable-hyperthreading" short:"t" description:"Disable CPU Hyperthreading"`
	FcCPUCount         int64    `long:"ncpus" short:"c" description:"Number of CPUs" default:"1"`
	FcCPUTemplate      string   `long:"cpu-template" description:"Firecracker CPU Template (C3 or T2)"`
	FcMemSz            int64    `long:"memory" short:"m" description:"VM memory, in MiB" default:"512"`
	FcMetadata         string   `long:"metadata" description:"Firecracker Metadata for MMDS (json)"`
	FcFifoLogFile      string   `long:"firecracker-log" short:"l" description:"pipes the fifo contents to the specified file"`
	FcSocketPath       string   `long:"socket-path" short:"s" description:"path to use for firecracker socket, defaults to a unique file in in the first existing directory from {$HOME, $TMPDIR, or /tmp}"`
	Debug              bool     `long:"debug" short:"d" description:"Enable debug output"`

	closers       []func() error
	validMetadata interface{}

	createFifoFileLogs func(fifoPath string) (*os.File, error)
}

// Converts options to a usable firecracker config
func (opts *options) getFirecrackerConfig() (firecracker.Config, error) {
	// validate metadata json
	if opts.FcMetadata != "" {
		if err := json.Unmarshal([]byte(opts.FcMetadata), &opts.validMetadata); err != nil {
			return firecracker.Config{},
				errors.Wrap(err, errInvalidMetadata.Error())
		}
	}
	//setup NICs
	NICs, err := opts.getNetwork()
	if err != nil {
		return firecracker.Config{}, err
	}
	// BlockDevices
	blockDevices, err := opts.getBlockDevices()
	if err != nil {
		return firecracker.Config{}, err
	}

	// vsocks
	vsocks, err := parseVsocks(opts.FcVsockDevices)
	if err != nil {
		return firecracker.Config{}, err
	}

	//fifos
	fifo, err := opts.handleFifos()
	if err != nil {
		return firecracker.Config{}, err
	}

	var socketPath string
	if opts.FcSocketPath != "" {
		socketPath = opts.FcSocketPath
	} else {
		socketPath = getSocketPath()
	}

	return firecracker.Config{
		SocketPath:        socketPath,
		LogFifo:           opts.FcLogFifo,
		LogLevel:          opts.FcLogLevel,
		MetricsFifo:       opts.FcMetricsFifo,
		FifoLogWriter:     fifo,
		KernelImagePath:   opts.FcKernelImage,
		KernelArgs:        opts.FcKernelCmdLine,
		Drives:            blockDevices,
		NetworkInterfaces: NICs,
		VsockDevices:      vsocks,
		MachineCfg: models.MachineConfiguration{
			VcpuCount:   opts.FcCPUCount,
			CPUTemplate: models.CPUTemplate(opts.FcCPUTemplate),
			HtEnabled:   !opts.FcDisableHt,
			MemSizeMib:  opts.FcMemSz,
		},
		Debug: opts.Debug,
	}, nil
}

func (opts *options) getNetwork() ([]firecracker.NetworkInterface, error) {
	var NICs []firecracker.NetworkInterface
	if len(opts.FcNicConfig) > 0 {
		tapDev, tapMacAddr, err := parseNicConfig(opts.FcNicConfig)
		if err != nil {
			return nil, err
		}
		allowMDDS := opts.validMetadata != nil
		NICs = []firecracker.NetworkInterface{
			firecracker.NetworkInterface{
				MacAddress:  tapMacAddr,
				HostDevName: tapDev,
				AllowMDDS:   allowMDDS,
			},
		}
	}
	return NICs, nil
}

// constructs a list of drives from the options config
func (opts *options) getBlockDevices() ([]models.Drive, error) {
	blockDevices, err := parseBlockDevices(opts.FcAdditionalDrives)
	if err != nil {
		return nil, err
	}
	rootDrive := models.Drive{
		DriveID:      firecracker.String("1"),
		PathOnHost:   &opts.FcRootDrivePath,
		IsRootDevice: firecracker.Bool(true),
		IsReadOnly:   firecracker.Bool(false),
		Partuuid:     opts.FcRootPartUUID,
	}
	blockDevices = append(blockDevices, rootDrive)
	return blockDevices, nil
}

// handleFifos will see if any fifos need to be generated and if a fifo log
// file should be created.
func (opts *options) handleFifos() (io.Writer, error) {
	// these booleans are used to check whether or not the fifo queue or metrics
	// fifo queue needs to be generated. If any which need to be generated, then
	// we know we need to create a temporary directory. Otherwise, a temporary
	// directory does not need to be created.
	generateFifoFilename := false
	generateMetricFifoFilename := false
	var err error
	var fifo io.WriteCloser

	if len(opts.FcFifoLogFile) > 0 {
		if len(opts.FcLogFifo) > 0 {
			return nil, errConflictingLogOpts
		}
		generateFifoFilename = true
		// if a fifo log file was specified via the CLI then we need to check if
		// metric fifo was also specified. If not, we will then generate that fifo
		if len(opts.FcMetricsFifo) == 0 {
			generateMetricFifoFilename = true
		}
		if fifo, err = opts.createFifoFileLogs(opts.FcFifoLogFile); err != nil {
			return nil, errors.Wrap(err, errUnableToCreateFifoLogFile.Error())
		}
		opts.addCloser(func() error {
			return fifo.Close()
		})

	} else if len(opts.FcLogFifo) > 0 || len(opts.FcMetricsFifo) > 0 {
		// this checks to see if either one of the fifos was set. If at least one
		// has been set we check to see if any of the others were not set. If one
		// isn't set, we will generate the proper file path.
		if len(opts.FcLogFifo) == 0 {
			generateFifoFilename = true
		}

		if len(opts.FcMetricsFifo) == 0 {
			generateMetricFifoFilename = true
		}
	}

	if generateFifoFilename || generateMetricFifoFilename {
		dir, err := ioutil.TempDir(os.TempDir(), "fcfifo")
		if err != nil {
			return fifo, fmt.Errorf("Fail to create temporary directory: %v", err)
		}
		opts.addCloser(func() error {
			return os.RemoveAll(dir)
		})
		if generateFifoFilename {
			opts.FcLogFifo = filepath.Join(dir, "fc_fifo")
		}

		if generateMetricFifoFilename {
			opts.FcMetricsFifo = filepath.Join(dir, "fc_metrics_fifo")
		}
	}

	return fifo, nil
}

func (opts *options) addCloser(c func() error) {
	opts.closers = append(opts.closers, c)
}

func (opts *options) Close() {
	for _, closer := range opts.closers {
		err := closer()
		if err != nil {
			log.Error(err)
		}
	}
}

// given a []string in the form of path:suffix converts to []modesl.Drive
func parseBlockDevices(entries []string) ([]models.Drive, error) {
	devices := []models.Drive{}

	for i, entry := range entries {
		path := ""
		readOnly := true

		if strings.HasSuffix(entry, ":rw") {
			readOnly = false
			path = strings.TrimSuffix(entry, ":rw")
		} else if strings.HasSuffix(entry, ":ro") {
			path = strings.TrimSuffix(entry, ":ro")
		} else {
			return nil, errInvalidDriveSpecificationNoSuffix
		}

		if path == "" {
			return nil, errInvalidDriveSpecificationNoPath
		}

		if _, err := os.Stat(path); err != nil {
			return nil, err
		}

		e := models.Drive{
			// i + 2 represents the drive ID. We will reserve 1 for root.
			DriveID:      firecracker.String(strconv.Itoa(i + 2)),
			PathOnHost:   firecracker.String(path),
			IsReadOnly:   firecracker.Bool(readOnly),
			IsRootDevice: firecracker.Bool(false),
		}
		devices = append(devices, e)
	}
	return devices, nil
}

// Given a string of the form DEVICE/MACADDR, return the device name and the mac address, or an error
func parseNicConfig(cfg string) (string, string, error) {
	fields := strings.Split(cfg, "/")
	if len(fields) != 2 || len(fields[0]) == 0 || len(fields[1]) == 0 {
		return "", "", errInvalidNicConfig
	}
	return fields[0], fields[1], nil
}

// Given a list of string representations of vsock devices,
// return a corresponding slice of machine.VsockDevice objects
func parseVsocks(devices []string) ([]firecracker.VsockDevice, error) {
	var result []firecracker.VsockDevice
	for _, entry := range devices {
		fields := strings.Split(entry, ":")
		if len(fields) != 2 || len(fields[0]) == 0 || len(fields[1]) == 0 {
			return []firecracker.VsockDevice{}, errUnableToParseVsockDevices
		}
		CID, err := strconv.ParseUint(fields[1], 10, 32)
		if err != nil {
			return []firecracker.VsockDevice{}, errUnableToParseVsockCID
		}
		dev := firecracker.VsockDevice{
			Path: fields[0],
			CID:  uint32(CID),
		}
		result = append(result, dev)
	}
	return result, nil
}

func createFifoFileLogs(fifoPath string) (*os.File, error) {
	return os.OpenFile(fifoPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
}

// getSocketPath provides a randomized socket path by building a unique fielname
// and searching for the existance of directories {$HOME, os.TempDir()} and returning
// the path with the first directory joined with the unique filename. If we can't
// find a good path panics.
func getSocketPath() string {
	filename := strings.Join([]string{
		".firecracker.sock",
		strconv.Itoa(os.Getpid()),
		strconv.Itoa(rand.Intn(1000))},
		"-",
	)
	var dir string
	if d := os.Getenv("HOME"); checkExistsAndDir(d) {
		dir = d
	} else if checkExistsAndDir(os.TempDir()) {
		dir = os.TempDir()
	} else {
		panic("Unable to find a location for firecracker socket. 'It's not going to do any good to land on mars if we're stupid.' --Ray Bradbury")
	}

	return filepath.Join(dir, filename)
}

// checkExistsAndDir returns true if path exists and is a Dir
func checkExistsAndDir(path string) bool {
	// empty
	if path == "" {
		return false
	}
	// does it exist?
	if info, err := os.Stat(path); err == nil {
		// is it a directory?
		return info.IsDir()
	}
	return false
}
