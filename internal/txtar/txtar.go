// Package txtar shares bounded snapshot parsing and provider construction between
// the CLI and regression fixtures. Archives are never extracted to disk.
package txtar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing/fstest"

	"github.com/arran4/vifstab/internal/editor"
	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/textinput"
	x_txtar "golang.org/x/tools/txtar"
)

type Manifest struct {
	Version  string `json:"version"`
	Type     string `json:"type"`
	Redacted *bool  `json:"redacted"`
}

// Observations keeps negative, unknown and permission-denied results separate.
// Maps use original query strings, translated by one redactor before serialization.
type Observations struct {
	Files           map[string]rules.ExistState `json:"files,omitempty"`
	Blocks          map[string]rules.ExistState `json:"blocks,omitempty"`
	Users           map[string]rules.ExistState `json:"users,omitempty"`
	Groups          map[string]rules.ExistState `json:"groups,omitempty"`
	Resolved        map[string]string           `json:"resolved,omitempty"`
	Filesystems     map[string]bool             `json:"filesystems"`
	FilesystemError string                      `json:"filesystem_error,omitempty"`
	InventoryError  string                      `json:"inventory_error,omitempty"`
}

type Snapshot struct {
	Fstab        []byte
	Cache        map[string]string
	MockFS       fstest.MapFS
	KernelVer    string
	Manifest     *Manifest
	Observations *Observations
}

const MaxArchiveSize = 10 * 1024 * 1024
const MaxSectionSize = 2 * 1024 * 1024
const MaxSectionCount = 100
const MaxNameLength = 1024

// Read bounds allocation before parsing, including for stdin.
func Read(r io.Reader) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, MaxArchiveSize+1))
	if err != nil {
		return nil, err
	}
	if len(b) > MaxArchiveSize {
		return nil, fmt.Errorf("archive exceeds limit of %d bytes", MaxArchiveSize)
	}
	return b, nil
}

// decodeJSON rejects unknown and duplicate object fields as well as trailing data.
func decodeJSON(data []byte, target any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	depth := 0
	var value func() error
	value = func() error {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		if delim, ok := tok.(json.Delim); ok {
			depth++
			if depth > 100 {
				return fmt.Errorf("JSON nesting exceeds limit")
			}
			defer func() { depth-- }()
			switch delim {
			case '{':
				seen := map[string]bool{}
				for d.More() {
					k, err := d.Token()
					if err != nil {
						return err
					}
					key, ok := k.(string)
					if !ok || seen[key] {
						return fmt.Errorf("duplicate or invalid JSON field %v", k)
					}
					seen[key] = true
					if err := value(); err != nil {
						return err
					}
				}
			case '[':
				for d.More() {
					if err := value(); err != nil {
						return err
					}
				}
			default:
				return fmt.Errorf("unexpected JSON delimiter")
			}
			_, err = d.Token()
			return err
		}
		return nil
	}
	if err := value(); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("trailing JSON data")
	}
	// encoding/json accepts case-insensitive struct field aliases. Require the
	// schema's exact field names so an alias cannot override an earlier field.
	t := reflect.TypeOf(target)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(data, &fields); err != nil {
			return err
		}
		allowed := map[string]bool{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			name := strings.Split(f.Tag.Get("json"), ",")[0]
			if name == "" {
				name = f.Name
			}
			allowed[name] = true
		}
		for key := range fields {
			if !allowed[key] {
				return fmt.Errorf("unknown JSON field %q", key)
			}
		}
	}
	d = json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	return d.Decode(target)
}

