firectl
===

Firectl is a basic command-line tool that lets you run arbitrary
Firecracker MicroVMs via the command line. This lets you run a fully
functional Firecracker MicroVM, including console access, read/write
access to filesystems, and network connectivity.

Building
---

We use [go modules](https://github.com/golang/go/wiki/Modules), so you need to build with Go 1.11 or newer. `go build` or `make` should be sufficient to generate a working firectl binary.

Usage
---

You'll need to have a [firecracker](https://github.com/firecracker-microvm/firecracker) build, as well as an uncompressed Linux kernel image (`vmlinux`) and root filesystem image.

By default, firectl searches `PATH` for the firecracker binary. The location of the kernel and filesystem image must be provided explicitly.

```
Usage:
  firectl

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
  -d, --debug                   Enable debug output
  -h, --help                    Show usage
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

- Create an `i3.metal` instance using Amazon Linux 2
- Build latest version of firectl:

  ```
  sudo yum install -y git
  git clone https://github.com/firecracker-microvm/firectl
  sudo amazon-linux-extras install -y golang1.11
  cd firectl
  make
  ```

- Get Firecracker binary:

  ```
  curl -LOJ https://github.com/firecracker-microvm/firecracker/releases/download/v0.12.0/firecracker-v0.12.0
  chmod +x firecracker-v0.12.0
  sudo mv firecracker-v0.12.0 /usr/local/bin/firecracker
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
  firectl \
    --kernel=hello-vmlinux.bin \
    --root-drive=hello-rootfs.ext4 \
    --firecracker-log=./firecracker-vmm.log \
    --kernel-opts="console=ttyS0 noapic reboot=k panic=1 pci=off nomodules rw"
  ```
