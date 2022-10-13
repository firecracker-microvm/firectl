# 0.2.0

* Upgraded to v1.0.0 of the firecracker-go-sdk
* Added initrd option (#84)
* Added read-only option for rootfs (#74)
* Added jailer support (#57)
* Added option to configure multiple NICs (#44)

*Note* - Although still compatible, this release does not provide full feature parity with firecracker v1.0.0. Majors features that are not included in this release are:

* Creating and loading snapshots
* Setting drive IO engine type
* Pause/Resume VM
* MMDS version 2

# 0.1.1 (unreleased)

* Add an `install` target to Makefile

# 0.1.0

* Initial release