func ParseTXTAR(data []byte) (*Snapshot, error) {
	if len(data) > MaxArchiveSize {
		return nil, fmt.Errorf("archive size %d exceeds limit of %d", len(data), MaxArchiveSize)
	}
	ar := x_txtar.Parse(data)
	if len(ar.Files) > MaxSectionCount {
		return nil, fmt.Errorf("archive exceeds maximum section count")
	}
	s := &Snapshot{Cache: map[string]string{}, MockFS: fstest.MapFS{}}
	seen := map[string]bool{}
	hasFstab := false
	for _, file := range ar.Files {
		name := file.Name
		if !fs.ValidPath(name) || name == "." || strings.ContainsAny(name, "\\\x00\r\n") {
			return nil, fmt.Errorf("unsafe path in archive: %s", name)
		}
		if len(name) > MaxNameLength {
			return nil, fmt.Errorf("archive file name exceeds maximum length")
		}
		if len(file.Data) > MaxSectionSize {
			return nil, fmt.Errorf("archive file %s exceeds maximum section size", name)
		}
		if seen[name] && name != "cache" {
			return nil, fmt.Errorf("duplicate file in archive: %s", name)
		}
		seen[name] = true
		switch {
		case name == "manifest.json":
			m := &Manifest{}
			if err := decodeJSON(file.Data, m); err != nil {
				return nil, fmt.Errorf("malformed manifest.json: %w", err)
			}
			if m.Redacted == nil {
				return nil, fmt.Errorf("manifest missing required field 'redacted'")
			}
			if m.Version != "1.0" {
				return nil, fmt.Errorf("unsupported manifest version: %s", m.Version)
			}
			if m.Type != "system" {
				return nil, fmt.Errorf("unsupported manifest type: %s", m.Type)
			}
			s.Manifest = m
		case name == "provider/inventory.json":
			var inv rules.BlockDeviceInventory
			if err := decodeJSON(file.Data, &inv); err != nil || bytes.Equal(bytes.TrimSpace(file.Data), []byte("null")) {
				return nil, fmt.Errorf("malformed provider/inventory.json: %v", err)
			}
			if _, ok := s.MockFS[name]; ok {
				return nil, fmt.Errorf("ambiguous provider inventory section")
			}
			s.MockFS[name] = &fstest.MapFile{Data: file.Data}
		case name == "provider/observations.json":
			var o *Observations
			if err := decodeJSON(file.Data, &o); err != nil || o == nil {
				return nil, fmt.Errorf("malformed provider/observations.json: %v", err)
			}
			// JSON null must not decode into ExistState's zero value (Exists).
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(file.Data, &fields); err != nil {
				return nil, err
			}
			for name, raw := range fields {
				switch strings.ToLower(name) {
				case "files", "blocks", "users", "groups", "filesystems":
					var values map[string]json.RawMessage
					if err := json.Unmarshal(raw, &values); err != nil {
						return nil, err
					}
					for key, value := range values {
						if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
							return nil, fmt.Errorf("null observation for %q", key)
						}
					}
				}
			}
			for _, states := range []map[string]rules.ExistState{o.Files, o.Blocks, o.Users, o.Groups} {
				for key, state := range states {
					if state < rules.Exists || state > rules.Unknown {
						return nil, fmt.Errorf("invalid observation state for %q", key)
					}
				}
			}
			for _, states := range []map[string]rules.ExistState{o.Users, o.Groups} {
				for key := range states {
					if n, err := strconv.Atoi(key); err != nil || n < 0 || strconv.Itoa(n) != key {
						return nil, fmt.Errorf("invalid account ID %q", key)
					}
				}
			}
			s.Observations = o
		case name == "input" || name == "input/etc/fstab":
			if hasFstab {
				return nil, fmt.Errorf("duplicate or ambiguous fstab section in archive")
			}
			hasFstab = true
			s.Fstab = append([]byte{}, file.Data...)
		case name == "kernel":
			s.KernelVer = strings.TrimSpace(string(file.Data))
			if s.KernelVer != "" && !strings.HasPrefix(s.KernelVer, "Linux version ") {
				s.KernelVer = "Linux version " + s.KernelVer
			}
		case name == "cache":
			for _, line := range strings.Split(string(file.Data), "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				k, v, ok := strings.Cut(line, "=")
				if !ok || k == "" {
					return nil, fmt.Errorf("malformed cache entry")
				}
				if old, ok := s.Cache[k]; ok && old != v {
					return nil, fmt.Errorf("conflicting cache key %q", k)
				}
				s.Cache[k] = v
			}
		case strings.HasPrefix(name, "input/"):
			key := strings.TrimPrefix(name, "input/")
			if strings.HasPrefix(key, "provider/") {
				return nil, fmt.Errorf("reserved provider path in input section")
			}
			s.MockFS[key] = &fstest.MapFile{Data: file.Data}
		}
	}
	if !hasFstab {
		return nil, fmt.Errorf("invalid archive: missing fstab data")
	}
	// Validate known legacy cache values with the same adapter used by replay.
	if _, err := ReplaySnapshot(s); err != nil {
		return nil, err
	}
	return s, nil
}

