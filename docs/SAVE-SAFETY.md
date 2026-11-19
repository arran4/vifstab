# Save behaviour and limitations

Saving an existing regular file uses a temporary file in the effective target's
directory, followed by file sync, atomic rename and directory sync. File sync,
rename and directory-sync errors are reported; an error after rename can mean
the new content is already installed. Backup/versioning is a separate later
stage and can also fail after the destination has changed.

For a configured symlink, the save resolves the link and replaces its regular
target. It preserves the link itself, including relative links, chains and
links in parent directories. Resolution happens before interpreting `..`.
Broken links, loops, inaccessible metadata, missing parent directories and
non-regular targets are rejected. An absent ordinary file in an existing
directory may be created, including through a linked parent directory. A
detected change to the resolved path during replacement preparation is refused.

Before opening the editor, the workflow snapshots the resolved path, file
identity, mode, size, modification time and a SHA-256 content digest. It verifies
the snapshot around the initial read and again immediately before rename.
Detected content changes, inode replacements, creation/deletion or retargeting
refuse the save without updating RCS or running the post-save systemd action.
Unavailable observations also refuse the save; they are not reported as proof
of a conflict. An unchanged new-file destination may still be created.

After a refused edit save, the diagnostic names a new mode-0600 recovery file
containing the edited content. Reconcile that copy with the current destination
and retry; delete the copy when finished because it may contain private data.
If creating that copy fails, the original editor temporary file is retained and
named in the error instead. Recovery files are not automatically imported or
restored, and have no crash-durability guarantee. In-place formatting does not
yet capture a pre-format snapshot and therefore has no session conflict check.

These path-based checks are optimistic. A writer can still race the final
comparison and rename; they are not a filesystem transaction or protection from
hostile directory changes. Do not edit files in directories controlled by
another user. The existing save copies permission bits only; preserving
ownership, ACLs, xattrs, security labels
and special mode bits remains #155. RCS path identity and concurrent history
updates remain #156. These limitations apply to editing and in-place formatting;
do not treat this save path as production-safe until they are addressed.
