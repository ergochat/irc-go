package ircmsg

import (
	"reflect"
	"testing"
)

type testcode struct {
	raw     string
	message IrcMessage
}

var decodetests = []testcode{
	{":dan-!d@localhost PRIVMSG dan #test :What a cool message\r\n",
		MakeMessage(nil, "dan-!d@localhost", "PRIVMSG", "dan", "#test", "What a cool message")},
	{"@time=12732;re TEST *\r\n",
		MakeMessage(MakeTags("time", "12732", "re", nil), "", "TEST", "*")},
}

func TestDecode(t *testing.T) {
	for _, pair := range decodetests {
		ircmsg, err := ParseLine(pair.raw)
		if err != nil {
			t.Error(
				"For", pair.raw,
				"Failed to parse line:", err,
			)
		}

		if !reflect.DeepEqual(ircmsg, pair.message) {
			t.Error(
				"For", pair.raw,
				"expected", pair.message,
				"got", ircmsg,
			)
		}
	}
}

var encodetests = []testcode{
	{":dan-!d@localhost PRIVMSG dan #test :What a cool message\r\n",
		MakeMessage(nil, "dan-!d@localhost", "PRIVMSG", "dan", "#test", "What a cool message")},
	{"@time=12732 TEST *\r\n",
		MakeMessage(MakeTags("time", "12732"), "", "TEST", "*")},
	{"@re TEST *\r\n",
		MakeMessage(MakeTags("re", nil), "", "TEST", "*")},
}

func TestEncode(t *testing.T) {
	for _, pair := range encodetests {
		line, err := pair.message.Line()
		if err != nil {
			t.Error(
				"For", pair.raw,
				"Failed to parse line:", err,
			)
		}

		if line != pair.raw {
			t.Error(
				"For", pair.message,
				"expected", pair.raw,
				"got", line,
			)
		}
	}

	// make sure we fail on no command
	msg := MakeMessage(nil, "example.com", "", "*")
	_, err := msg.Line()
	if err == nil {
		t.Error(
			"For", "Test Failure 1",
			"expected", "an error",
			"got", err,
		)
	}

	// make sure we fail with params in right way
	msg = MakeMessage(nil, "example.com", "TEST", "*", "t s", "", "Param after empty!")
	_, err = msg.Line()
	if err == nil {
		t.Error(
			"For", "Test Failure 2",
			"expected", "an error",
			"got", err,
		)
	}
}
