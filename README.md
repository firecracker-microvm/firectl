firectl
===

Firectl is a basic command-line tool that lets you run arbitrary
Firecracker MicroVMs via the command line. This lets you run a fully
functional Firecracker MicroVM, including console access, read/write
access to filesystems, and network connectivity.

Usage
---

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
      --metadata=               Firecracker Meatadata for MMDS (json)
  -l, --firecracker-log=        pipes the fifo contents to the specified file
  -d, --debug                   Enable debug output
  -h, --help                    Show usage
```

Example
---

```
./cmd/firectl/firectl \
  --kernel=~/bin/vmlinux \
  --root-drive=/images/image-debootstrap.img -t \
  --cpu-template=T2 \
  --firecracker-log=~/firecracker-vmm.log \
  --kernel-opts="console=ttyS0 noapic reboot=k panic=1 pci=off nomodules rw" \
  --vsock-device=root:3 \
  --metadata='{"foo":"bar"}'
```