func legacyState(v string) (rules.ExistState, error) {
	switch v {
	case "true":
		return rules.Exists, nil
	case "false":
		return rules.DoesNotExist, nil
	case "permission_denied":
		return rules.PermissionDenied, nil
	case "unknown":
		return rules.Unknown, nil
	}
	return rules.Unknown, fmt.Errorf("invalid existence observation %q", v)
}

func ReplaySnapshot(s *Snapshot) (rules.API, error) {
	if s == nil {
		return nil, fmt.Errorf("missing snapshot")
	}
	c := rules.NewCache()
	if s.KernelVer != "" {
		c.Set("proc_version", s.KernelVer)
	}
	for k, v := range s.Cache {
		switch {
		case k == "proc_filesystems":
			m := map[string]bool{}
			for _, name := range strings.Split(v, ",") {
				if name = strings.TrimSpace(name); name != "" {
					m[name] = true
				}
			}
			c.Set(rules.CacheKey(k), m)
		case k == "block_devices":
			var inv rules.BlockDeviceInventory
			if strings.HasPrefix(strings.TrimSpace(v), "[") {
				if err := decodeJSON([]byte(v), &inv.Devices); err != nil {
					return nil, fmt.Errorf("malformed block_devices cache: %w", err)
				}
				inv.Available = true
			} else if err := decodeJSON([]byte(v), &inv); err != nil || strings.TrimSpace(v) == "null" {
				return nil, fmt.Errorf("malformed block_devices cache: %v", err)
			}
			c.Set(rules.CacheKey(k), inv)
		case strings.HasPrefix(k, "file_exists:"), strings.HasPrefix(k, "block_exists:"), strings.HasPrefix(k, "block_device_exists:"), strings.HasPrefix(k, "user_exists:"), strings.HasPrefix(k, "group_exists:"):
			st, err := legacyState(v)
			if err != nil {
				return nil, err
			}
			k = strings.Replace(k, "block_exists:", "block_device_exists:", 1)
			if _, ok := c.Get(rules.CacheKey(k)); ok {
				return nil, fmt.Errorf("ambiguous cache key %q", k)
			}
			c.Set(rules.CacheKey(k), st)
		default:
			switch v {
			case "true":
				c.Set(rules.CacheKey(k), true)
			case "false":
				c.Set(rules.CacheKey(k), false)
			default:
				c.Set(rules.CacheKey(k), v)
			}
		}
	}
	mockFS := fstest.MapFS{}
	for name, f := range s.MockFS {
		mockFS[name] = f
	}
	if f, ok := mockFS["provider/inventory.json"]; ok {
		if _, ok := s.Cache["block_devices"]; ok {
			return nil, fmt.Errorf("ambiguous inventory sections")
		}
		var inv rules.BlockDeviceInventory
		if err := decodeJSON(f.Data, &inv); err != nil {
			return nil, err
		}
		c.Set("block_devices", inv)
	}
	if o := s.Observations; o != nil {
		if o.Filesystems != nil || o.FilesystemError != "" {
			if _, ok := s.Cache["proc_filesystems"]; ok {
				return nil, fmt.Errorf("ambiguous filesystem support sections")
			}
			if _, ok := mockFS["proc/filesystems"]; ok {
				return nil, fmt.Errorf("ambiguous filesystem support sections")
			}
		}
		for prefix, states := range map[string]map[string]rules.ExistState{"file_exists:": o.Files, "block_device_exists:": o.Blocks, "user_exists:": o.Users, "group_exists:": o.Groups} {
			for k, v := range states {
				key := rules.CacheKey(prefix + k)
				if _, ok := c.Get(key); ok {
					return nil, fmt.Errorf("ambiguous observation %q", key)
				}
				c.Set(key, v)
			}
		}
		for k, v := range o.Resolved {
			c.Set(rules.CacheKey("resolved_device:"+k), v)
		}
		if o.Filesystems != nil {
			c.Set("proc_filesystems", o.Filesystems)
		}
		if o.FilesystemError != "" {
			c.Set("proc_filesystems_error", o.FilesystemError)
		}
		if o.InventoryError != "" {
			c.Set("block_devices_error", o.InventoryError)
		}
	}
	if s.Fstab != nil {
		mockFS["etc/fstab"] = &fstest.MapFile{Data: s.Fstab}
	}
	api := rules.NewCacheFS(c, mockFS)
	api.OfflineReplay = true
	return api, nil
}

