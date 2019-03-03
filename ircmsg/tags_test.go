package ircmsg

import (
	"fmt"
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
	{"0\\n1\\n2\\n3\\n4\\n5\\n6\\n\\", "0\n1\n2\n3\n4\n5\n6\n"},
	{"test\\", "test"},
	{"te\\:st\\", "te;st"},
	{"te\\:\\st\\", "te; t"},
	{"\\\\te\\:\\st", "\\te; t"},
	{"test\\", "test"},
	{"\\", ""},
	{"", ""},
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
	tags map[string]string
}

var tagdecodetests = []testtags{
	{"", map[string]string{}},
	{"time=12732;re", map[string]string{"time": "12732", "re": ""}},
	{"time=12732;re=;asdf=5678", map[string]string{"time": "12732", "re": "", "asdf": "5678"}},
}

func parseTags(rawTags string) (map[string]string, error) {
	message, err := ParseLineStrict(fmt.Sprintf("@%s :shivaram TAGMSG #darwin\r\n", rawTags), true, 0)
	return message.AllTags(), err
}

func TestDecodeTags(t *testing.T) {
	for _, pair := range tagdecodetests {
		tags, err := parseTags(pair.raw)
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
}
