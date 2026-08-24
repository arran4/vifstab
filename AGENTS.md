# Agent Constraints

* Do **not** execute external commands such as `lsblk`, `blkid`, `udevadm`, `fdisk`, etc.
* Do **not** open/read block devices directly in the normal/default mode.
* Live block-device metadata must come only from normal filesystem metadata/interfaces such as directory enumeration, symlinks, `stat`/`fstat`-style information, `/sys`, `/run/udev/data`, `/dev/disk/by-*`, and `/proc` where appropriate.
* A future mode may explicitly opt in to reading block devices directly to inspect partition tables/filesystem headers, but ONLY through an explicitly enabled future mode. Do not implement that mode here.
