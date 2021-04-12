firectl
===

[![Build status](https://badge.buildkite.com/92fe02b4bd9564be0f7ea21d1ee782f6a6fe55cbd5465e3480.svg?branch=master)](https://buildkite.com/firecracker-microvm/firectl)

Firectl is a basic command-line tool that lets you run arbitrary
Firecracker MicroVMs via the command line. This lets you run a fully
functional Firecracker MicroVM, including console access, read/write
access to filesystems, and network connectivity.

Building
---

The default Makefile rule executes `go build` and relies on the Go toolchain
installed on your computer.
_We use [go modules](https://github.com/golang/go/wiki/Modules), so you need to
build with Go 1.11 or newer._

If you do not have a new-enough Go toolchain installed, you can use `make
build-in-docker`.  This rule creates a temporary Docker container which builds
and copies the binary to your current directory.

Usage
---

You'll need to have a
[firecracker](https://github.com/firecracker-microvm/firecracker) build, as well
as an uncompressed Linux kernel image (`vmlinux`) and root filesystem image.

By default, firectl searches `PATH` for the firecracker binary. The location of
the kernel and filesystem image must be provided explicitly.

```
Usage:
  firectl [OPTIONS]

Application Options:
      --firecracker-binary=     Path to firecracker binary
      --kernel=                 Path to the kernel image (default: ./vmlinux)
      --kernel-opts=            Kernel commandline (default: ro console=ttyS0 noapic reboot=k panic=1 pci=off nomodules)
      --root-drive=             Path to root disk image
      --root-partition=         Root partition UUID
      --add-drive=              Path to additional drive, suffixed with :ro or :rw, can be specified multiple times
      --tap-device=             NIC info, specified as DEVICE/MAC
      --vsock-device=           Vsock interface, specified as PATH:CID. Multiple OK
      --vmm-log-fifo=           FIFO for firecracker logs
      --log-level=              vmm log level (default: Debug)
      --metrics-fifo=           FIFO for firecracker metrics
  -t, --disable-hyperthreading  Disable CPU Hyperthreading
  -c, --ncpus=                  Number of CPUs (default: 1)
      --cpu-template=           Firecracker CPU Template (C3 or T2)
  -m, --memory=                 VM memory, in MiB (default: 512)
      --metadata=               Firecracker Metadata for MMDS (json)
  -l, --firecracker-log=        pipes the fifo contents to the specified file
  -s, --socket-path=            path to use for firecracker socket, defaults to a unique file in in the first existing directory from {$HOME, $TMPDIR, or /tmp}
  -d, --debug                   Enable debug output

Help Options:
  -h, --help                    Show this help message
```

Example
---

```
firectl \
  --kernel=~/bin/vmlinux \
  --root-drive=/images/image-debootstrap.img -t \
  --cpu-template=T2 \
  --firecracker-log=~/firecracker-vmm.log \
  --kernel-opts="console=ttyS0 noapic reboot=k panic=1 pci=off nomodules rw" \
  --vsock-device=root:3 \
  --metadata='{"foo":"bar"}'
```

Getting Started on AWS
---

- Create an `m5d.metal` instance using Amazon Linux 2
- Get firectl binary:

  ```
  curl -Lo firectl https://firectl-release.s3.amazonaws.com/firectl-v0.1.0
  curl -Lo firectl.sha256 https://firectl-release.s3.amazonaws.com/firectl-v0.1.0.sha256
  sha256sum -c firectl.sha256
  chmod +x firectl
  ```

- Get Firecracker binary:

  ```
  curl -Lo firecracker https://github.com/firecracker-microvm/firecracker/releases/download/v0.16.0/firecracker-v0.16.0
  chmod +x firecracker
  sudo mv firecracker /usr/local/bin/firecracker
  ```

- Give read/write access to KVM:

  ```
  sudo setfacl -m u:${USER}:rw /dev/kvm
  ```

- Download kernel and root filesystem:

  ```
  curl -fsSL -o hello-vmlinux.bin https://s3.amazonaws.com/spec.ccfc.min/img/hello/kernel/hello-vmlinux.bin
  curl -fsSL -o hello-rootfs.ext4 https://s3.amazonaws.com/spec.ccfc.min/img/hello/fsfiles/hello-rootfs.ext4
  ```

- Create microVM:

  ```
  ./firectl \
    --kernel=hello-vmlinux.bin \
    --root-drive=hello-rootfs.ext4
  ```

Testing
---
By default the tests require the firectl binary to be built and a kernel image
to be present. The integration tests look for the binary and kernel image in
the root directory. By default it will look for vmlinux kernel image. This can
be overwritten by setting the environment variable `KERNELIMAGE` to the desired
path. To disable these tests simply set the environment variable
`SKIP_INTEG_TEST=1`.

Questions?
---

Please use
[GitHub issues](https://github.com/firecracker-microvm/firectl/issues)
to report problems, discuss roadmap items, or make feature requests.

If you've discovered an issue that may have security implications to
users or developers of this software, please do not report it using
GitHub issues, but instead follow
[Firecracker's security reporting guidelines](https://github.com/firecracker-microvm/firecracker/blob/main/SECURITY.md).

Other discussion: For general discussion, please join us in the
`#general` channel on the [Firecracker Slack](https://tinyurl.com/firecracker-microvm).
