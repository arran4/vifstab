# RCS history and recovery

Only the system /etc/fstab has an external history directory: the archive is <backup-dir>/fstab,v. Alternative files are not versioned by default. Explicit --rcs enables an adjacent file,v archive; its persistent advisory lock is file,v.lock. No alternative-file history or lock is created in the cache directory. Relative paths retain symlink-sensitive spelling. The source identity and revision integrity checks still protect against mixing unrelated histories. A pre-existing unlabelled or corrupt archive is refused without overwriting it; retain and inspect its contents before starting a new history.

## Atomic updates and concurrency

Directories are created privately (0700); history and lock files use 0600.
Use a trusted backup directory, not a location whose entries another user can
replace. Existing directory ownership/permissions are the caller's responsibility.
Path-based checks are not protection from hostile directory races.

A per-history advisory file lock covers reading, validation, revision creation,
atomic replacement and durability sync. It coordinates cooperating vifstab
writers, including separate processes. It does not constrain unrelated writers.
The lock is released on close/process exit; the lock file remains in place.
**Do not unlink a lock file to "clear" it**: that can create independent locks on
different inodes. There is no PID-based stale-lock guessing.

New directory entries are synced from the first existing ancestor inward. The
archive is prepared in a private temporary file, chmodded, synced, closed,
atomically renamed and its directory synced. Failures are reported and incomplete
temporary archives removed. A rename, directory-sync or lock-close failure can
mean the new archive is already present; an error does not guarantee rollback.
Filesystems that cannot provide the required operations report errors. Locking
is currently implemented on Linux; other platforms fail explicitly.

## Recovery contract

The first revision contains the content passed to the save operation. This does
not automatically back up an unversioned destination's original pre-edit bytes.
Retain an independently reviewed copy before the first important edit; do not
infer that every earlier functional state is present in the history.

New archives link trunk revisions and store backward replacement deltas. The
internal `rcs.Restore(source, revision, config, fs)` API reads and verifies the
complete chain and returns the selected bytes. It writes no destination file.
An empty revision selects the head. It preserves empty text, CRLF, absent final
newlines, at-sign quoting and arbitrary byte content. Invalid identities, broken
links, malformed deltas, missing revisions or digest mismatches return errors.

The existing Go RCS library handles the archive format and edit application.
Vifstab records exact ending information because the library checkout inherits
the newest revision's ending; its default delta scanner also strips CRLF payload
CR bytes. The recovery path parses the bounded replacement-script form emitted
by this writer and applies the recorded ending at each revision. It does not
accept arbitrary externally modified delta forms as if they were produced here.

There is no restore CLI command or automatic overwrite of the live fstab.
Native external RCS command interoperability has not been tested; the existing
library's checkout is checked for representative LF histories, and the internal
recovery API is tested byte-for-byte across ending changes. Do not edit generated
archives with an external check-in tool without preserving their provenance
contract. Inspect recovered data in a separate file and revalidate it before
any intentional replacement of an important configuration.

## Metadata-first archive and lock opens

History reads retain their regular-file, final-symlink refusal and identity checks,
but now use the shared metadata-first opener before obtaining a data descriptor.
A changed path cannot redirect that data open to a device. Existing lock files
use the same strict guard and are reopened read/write through the checked inode;
new locks use exclusive creation. Persistent names, 0600 permissions, close-on-exec,
read/write access, advisory lock scope and no-unlink behavior are preserved.
Linux metadata/proc-fd access is required; unavailable access is an error, with no
path-based fallback or privilege acquisition. The caller still supplies a trusted
backup directory; this does not claim race-free behavior against hostile directory
writers. Shared opener tests cover refusal before any data-open request. Native
fixtures check independent lock contention, descriptors and unchanged inode/data.
