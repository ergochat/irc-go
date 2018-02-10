package ircmsg

import (
	"reflect"
	"testing"
)

type testcase struct {
	escaped   string
	unescaped string
}

var tests = []testcase{
	{"te\\nst", "te\nst"},
	{"tes\\\\st", "tes\\st"},
	{"te😃st", "te😃st"},
}

var unescapeTests = []testcase{
	{"te\\n\\kst", "te\nkst"},
	{"te\\n\\kst\\", "te\nkst"},
	{"te\\\\nst", "te\\nst"},
	{"te😃st", "te😃st"},
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

// tag string tests
type testtags struct {
	raw  string
	tags map[string]TagValue
}
type testtagswithlen struct {
	raw    string
	length int
	tags   map[string]TagValue
}

var tagdecodelentests = []testtagswithlen{
	{"time=12732;re", 512, *MakeTags("time", "12732", "re", nil)},
	{"time=12732;re", 12, *MakeTags("time", "12732", "r", nil)},
	{"", 512, *MakeTags()},
}
var tagdecodetests = []testtags{
	{"", *MakeTags()},
	{"time=12732;re", *MakeTags("time", "12732", "re", nil)},
}
var tagdecodetesterrors = []string{
	"\r\n",
	"     \r\n",
	"tags=tesa\r\n",
	"tags=tested  \r\n",
}

func TestDecodeTags(t *testing.T) {
	for _, pair := range tagdecodelentests {
		tags, err := parseTags(pair.raw, pair.length, true)
		if err != nil {
			t.Error(
				"For", pair.raw,
				"Failed to parse tags:", err,
			)
		}

		if !reflect.DeepEqual(tags, pair.tags) {
			t.Error(
				"For", pair.raw,
				"expected", pair.tags,
				"got", tags,
			)
		}
	}
	for _, pair := range tagdecodetests {
		tags, err := ParseTags(pair.raw)
		if err != nil {
			t.Error(
				"For", pair.raw,
				"Failed to parse line:", err,
			)
		}

		if !reflect.DeepEqual(tags, pair.tags) {
			t.Error(
				"For", pair.raw,
				"expected", pair.tags,
				"got", tags,
			)
		}
	}
	for _, line := range tagdecodetesterrors {
		_, err := ParseTags(line)
		if err == nil {
			t.Error(
				"Expected to fail parsing", line,
			)
		}
	}
}
