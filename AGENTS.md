# Agent Constraints

* Do **not** execute external commands such as `lsblk`, `blkid`, `udevadm`, `fdisk`, etc.
* Do **not** open/read block devices directly in the normal/default mode.
* Live block-device metadata must come only from normal filesystem metadata/interfaces such as directory enumeration, symlinks, `stat`/`fstat`-style information, `/sys`, `/run/udev/data`, `/dev/disk/by-*`, and `/proc` where appropriate.
* Verifying the contents of a filesystem on a block device is currently **out of scope**. Do not mount a device, require it to be mounted, or rely on the OS/VFS view of a mounted filesystem in order to validate its contents.
* A future explicitly enabled mode may inspect filesystem contents, but it should do so by reading the block device directly and parsing the filesystem format itself rather than mounting it or relying on the host OS to interpret the filesystem.
* A future mode may also explicitly opt in to reading block devices directly to inspect partition tables/filesystem headers, but ONLY through an explicitly enabled future mode. Do not implement that mode here.
