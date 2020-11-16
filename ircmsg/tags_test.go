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

func TestValidateTagName(t *testing.T) {
	if !validateTagName("c") {
		t.Error("c is valid")
	}
	if validateTagName("a_b") {
		t.Error("a_b is invalid")
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
	{"time=12732;draft/label=b;re=;asdf=5678", map[string]string{"time": "12732", "re": "", "asdf": "5678", "draft/label": "b"}},
	{"=these;time=12732;=shouldbe;re=;asdf=5678;=ignored", map[string]string{"time": "12732", "re": "", "asdf": "5678"}},
	{"dolphin=🐬;time=123456", map[string]string{"dolphin": "🐬", "time": "123456"}},
	{"+dolphin=🐬;+draft/fox=f🦊x", map[string]string{"+dolphin": "🐬", "+draft/fox": "f🦊x"}},
	{"+dolphin=🐬;+draft/f🦊x=fox", map[string]string{"+dolphin": "🐬"}},
	{"+dolphin=🐬;+f🦊x=fox", map[string]string{"+dolphin": "🐬"}},
	{"+dolphin=🐬;f🦊x=fox", map[string]string{"+dolphin": "🐬"}},
	{"dolphin=🐬;f🦊x=fox", map[string]string{"dolphin": "🐬"}},
	{"f🦊x=fox;+oragono.io/dolphin=🐬", map[string]string{"+oragono.io/dolphin": "🐬"}},
	{"a=b;\\/=.", map[string]string{"a": "b"}},
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

var invalidtagdatatests = []string{
	"label=\xff;batch=c",
	"label=a\xffb;batch=c",
	"label=a\xffb",
	"label=a\xff",
	"label=a\xff",
	"label=a\xf0a",
}

func TestTagInvalidUtf8(t *testing.T) {
	for _, tags := range invalidtagdatatests {
		_, err := ParseLineStrict(fmt.Sprintf("@%s PRIVMSG #chan hi\r\n", tags), true, 0)
		if err != ErrorInvalidTagContent {
			t.Errorf("")
		}
	}
}
