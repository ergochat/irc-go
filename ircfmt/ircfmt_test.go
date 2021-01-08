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
	{"test $$c", "test $c"},
	{"test $c[]", "test \x03"},
	{"test $$", "test $"},
}

var escapetests = []testcase{
	{"te$c[]st", "te\x03st"},
	{"test$c[]", "test\x03"},
}

var unescapetests = []testcase{
	{"te$xt", "text"},
	{"te$st", "te\x1et"},
	{"test$c", "test\x03"},
	{"te$c[red velvet", "te\x03[red velvet"},
	{"te$c[[red velvet", "te\x03[[red velvet"},
	{"test$c[light blue,black]asdf", "test\x0312,1asdf"},
	{"test$c[light blue, black]asdf", "test\x0312,1asdf"},
	{"te$c[4,3]st", "te\x034,3st"},
	{"te$c[4]1st", "te\x03041st"},
	{"te$c[4,3]9st", "te\x034,039st"},
	{"te$c[04,03]9st", "te\x0304,039st"},
	{"te$c[asdf   !23a fd4*#]st", "te\x03st"},
	{"te$c[asdf  , !2,3a fd4*#]st", "te\x03st"},
	{"Client opered up $c[grey][$r%s$c[grey], $r%s$c[grey]]", "Client opered up \x0314[\x0f%s\x0314, \x0f%s\x0314]"},
}

var stripTests = []testcase{
	{"te\x02st", "test"},
	{"te\x033st", "test"},
	{"te\x034,3st", "test"},
	{"te\x03034st", "te4st"},
	{"te\x034,039st", "te9st"},
	{" ▀█▄▀▪.▀  ▀ ▀  ▀ ·▀▀▀▀  ▀█▄▀ ▀▀ █▪ ▀█▄▀▪", " ▀█▄▀▪.▀  ▀ ▀  ▀ ·▀▀▀▀  ▀█▄▀ ▀▀ █▪ ▀█▄▀▪"},
	{"test\x02case", "testcase"},
	{"", ""},
	{"test string", "test string"},
	{"test \x03", "test "},
	{"test \x0301string", "test string"},
	{"test\x031,2 string", "test string"},
	{"test\x0301,02 string", "test string"},
	{"test\x03, string", "test, string"},
	{"test\x03,12 string", "test string"},
	{"\x02\x031,2\x11\x16\x1d\x1e\x0f\x1f", ""},
	{"\x03", ""},
	{"\x03,", ","},
	{"\x031,2", ""},
	{"\x0315,1234", "34"},
	{"\x03151234", "1234"},
	{"\x03\x03\x03\x03\x03\x03\x03", ""},
	{"\x03\x03\x03\x03\x03\x03\x03\x03", ""},
	{"\x03,\x031\x0312\x0334,\x0356,\x0378,90\x031234", ",,,34"},
	{"\x0312,12\x03121212\x0311,333\x03,3\x038\x0399\x0355\x03test", "12123test"},
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

func TestChain(t *testing.T) {
	for _, pair := range tests {
		escaped := Escape(pair.unescaped)
		unescaped := Unescape(escaped)
		if unescaped != pair.unescaped {
			t.Errorf("for %q expected %q got %q", pair.unescaped, pair.unescaped, unescaped)
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

func TestStrip(t *testing.T) {
	for _, pair := range stripTests {
		val := Strip(pair.escaped)
		if val != pair.unescaped {
			t.Error(
				"For", pair.escaped,
				"expected", pair.unescaped,
				"got", val,
			)
		}
	}
}
