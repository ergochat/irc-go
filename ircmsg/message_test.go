package ircmsg

import (
	"reflect"
	"strings"
	"testing"
)

type testcode struct {
	raw     string
	message IrcMessage
}
type testcodewithlen struct {
	raw     string
	length  int
	message IrcMessage
}

var decodelentests = []testcodewithlen{
	{":dan-!d@localhost PRIVMSG dan #test :What a cool message\r\n", 20,
		MakeMessage(nil, "dan-!d@localhost", "PR")},
	{"@time=12732;re TEST *\r\n", 512,
		MakeMessage(MakeTags("time", "12732", "re", nil), "", "TEST", "*")},
	{"@time=12732;re TEST *\r\n", 12,
		MakeMessage(MakeTags("time", "12732", "r", nil), "", "TEST", "*")},
	{":dan- TESTMSG\r\n", 2048,
		MakeMessage(nil, "dan-", "TESTMSG")},
	{":dan- TESTMSG dan \r\n", 12,
		MakeMessage(nil, "dan-", "TESTMS")},
	{"TESTMSG\r\n", 6,
		MakeMessage(nil, "", "TESTMS")},
	{"TESTMSG\r\n", 7,
		MakeMessage(nil, "", "TESTMSG")},
	{"TESTMSG\r\n", 8,
		MakeMessage(nil, "", "TESTMSG")},
	{"TESTMSG\r\n", 9,
		MakeMessage(nil, "", "TESTMSG")},
}
var decodetests = []testcode{
	{":dan-!d@localhost PRIVMSG dan #test :What a cool message\r\n",
		MakeMessage(nil, "dan-!d@localhost", "PRIVMSG", "dan", "#test", "What a cool message")},
	{"@time=12732;re TEST *a asda:fs :fhye tegh\r\n",
		MakeMessage(MakeTags("time", "12732", "re", nil), "", "TEST", "*a", "asda:fs", "fhye tegh")},
	{"@time=12732;re TEST *\r\n",
		MakeMessage(MakeTags("time", "12732", "re", nil), "", "TEST", "*")},
	{":dan- TESTMSG\r\n",
		MakeMessage(nil, "dan-", "TESTMSG")},
	{":dan- TESTMSG dan \r\n",
		MakeMessage(nil, "dan-", "TESTMSG", "dan")},
}
var decodetesterrors = []string{
	"\r\n",
	"     \r\n",
	"@tags=tesa\r\n",
	"@tags=tested  \r\n",
	":dan-   \r\n",
	":dan-\r\n",
}

func TestDecode(t *testing.T) {
	for _, pair := range decodelentests {
		ircmsg, err := ParseLineMaxLen(pair.raw, pair.length, pair.length)
		if err != nil {
			t.Error(
				"For", pair.raw,
				"Failed to parse line:", err,
			)
		}

		// short-circuit sourceline so tests work
		pair.message.SourceLine = strings.TrimRight(pair.raw, "\r\n")

		if !reflect.DeepEqual(ircmsg, pair.message) {
			t.Error(
				"For", pair.raw,
				"expected", pair.message,
				"got", ircmsg,
			)
		}
	}
	for _, pair := range decodetests {
		ircmsg, err := ParseLine(pair.raw)
		if err != nil {
			t.Error(
				"For", pair.raw,
				"Failed to parse line:", err,
			)
		}

		// short-circuit sourceline so tests work
		pair.message.SourceLine = strings.TrimRight(pair.raw, "\r\n")

		if !reflect.DeepEqual(ircmsg, pair.message) {
			t.Error(
				"For", pair.raw,
				"expected", pair.message,
				"got", ircmsg,
			)
		}
	}
	for _, line := range decodetesterrors {
		_, err := ParseLine(line)
		if err == nil {
			t.Error(
				"Expected to fail parsing", line,
			)
		}
	}
}

var encodetests = []testcode{
	{":dan-!d@localhost PRIVMSG dan #test :What a cool message\r\n",
		MakeMessage(nil, "dan-!d@localhost", "PRIVMSG", "dan", "#test", "What a cool message")},
	{"@time=12732 TEST *a asda:fs :fhye tegh\r\n",
		MakeMessage(MakeTags("time", "12732"), "", "TEST", "*a", "asda:fs", "fhye tegh")},
	{"@time=12732 TEST *\r\n",
		MakeMessage(MakeTags("time", "12732"), "", "TEST", "*")},
	{"@re TEST *\r\n",
		MakeMessage(MakeTags("re", nil), "", "TEST", "*")},
}
var encodelentests = []testcodewithlen{
	{":dan-!d@lo\r\n", 12,
		MakeMessage(nil, "dan-!d@localhost", "PRIVMSG", "dan", "#test", "What a cool message")},
	{"@time=12732 TEST *\r\n", 52,
		MakeMessage(MakeTags("time", "12732"), "", "TEST", "*")},
	{"@riohwih TEST *\r\n", 8,
		MakeMessage(MakeTags("riohwihowihirgowihre", nil), "", "TEST", "*", "*")},
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
	for _, pair := range encodelentests {
		line, err := pair.message.LineMaxLen(pair.length, pair.length)
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
