package girc

import (
	"fmt"
	"reflect"
	"testing"
)

type testdecode struct {
	raw     string
	message IrcMessage
}

var tests = []testdecode{
	{":dan-!d@localhost PRIVMSG dan #test :What a cool message",
		MakeMessage(nil, "dan-!d@localhost", "PRIVMSG", "dan", "#test", "What a cool message")},
}

func TestDecode(t *testing.T) {
	for _, pair := range tests {
		ircmsg, err := ParseLine(pair.raw)
		if err != nil {
			fmt.Println("FAILED TO PARSE LINE:", pair.raw)
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