type recordingAPI struct {
	rules.API
	observations Observations
	inventory    rules.BlockDeviceInventory
	inventoryErr error
	kernel       string
}

func (r *recordingAPI) CheckFileExists(p string) rules.ExistState {
	if s, ok := r.observations.Files[p]; ok {
		return s
	}
	s := r.API.CheckFileExists(p)
	r.observations.Files[p] = s
	return s
}
func (r *recordingAPI) CheckBlockDeviceExists(p string) rules.ExistState {
	if s, ok := r.observations.Blocks[p]; ok {
		return s
	}
	s := r.API.CheckBlockDeviceExists(p)
	r.observations.Blocks[p] = s
	return s
}
func (r *recordingAPI) CheckUserExists(id int) rules.ExistState {
	k := strconv.Itoa(id)
	if s, ok := r.observations.Users[k]; ok {
		return s
	}
	s := r.API.CheckUserExists(id)
	r.observations.Users[k] = s
	return s
}
func (r *recordingAPI) CheckGroupExists(id int) rules.ExistState {
	k := strconv.Itoa(id)
	if s, ok := r.observations.Groups[k]; ok {
		return s
	}
	s := r.API.CheckGroupExists(id)
	r.observations.Groups[k] = s
	return s
}
func (r *recordingAPI) ResolveDevicePath(p string) string {
	if s, ok := r.observations.Resolved[p]; ok {
		return s
	}
	s := r.API.ResolveDevicePath(p)
	r.observations.Resolved[p] = s
	return s
}
func (r *recordingAPI) GetBlockDevices() (rules.BlockDeviceInventory, error) {
	return r.inventory, r.inventoryErr
}
func (r *recordingAPI) GetHostKernelVersion() string { return r.kernel }
func (r *recordingAPI) GetSupportedFilesystems() (map[string]bool, error) {
	if r.observations.FilesystemError != "" {
		return nil, fmt.Errorf("%s", r.observations.FilesystemError)
	}
	return r.observations.Filesystems, nil
}

// CaptureResult separates validation status from archive construction. A failed
// verification is often precisely the support case the user needs to retain.
type CaptureResult struct {
	Archive         []byte
	Issues          []rules.ValidationIssue
	ValidationError error
}

