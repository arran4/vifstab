package rules

import (
	"bufio"
	"errors"
	"io/fs"
	"strconv"
	"strings"
)

// MaxLinuxAccountID excludes the reserved invalid uid_t/gid_t value. This is
// syntax/range policy, not proof that an ID is mapped in the caller's namespace.
const MaxLinuxAccountID = uint64(1<<32 - 2)

func ValidLinuxAccountID(id int64) bool {
	return id >= 0 && uint64(id) <= MaxLinuxAccountID
}

// A local account file can establish a valid matching observation. Its EOF is
// not global absence: NSS may use other sources, and incomplete/malformed local
// data is not a complete inventory. Explicit provider/cache observations remain
// the route for known negatives; this helper never invents one from file data.
func observeAccountFile(open func(string) (fs.File, error), name string, id int, user bool) ExistState {
	if !ValidLinuxAccountID(int64(id)) {
		return Unknown
	}
	f, err := open(name)
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			return PermissionDenied
		}
		return Unknown
	}
	if f == nil {
		return Unknown
	}
	defer func() { _ = f.Close() }()

	columns := 4
	if user {
		columns = 7
	}
	scanner := bufio.NewScanner(f)
	terminated := false
	scanner.Split(func(data []byte, atEOF bool) (int, []byte, error) {
		advance, token, err := bufio.ScanLines(data, atEOF)
		terminated = advance > 0 && data[advance-1] == '\n'
		return advance, token, err
	})
	for scanner.Scan() {
		// A read fault can expose a structurally plausible partial final row.
		// Accept a final row without LF only at ordinary EOF, not after I/O loss.
		if !terminated && scanner.Err() != nil {
			continue
		}
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.IndexByte(line, 0) >= 0 {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) != columns || parts[0] == "" || parts[0][0] == '+' || parts[0][0] == '-' {
			continue
		}
		observed, valid := accountRecordID(parts[2])
		if !valid || observed != uint64(id) {
			continue
		}
		if user {
			if _, valid := accountRecordID(parts[3]); !valid {
				continue
			}
		}
		return Exists
	}
	if errors.Is(scanner.Err(), fs.ErrPermission) {
		return PermissionDenied
	}
	return Unknown
}

func accountRecordID(text string) (uint64, bool) {
	value, err := strconv.ParseUint(text, 10, 32)
	return value, err == nil && value <= MaxLinuxAccountID
}
