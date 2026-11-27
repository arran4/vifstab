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

This path-based resolution and recheck is optimistic. It does not provide a
race-free guarantee against hostile directory changes, and does not yet prevent
a stale editor session from overwriting a concurrent content update (#153).
Do not edit files in directories controlled by another user. The existing save
copies permission bits only; preserving ownership, ACLs, xattrs, security labels
and special mode bits remains #155. RCS path identity and concurrent history
updates remain #156. These limitations apply to editing and in-place formatting;
do not treat this save path as production-safe until they are addressed.