// CaptureSystem preserves the existing file-based capture API.
func CaptureSystem(ctx context.Context, api rules.API, fstabPath string, unredacted bool) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := textinput.Open(fstabPath)
	if err != nil {
		return nil, fmt.Errorf("read fstab: %w", err)
	}
	result, err := CaptureReader(ctx, api, f, unredacted)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, err
	}
	if result.ValidationError != nil {
		return nil, fmt.Errorf("capture validation: %w", result.ValidationError)
	}
	return result.Archive, nil
}

// CaptureReader records production validation and returns its findings alongside
// the bounded archive. It never treats blocking findings as a capture failure.
// The caller owns input and may explicitly supply stdin.
func CaptureReader(ctx context.Context, api rules.API, input io.Reader, unredacted bool) (*CaptureResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	b, err := io.ReadAll(io.LimitReader(input, MaxSectionSize+1))
	if err != nil {
		return nil, err
	}
	if len(b) > MaxSectionSize {
		return nil, fmt.Errorf("fstab exceeds maximum section size")
	}
	r := &recordingAPI{API: api, observations: Observations{Files: map[string]rules.ExistState{}, Blocks: map[string]rules.ExistState{}, Users: map[string]rules.ExistState{}, Groups: map[string]rules.ExistState{}, Resolved: map[string]string{}}}
	r.inventory, r.inventoryErr = api.GetBlockDevices()
	if r.inventoryErr != nil {
		r.observations.InventoryError = r.inventoryErr.Error()
	}
	r.kernel = api.GetHostKernelVersion()
	r.observations.Filesystems, err = rules.SupportedFilesystems(api)
	if err != nil {
		r.observations.FilesystemError = err.Error()
	}
	issues, validationErr := editor.VerifyAndLintReader(bytes.NewReader(b), r)
	// Also retain direct source/target observations for callers using the provider
	// beyond the initial rule set. These are metadata existence checks only.
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		r.CheckFileExists(fields[1])
		r.CheckBlockDeviceExists(fields[0])
		r.ResolveDevicePath(fields[0])
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !unredacted {
		red := newRedactor()
		b = red.fstab(b)
		r.inventory = red.inventory(r.inventory)
		r.observations = red.observations(r.observations)
		// Keep the version used by validation, excluding build usernames/hostnames.
		r.kernel = red.kernel(r.kernel)
	}
	redacted := !unredacted
	manifest := Manifest{Version: "1.0", Type: "system", Redacted: &redacted}
	ar := &x_txtar.Archive{Files: []x_txtar.File{{Name: "input/etc/fstab", Data: b}}}
	for name, v := range map[string]any{"manifest.json": manifest, "provider/inventory.json": r.inventory, "provider/observations.json": r.observations} {
		data, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		ar.Files = append(ar.Files, x_txtar.File{Name: name, Data: append(data, '\n')})
	}
	if r.kernel != "" {
		ar.Files = append(ar.Files, x_txtar.File{Name: "kernel", Data: []byte(r.kernel + "\n")})
	}
	sort.Slice(ar.Files, func(i, j int) bool { return ar.Files[i].Name < ar.Files[j].Name })
	data := x_txtar.Format(ar)
	// Enforce the same bounds on generated archives as on replay input. TXTAR
	// delimiter injection in an unredacted field must never create extra sections.
	_, err = ParseTXTAR(data)
	if err != nil {
		return nil, fmt.Errorf("captured archive: %w", err)
	}
	formatted := x_txtar.Parse(data)
	if len(formatted.Files) != len(ar.Files) {
		return nil, fmt.Errorf("capture contains TXTAR section delimiter")
	}
	for i, file := range ar.Files {
		actual := formatted.Files[i]
		if actual.Name != file.Name || (!bytes.Equal(actual.Data, file.Data) && !bytes.Equal(actual.Data, append(bytes.Clone(file.Data), '\n'))) {
			return nil, fmt.Errorf("capture section %s contains TXTAR delimiter", file.Name)
		}
	}
	return &CaptureResult{Archive: data, Issues: issues, ValidationError: validationErr}, nil
}
