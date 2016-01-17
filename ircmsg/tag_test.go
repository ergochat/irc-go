package ircmsg

import "testing"

type testcase struct {
	escaped   string
	unescaped string
}

var tests = []testcase{
	{"te\\rst", "te\rst"},
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
}
