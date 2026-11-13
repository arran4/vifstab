# Configuration text inputs

Fstab text inputs for verification, formatting and editing, and RCS archive text
reads, are classified before opening a data descriptor. Block-device inputs are
refused; unknown or inaccessible classification is an operational error. This
does not add device discovery or inspect any block-device contents.

Linux uses an `O_PATH` metadata reference, checks its type, and obtains read access
through that same inode under `/proc/self/fd`. A subsequent identity/type check
must also succeed. Retargeting the original pathname cannot redirect the data
open. Metadata and data descriptors use close-on-exec; the data descriptor is
read-only and ordinary permissions use the caller's credentials. There is no
unsafe pathname fallback when these interfaces are unavailable. See
[open(2)](https://man7.org/linux/man-pages/man2/open.2.html).

Regular files and symlinked files remain supported. Intentional named-pipe and
character-stream inputs retain blocking stream semantics. Formatting still
accepts piped stdin; its borrowed descriptor is checked before reading. A
caller-supplied block descriptor is refused as input. Reader-based parsing APIs
remain available for explicitly supplied data and deterministic tests; their
callers own that input. The helper does not close borrowed standard input.

The pathname helper currently requires Linux. Other platforms or unavailable
metadata/proc-fd interfaces report an explicit limitation. Linux remains the
supported product target. This does not implement BSD support or infer privileges
from the process identity.

This boundary does not guarantee a snapshot of concurrently modified file
contents, solve #155's metadata preservation contract, or replace #153's save
conflict detection. Source copying uses the injected editor FS so virtual test
sources never cause a host fstab read. Existing RCS recovery/identity work remains
separate; preserve #188's stronger regular-file and symlink requirements during
integration.

This root change applies to the current main CLI. The generated #182 CLI and
#174 snapshot/capture branch need matching calls to the shared helper, including
their borrowed-stdin paths, before #194 is fully resolved. Regenerate command
bindings only if their annotated interfaces change; do not hand-edit generated
files. #193's image parser has a stricter regular-image policy and block diagrams
continue to use canonical metadata inventory/resolution.

Tests use injected block-type metadata, temporary regular files/symlinks/FIFOs
and anonymous pipes. No real block device or host fstab is opened. Six compiled
TXTAR scenarios check exact output/status, inherited read-only inputs and fixture
inode/content/mode/mtime preservation. This is fixture evidence, not a guarantee
for all kernels or unusual device drivers.
