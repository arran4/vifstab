#!/bin/bash
sed -i 's/if fsType != "fuse" && fsType != "fuse3" && !strings.HasPrefix(fsType, "fuse.") {/if fsType != "fuse" \&\& fsType != "fuse3" \&\& fsType != "fuseblk" \&\& fsType != "glusterfs" \&\& fsType != "ntfs-3g" \&\& fsType != "mergerfs" \&\& fsType != "sshfs" \&\& fsType != "s3fs" \&\& !strings.HasPrefix(fsType, "fuse.") {/g' internal/rules/builtin/layered/fuse/fuse.go

cat << 'TEST' >> internal/rules/builtin/layered/fuse/testdata/fuse.txtar
/dev/fuse /mnt/fuseblk fuseblk default_permissions,allow_other 0 0
/dev/fuse /mnt/fuseblk ntfs-3g default_permissions,allow_other 0 0
/dev/fuse /mnt/fuseblk sshfs default_permissions,allow_other 0 0
/dev/fuse /mnt/fuseblk s3fs default_permissions,allow_other 0 0
/dev/fuse /mnt/fuseblk mergerfs default_permissions,allow_other 0 0
/dev/fuse /mnt/fuseblk glusterfs default_permissions,allow_other 0 0
TEST
