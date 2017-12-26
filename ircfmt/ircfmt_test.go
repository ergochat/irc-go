package ircfmt

import "testing"

type testcase struct {
	escaped   string
	unescaped string
}

var tests = []testcase{
	{"te$bst", "te\x02st"},
	{"te$c[green]st", "te\x033st"},
	{"te$c[red,green]st", "te\x034,3st"},
	{"te$c[green]4st", "te\x03034st"},
	{"te$c[red,green]9st", "te\x034,039st"},
	{" ▀█▄▀▪.▀  ▀ ▀  ▀ ·▀▀▀▀  ▀█▄▀ ▀▀ █▪ ▀█▄▀▪", " ▀█▄▀▪.▀  ▀ ▀  ▀ ·▀▀▀▀  ▀█▄▀ ▀▀ █▪ ▀█▄▀▪"},
}

var escapetests = []testcase{
	{"te$c[]st", "te\x03st"},
	{"test$c[]", "test\x03"},
}

var unescapetests = []testcase{
	{"te$xt", "text"},
	{"te$st", "te\x1et"},
	{"test$c", "test\x03"},
}

func TestEscape(t *testing.T) {
	for _, pair := range tests {
		val := Escape(pair.unescaped)

		if val != pair.escaped {
			t.Error(
				"For", pair.unescaped,
				"expected", pair.escaped,
				"got", val,
			)
		}
	}
	for _, pair := range escapetests {
		val := Escape(pair.unescaped)

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
		val := Unescape(pair.escaped)

		if val != pair.unescaped {
			t.Error(
				"For", pair.escaped,
				"expected", pair.unescaped,
				"got", val,
			)
		}
	}
	for _, pair := range unescapetests {
		val := Unescape(pair.escaped)

		if val != pair.unescaped {
			t.Error(
				"For", pair.escaped,
				"expected", pair.unescaped,
				"got", val,
			)
		}
	}
}
