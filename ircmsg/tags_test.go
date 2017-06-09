package ircmsg

import "testing"

type testcase struct {
	escaped   string
	unescaped string
}

var tests = []testcase{
	{"te\\nst", "te\nst"},
	{"tes\\\\st", "tes\\st"},
}

var unescapeTests = []testcase{
	{"te\\n\\kst", "te\nkst"},
	{"te\\n\\kst\\", "te\nkst"},
	{"te\\\\nst", "te\\nst"},
}

func TestEscape(t *testing.T) {
	for _, pair := range tests {
		val := EscapeTagValue(pair.unescaped)

		if val != pair.escaped {
			t.Error(
				"For", pair.unescaped,
				"expected", pair.escaped,
				"got", val,
			)
		}
	}
}

func TestUnescape(t *testing.T) {
	for _, pair := range tests {
		val := UnescapeTagValue(pair.escaped)

		if val != pair.unescaped {
			t.Error(
				"For", pair.escaped,
				"expected", pair.unescaped,
				"got", val,
			)
		}
	}
	for _, pair := range unescapeTests {
		val := UnescapeTagValue(pair.escaped)

		if val != pair.unescaped {
			t.Error(
				"For", pair.escaped,
				"expected", pair.unescaped,
				"got", val,
			)
		}
	}
}
