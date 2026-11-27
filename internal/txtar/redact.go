package txtar

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

// One mapping handles inventory identities, fstab references and observation
// keys. Namespaces prevent e.g. a label and a path with equal text colliding.
type redactor struct{ values map[string]string }

func newRedactor() *redactor { return &redactor{values: map[string]string{}} }
func (r *redactor) token(kind, value string) string {
	if value == "" {
		return ""
	}
	key := kind + ":" + value
	if s, ok := r.values[key]; ok {
		return s
	}
	s := fmt.Sprintf("sanitized-%s-%d", kind, len(r.values)+1)
	if kind == "uuid" || kind == "partuuid" {
		s = fmt.Sprintf("00000000-0000-0000-0000-%012d", len(r.values)+1)
	}
	r.values[key] = s
	return s
}
func decodeEscapes(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) {
			start, base := i+1, 8
			if s[i+1] == 'x' {
				start, base = i+2, 16
			}
			end := i + 4
			if n, err := strconv.ParseUint(s[start:end], base, 8); err == nil {
				b.WriteByte(byte(n))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
func (r *redactor) identity(kind, value string) string {
	value = strings.Trim(value, `"'`)
	if kind == "uuid" || kind == "partuuid" {
		value = strings.ToLower(value)
	}
	if kind == "label" || kind == "partlabel" {
		value = decodeEscapes(value)
	}
	return r.token(kind, value)
}
func (r *redactor) reference(ref string) string {
	resolved := rules.ResolveDevicePath(ref)
	if resolved != ref {
		key, _, _ := strings.Cut(ref, "=")
		return strings.ToUpper(strings.TrimSpace(key)) + "=" + r.alias(resolved)
	}
	if strings.HasPrefix(ref, "/dev/disk/by-") {
		dir, _, ok := strings.Cut(strings.TrimPrefix(ref, "/dev/disk/by-"), "/")
		if ok {
			return "/dev/disk/by-" + dir + "/" + r.alias(ref)
		}
	}
	if strings.HasPrefix(ref, "/dev/mapper/") {
		return "/dev/mapper/" + r.token("device", ref)
	}
	if strings.HasPrefix(ref, "/dev/") {
		return "/dev/" + r.token("device", ref)
	}
	if strings.HasPrefix(ref, "//") {
		return "//" + r.token("network", ref) + "/share"
	}
	if strings.Contains(ref, ":/") {
		return r.token("network", ref) + ":/export"
	}
	if strings.HasPrefix(ref, "/") {
		return r.path(ref)
	}
	switch ref {
	case "", "none", "tmpfs", "proc", "sysfs", "devtmpfs", "devpts", "ramfs", "overlay", "swap", "cgroup", "cgroup2", "autofs", "securityfs", "debugfs", "tracefs", "mqueue", "hugetlbfs":
		return ref
	}
	return r.token("source", ref)
}
func (r *redactor) alias(ref string) string {
	tail := strings.TrimPrefix(ref, "/dev/disk/by-")
	kind, value, ok := strings.Cut(tail, "/")
	if !ok {
		return r.token("alias", ref)
	}
	return r.identity(kind, value)
}
func (r *redactor) path(p string) string {
	// Public role paths affect partition-role validation and contain no personal data.
	switch p {
	case "", "/", "/boot", "/efi", "/boot/efi", "none":
		return p
	}
	if strings.HasPrefix(p, "/dev/") {
		return r.reference(p)
	}
	if strings.HasPrefix(p, "/") {
		return "/" + r.token("path", decodeEscapes(p))
	}
	return r.token("path", decodeEscapes(p))
}

var versionPattern = regexp.MustCompile(`(?:Linux version )?(\d+\.\d+(?:\.\d+)?)`)

func (r *redactor) kernel(s string) string {
	match := versionPattern.FindStringSubmatch(s)
	if len(match) == 2 {
		return "Linux version " + match[1]
	}
	return ""
}

// Numeric values and a small set of non-identifying option enums retain their
// syntax. Every other value is pseudonymized, including unknown option values.
var publicValueOptions = map[string]bool{
	"vers": true, "nfsvers": true, "minorversion": true, "rsize": true, "wsize": true,
	"bsize": true, "timeo": true, "retrans": true, "actimeo": true, "port": true,
	"mode": true, "umask": true, "fmask": true, "dmask": true, "dmode": true, "fmode": true,
	"compress": true, "compress-force": true, "discard": true, "cache": true, "data": true,
	"commit": true, "subvolid": true, "max_inline": true, "thread_pool": true,
	"metadata_ratio": true, "barrier": true, "size": true, "nr_inodes": true,
	"index": true, "metacopy": true, "redirect_dir": true, "uuid": true, "xino": true,
	"nfs_export": true, "verity": true, "fsync": true, "iocharset": true,
}
var safeOptionValue = regexp.MustCompile(`^(?:[0-9]+(?:\.[0-9]+)*|[0-9]+[kKmMgGtT]?|async|sync|strict|loose|none|always|never|auto|ordered|writeback|journal|on|off|yes|no|zstd(?::[0-9]+)?|lzo|zlib(?::[0-9]+)?|gzip|lz4)$`)

func (r *redactor) options(s string) string {
	opts := rules.SplitOptions(s)
	for i, opt := range opts {
		key, val, has := strings.Cut(opt, "=")
		if !has {
			continue
		}
		switch strings.ToLower(key) {
		case "uid", "gid":
			if n, err := strconv.ParseInt(val, 10, 64); err == nil && rules.ValidLinuxAccountID(n) {
				val = r.account(val)
			} else {
				val = r.token("account", val)
			}
		case "device", "logdev", "rtdev", "journal_dev":
			val = r.reference(val)
		case "credentials", "cred", "secretfile", "keyfile", "password", "password2", "passwd", "pass", "username", "user", "user2", "domain", "workgroup", "secret", "sec", "addr", "ip", "servern", "clientaddr", "unc", "prefixpath", "context", "fscontext", "rootcontext", "defcontext":
			val = r.path(val)
		case "lowerdir":
			paths := strings.Split(val, ":")
			for j, p := range paths {
				paths[j] = r.path(p)
			}
			val = strings.Join(paths, ":")
		case "upperdir", "workdir", "subvol":
			val = r.path(val)
		default:
			if strings.HasPrefix(val, "/") {
				val = r.path(val)
			} else if !publicValueOptions[strings.ToLower(key)] || !safeOptionValue.MatchString(val) {
				val = r.token("option", val)
			}
		}
		opts[i] = key + "=" + val
	}
	return strings.Join(opts, ",")
}
func (r *redactor) account(s string) string {
	// Keep UID/GID observations linked while hiding local account IDs. Root is a
	// public semantic identity; other IDs use a separate deterministic namespace.
	if n, err := strconv.Atoi(s); err == nil {
		s = strconv.Itoa(n)
	}
	if s == "0" {
		return s
	}
	key := "account:" + s
	if v, ok := r.values[key]; ok {
		return v
	}
	v := strconv.Itoa(60000 + len(r.values))
	r.values[key] = v
	return v
}
func (r *redactor) fstab(b []byte) []byte {
	var out strings.Builder
	lines := strings.Split(string(b), "\n")
	for i, line := range lines {
		if i == len(lines)-1 && line == "" {
			break
		}
		f := strings.Fields(line)
		switch {
		case strings.HasPrefix(strings.TrimSpace(line), "#"):
			out.WriteString("# <sanitized-comment>")
		case len(f) == 0:
		case len(f) < 2:
			out.WriteString("sanitized-malformed-line")
		default:
			f[0] = r.reference(f[0])
			f[1] = r.path(f[1])
			if len(f) > 3 {
				f[3] = r.options(f[3])
			}
			// Inline comments and malformed extra fields can contain private data.
			for j := 6; j < len(f); j++ {
				f[j] = r.token("extra", f[j])
			}
			out.WriteString(strings.Join(f, " "))
		}
		out.WriteByte('\n')
	}
	return []byte(out.String())
}
func (r *redactor) inventory(inv rules.BlockDeviceInventory) rules.BlockDeviceInventory {
	out := inv
	out.Devices = make([]rules.BlockDevice, len(inv.Devices))
	out.Diagnostics = make([]rules.BlockDeviceDiagnostic, len(inv.Diagnostics))
	for i, d := range inv.Devices {
		d.Path = r.reference(d.Path)
		d.UUID = r.identity("uuid", d.UUID)
		d.PartUUID = r.identity("partuuid", d.PartUUID)
		d.Label = r.identity("label", d.Label)
		d.DeviceID = r.token("device-id", d.DeviceID)
		d.SysfsName = r.token("sysfs", d.SysfsName)
		aliases := make([]string, len(d.Aliases))
		for j, a := range d.Aliases {
			aliases[j] = r.reference(a)
		}
		d.Aliases = aliases
		mounts := make([]string, len(d.MountPoints))
		for j, m := range d.MountPoints {
			mounts[j] = r.path(m)
		}
		d.MountPoints = mounts
		out.Devices[i] = d
	}
	for i := range out.Diagnostics {
		out.Diagnostics[i] = rules.BlockDeviceDiagnostic{Source: "metadata", Error: "capture metadata unavailable (details redacted)"}
	}
	return out
}
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
func (r *redactor) observations(o Observations) Observations {
	out := o
	out.Files = map[string]rules.ExistState{}
	for _, k := range sortedKeys(o.Files) {
		out.Files[r.path(k)] = o.Files[k]
	}
	out.Blocks = map[string]rules.ExistState{}
	for _, k := range sortedKeys(o.Blocks) {
		out.Blocks[r.reference(k)] = o.Blocks[k]
	}
	out.Users = map[string]rules.ExistState{}
	for _, k := range sortedKeys(o.Users) {
		out.Users[r.account(k)] = o.Users[k]
	}
	out.Groups = map[string]rules.ExistState{}
	for _, k := range sortedKeys(o.Groups) {
		out.Groups[r.account(k)] = o.Groups[k]
	}
	out.Resolved = map[string]string{}
	for _, k := range sortedKeys(o.Resolved) {
		out.Resolved[r.reference(k)] = r.reference(o.Resolved[k])
	}
	if out.InventoryError != "" {
		out.InventoryError = "capture inventory unavailable (details redacted)"
	}
	if out.FilesystemError != "" {
		out.FilesystemError = "capture filesystem support unavailable (details redacted)"
	}
	return out
}
