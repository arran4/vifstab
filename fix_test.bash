#!/bin/bash
sed -i '/\/dev\/fuse \/mnt\/fuseblk/d' internal/rules/builtin/layered/fuse/testdata/fuse.txtar
