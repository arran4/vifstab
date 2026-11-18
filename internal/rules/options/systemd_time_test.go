package options

import "testing"

func TestSystemdDurationSyntax(t *testing.T) {
	valid := []string{"0", "+0", "30", "30s", "2min", "1h30min", "2 h", "55s500ms", "300ms20s 5day", "1y 12month", "infinity", " infinity\t", ".5", "+0.5h", "1.2s.3", "1.2 .3", "1s+2s", "1 2", "1μs", "1µs", "1.234567890s", "0.00000000000000000001us", "9223372036854775807us", "18446744073708s1551614us"}
	invalid := []string{"", " ", "-0", "-1s", "1s-2s", "garbage", "NaN", "1e3", "Infinity", "infinityx", "inf", "1ns", "1fortnight", "1.2.3", "1+2", "3.", "+", "3.sec", "3. 1", "0.-0", "3.+1", "0x10", "30s=bad", "9223372036854775808us", "18446744073708s1551615us", "18446744073710s"}
	for _, value := range valid {
		if !validSystemdDuration(value) {
			t.Errorf("documented/parser-compatible duration rejected: %q", value)
		}
	}
	for _, value := range invalid {
		if validSystemdDuration(value) {
			t.Errorf("malformed/overflowing duration accepted: %q", value)
		}
	}
}

func FuzzSystemdDuration(f *testing.F) {
	for _, value := range []string{"", "1h30min", "-0", "infinity", "18446744073708s1551614us", "1.2.3", "1μs", "\xff\x00"} {
		f.Add(value)
	}
	f.Fuzz(func(t *testing.T, value string) { _ = validSystemdDuration(value) })
}
