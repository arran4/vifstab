# RCS history identity and recovery

New histories live under `<backup-dir>/v1/<name>--<digest>,v`. The display name
contains at most 40 ASCII letters, digits, hyphens or underscores from the source
basename. The digest is the full SHA-256 of the configured absolute source path.
Source components never become backup subdirectories, so arbitrary source names
cannot escape the selected backup directory or create an uncontrolled hierarchy.

Relative sources are prefixed with the caller's working directory. The mapping
does not collapse `.`/`..` or filesystem aliases: cleaning `link/../fstab` before
resolving the link can identify a different file. Use consistent path spelling;
aliases may have separate histories rather than risk merging unrelated sources.
The selected backup directory also retains symlink-sensitive path semantics.
This is a pathname identity, not an inode identity; ordinary replacement at the
same configured path continues its history.

Each archive's RCS description records a versioned source identity (base64 of the
path bytes) and revision sizes, ending information and SHA-256 digests. The writer
checks that identity before appending. A mismatched label, including a hypothetical
hash-name collision, refuses the update instead of combining histories. Base64
is not anonymization: archives contain private file contents and identifying data.
Digests detect corruption; they are not signatures against an attacker who can
rewrite both the contents and metadata.

## Existing basename-only histories

The previous layout used `<backup-dir>/<basename>,v`, which could already contain
revisions from different sources. When a source has no labelled new history but
a legacy basename file exists, updating returns an error naming both paths.
The program does not rename, overwrite, import or guess the ownership of that file.

Before starting a new history:

1. Retain a private copy of the old archive and inspect it. Establish which source
   each recoverable revision belongs to; do not assume every entry belongs to the
   current fstab. Older vifstab archives may contain unlinked full-text revisions,
   so ordinary RCS checkout is not proof that every old revision is recoverable.
2. As an explicit operator action, archive the legacy file under a different
   private name/location so it no longer occupies the ambiguous basename path.
   Keep its bytes and provenance; the application does not perform this action.
3. Retry the reviewed save. It starts a separate labelled history at revision
   `1.1`. This is not an automatic migration of old revision numbers or contents.

An already labelled history can continue even if a legacy archive is retained
alongside it; the source label is authoritative. Foreign or corrupt labelled
histories are refused without rewriting their bytes. A versioning failure can
occur after the fstab itself was saved: report that partial result and inspect
both files rather than assuming the destination was rolled back.

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
