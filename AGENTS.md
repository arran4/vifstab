# Agent Constraints

* Do **not** execute external commands such as `lsblk`, `blkid`, `udevadm`, `fdisk`, etc.
* Do **not** open/read block devices directly.
* Live block-device metadata must come only from normal filesystem metadata/interfaces such as directory enumeration, symlinks, `stat`/`fstat`-style information, `/sys`, `/run/udev/data`, `/dev/disk/by-*`, and `/proc` where appropriate.
