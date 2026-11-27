package rules

import (
	"bufio"
	"errors"
	"io"
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
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		// A complete line is usable even if subsequent input is unavailable.
		// An unterminated final record is usable only at ordinary EOF.
		if strings.HasSuffix(line, "\n") || errors.Is(err, io.EOF) {
			line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
			if accountLineMatches(line, columns, id, user) {
				return Exists
			}
		}
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				return PermissionDenied
			}
			return Unknown
		}
	}
}

func accountLineMatches(line string, columns, id int, user bool) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.IndexByte(line, 0) >= 0 {
		return false
	}
	parts := strings.Split(line, ":")
	if len(parts) != columns || parts[0] == "" || parts[0][0] == '+' || parts[0][0] == '-' {
		return false
	}
	observed, valid := accountRecordID(parts[2])
	if !valid || observed != uint64(id) {
		return false
	}
	if user {
		if _, valid := accountRecordID(parts[3]); !valid {
			return false
		}
	}
	return true
}

func accountRecordID(text string) (uint64, bool) {
	value, err := strconv.ParseUint(text, 10, 32)
	return value, err == nil && value <= MaxLinuxAccountID
}
