package options

import (
	"math"
	"strconv"
	"strings"
)

// Microsecond multipliers follow the documented systemd time-span vocabulary.
// Validate only: nothing sleeps, schedules, consults the clock or invokes a tool.
var systemdTimeUnits = map[string]uint64{
	"":     1000000,
	"usec": 1, "us": 1, "μs": 1, "µs": 1,
	"msec": 1000, "ms": 1000,
	"seconds": 1000000, "second": 1000000, "sec": 1000000, "s": 1000000,
	"minutes": 60000000, "minute": 60000000, "min": 60000000, "m": 60000000,
	"hours": 3600000000, "hour": 3600000000, "hr": 3600000000, "h": 3600000000,
	"days": 86400000000, "day": 86400000000, "d": 86400000000,
	"weeks": 604800000000, "week": 604800000000, "w": 604800000000,
	"months": 2629800000000, "month": 2629800000000, "M": 2629800000000,
	"years": 31557600000000, "year": 31557600000000, "y": 31557600000000,
}

func systemdSpace(b byte) bool { return strings.IndexByte(" \t\r\n\v\f", b) >= 0 }

// validSystemdDuration checks documented nonnegative decimal/compound spans,
// including infinity, plus signs and fractional components. Checked arithmetic
// bounds the same microsecond domain without Go time.Duration's smaller limit.
func validSystemdDuration(value string) bool {
	value = strings.Trim(value, " \t\r\n\v\f")
	if value == "infinity" {
		return true
	}
	if value == "" {
		return false
	}
	var total uint64
	for pos := 0; pos < len(value); {
		for pos < len(value) && systemdSpace(value[pos]) {
			pos++
		}
		if pos == len(value) {
			return true
		}
		if value[pos] == '+' {
			pos++
		}
		start := pos
		for pos < len(value) && value[pos] >= '0' && value[pos] <= '9' {
			pos++
		}
		integerText := value[start:pos]
		fraction := ""
		hasFraction := pos < len(value) && value[pos] == '.'
		if hasFraction {
			pos++
			start = pos
			for pos < len(value) && value[pos] >= '0' && value[pos] <= '9' {
				pos++
			}
			fraction = value[start:pos]
			if fraction == "" {
				return false
			}
		}
		if integerText == "" && !hasFraction {
			return false
		}
		integer := uint64(0)
		if integerText != "" {
			var err error
			integer, err = strconv.ParseUint(integerText, 10, 63)
			if err != nil {
				return false
			}
		}
		beforeSpace := pos
		for pos < len(value) && systemdSpace(value[pos]) {
			pos++
		}
		unitStart := pos
		for pos < len(value) && (value[pos] >= 'a' && value[pos] <= 'z' || value[pos] >= 'A' && value[pos] <= 'Z' || value[pos] >= 128) {
			pos++
		}
		multiplier, ok := systemdTimeUnits[value[unitStart:pos]]
		if !ok || integer >= math.MaxUint64/multiplier {
			return false
		}
		// Adjacent fractions need either a unit or whitespace: 1.2.3 is invalid,
		// while 1.2s.3 and 1.2 .3 are documented parser-compatible compounds.
		if pos < len(value) && unitStart == pos && beforeSpace == pos {
			return false
		}
		component := integer * multiplier
		for i, place := 0, multiplier/10; i < len(fraction) && place != 0; i, place = i+1, place/10 {
			addition := uint64(fraction[i]-'0') * place
			if addition >= math.MaxUint64-component {
				return false
			}
			component += addition
		}
		if component >= math.MaxUint64-total {
			return false
		}
		total += component
	}
	return true
}
