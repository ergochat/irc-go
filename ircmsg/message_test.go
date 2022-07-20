package ircmsg

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

type testcode struct {
	raw     string
	message Message
}
type testcodewithlen struct {
	raw              string
	length           int
	message          Message
	truncateExpected bool
}

var decodelentests = []testcodewithlen{
	{":dan-!d@localhost PRIVMSG dan #test :What a cool message\r\n", 22,
		MakeMessage(nil, "dan-!d@localhost", "PR"), true},
	{"@time=12732;re TEST *\r\n", 512,
		MakeMessage(map[string]string{"time": "12732", "re": ""}, "", "TEST", "*"), false},
	{"@time=12732;re TEST *\r\n", 512,
		MakeMessage(map[string]string{"time": "12732", "re": ""}, "", "TEST", "*"), false},
	{":dan- TESTMSG\r\n", 2048,
		MakeMessage(nil, "dan-", "TESTMSG"), false},
	{":dan- TESTMSG dan \r\n", 14,
		MakeMessage(nil, "dan-", "TESTMS"), true},
	{"TESTMSG\r\n", 6,
		MakeMessage(nil, "", "TEST"), true},
	{"TESTMSG\r\n", 7,
		MakeMessage(nil, "", "TESTM"), true},
	{"TESTMSG\r\n", 8,
		MakeMessage(nil, "", "TESTMS"), true},
	{"TESTMSG\r\n", 9,
		MakeMessage(nil, "", "TESTMSG"), false},
}

// map[string]string{"time": "12732", "re": ""}
var decodetests = []testcode{
	{":dan-!d@localhost PRIVMSG dan #test :What a cool message\r\n",
		MakeMessage(nil, "dan-!d@localhost", "PRIVMSG", "dan", "#test", "What a cool message")},
	{"@time=2848 :dan-!d@localhost LIST\r\n",
		MakeMessage(map[string]string{"time": "2848"}, "dan-!d@localhost", "LIST")},
	{"@time=2848 LIST\r\n",
		MakeMessage(map[string]string{"time": "2848"}, "", "LIST")},
	{"LIST\r\n",
		MakeMessage(nil, "", "LIST")},
	{"@time=12732;re TEST *a asda:fs :fhye tegh\r\n",
		MakeMessage(map[string]string{"time": "12732", "re": ""}, "", "TEST", "*a", "asda:fs", "fhye tegh")},
	{"@time=12732;re TEST *\r\n",
		MakeMessage(map[string]string{"time": "12732", "re": ""}, "", "TEST", "*")},
	{":dan- TESTMSG\r\n",
		MakeMessage(nil, "dan-", "TESTMSG")},
	{":dan- TESTMSG dan \r\n",
		MakeMessage(nil, "dan-", "TESTMSG", "dan")},
	{"@time=2019-02-28T19:30:01.727Z ping HiThere!\r\n",
		MakeMessage(map[string]string{"time": "2019-02-28T19:30:01.727Z"}, "", "PING", "HiThere!")},
	{"@+draft/test=hi\\nthere PING HiThere!\r\n",
		MakeMessage(map[string]string{"+draft/test": "hi\nthere"}, "", "PING", "HiThere!")},
	{"ping asdf\n",
		MakeMessage(nil, "", "PING", "asdf")},
	{"JoIN  #channel\n",
		MakeMessage(nil, "", "JOIN", "#channel")},
	{"@draft/label=l  join   #channel\n",
		MakeMessage(map[string]string{"draft/label": "l"}, "", "JOIN", "#channel")},
	{"list",
		MakeMessage(nil, "", "LIST")},
	{"list ",
		MakeMessage(nil, "", "LIST")},
	{"list  ",
		MakeMessage(nil, "", "LIST")},
	{"@time=2848  :dan-!d@localhost  LIST \r\n",
		MakeMessage(map[string]string{"time": "2848"}, "dan-!d@localhost", "LIST")},
	{"@time=2848 :dan-!d@localhost PRIVMSG a:b :\r\n",
		MakeMessage(map[string]string{"time": "2848"}, "dan-!d@localhost", "PRIVMSG", "a:b", "")},
	{"@time=2848 :dan-!d@localhost PRIVMSG a:b ::\r\n",
		MakeMessage(map[string]string{"time": "2848"}, "dan-!d@localhost", "PRIVMSG", "a:b", ":")},
	{"@time=2848 :dan-!d@localhost PRIVMSG a:b ::hi\r\n",
		MakeMessage(map[string]string{"time": "2848"}, "dan-!d@localhost", "PRIVMSG", "a:b", ":hi")},
	{"@time=2848 :dan-!d@localhost PRIVMSG a:b :hi\r\n",
		MakeMessage(map[string]string{"time": "2848"}, "dan-!d@localhost", "PRIVMSG", "a:b", "hi")},
	// invalid UTF8:
	{"@time=2848 :dan-!d@localhost PRIVMSG a:b :hi\xf0\xf0\r\n",
		MakeMessage(map[string]string{"time": "2848"}, "dan-!d@localhost", "PRIVMSG", "a:b", "hi\xf0\xf0")},
	{"@time=2848 :dan-!d@localhost PRIVMSG a:b :\xf0hi\xf0\r\n",
		MakeMessage(map[string]string{"time": "2848"}, "dan-!d@localhost", "PRIVMSG", "a:b", "\xf0hi\xf0")},
	{"@time=2848 :dan-!d@localhost PRIVMSG a:b :\xff\r\n",
		MakeMessage(map[string]string{"time": "2848"}, "dan-!d@localhost", "PRIVMSG", "a:b", "\xff")},
	{"@time=2848 :dan-!d@localhost PRIVMSG a:b :\xf9g\xa6=\xcf6s\xb2\xe2\xaf\xa0kSN?\x95\r\n",
		MakeMessage(map[string]string{"time": "2848"}, "dan-!d@localhost", "PRIVMSG", "a:b", "\xf9g\xa6=\xcf6s\xb2\xe2\xaf\xa0kSN?\x95")},
	{"@time=2848 :dan-!d@localhost PRIVMSG a:b \xf9g\xa6=\xcf6s\xb2\xe2\xaf\xa0kSN?\x95\r\n",
		MakeMessage(map[string]string{"time": "2848"}, "dan-!d@localhost", "PRIVMSG", "a:b", "\xf9g\xa6=\xcf6s\xb2\xe2\xaf\xa0kSN?\x95")},
}

type testparseerror struct {
	raw string
	err error
}

var decodetesterrors = []testparseerror{
	{"", ErrorLineIsEmpty},
	{"\r\n", ErrorLineIsEmpty},
	{"\r\n    ", ErrorLineContainsBadChar},
	{"\r\n ", ErrorLineContainsBadChar},
	{" \r\n", ErrorLineIsEmpty},
	{" \r\n ", ErrorLineContainsBadChar},
	{"     \r\n  ", ErrorLineContainsBadChar},
	{"@tags=tesa\r\n", ErrorLineIsEmpty},
	{"@tags=tested  \r\n", ErrorLineIsEmpty},
	{":dan-   \r\n", ErrorLineIsEmpty},
	{":dan-\r\n", ErrorLineIsEmpty},
	{"@tag1=1;tag2=2 :dan \r\n", ErrorLineIsEmpty},
	{"@tag1=1;tag2=2 :dan      \r\n", ErrorLineIsEmpty},
	{"@tag1=1;tag2=2\x00 :dan      \r\n", ErrorLineContainsBadChar},
	{"@tag1=1;tag2=2\x00 :shivaram PRIVMSG #channel  hi\r\n", ErrorLineContainsBadChar},
	{"privmsg #channel :command injection attempt \n:Nickserv PRIVMSG user :Please re-enter your password", ErrorLineContainsBadChar},
	{"privmsg #channel :command injection attempt \r:Nickserv PRIVMSG user :Please re-enter your password", ErrorLineContainsBadChar},
}

func validateTruncateError(pair testcodewithlen, err error, t *testing.T) {
	if pair.truncateExpected {
		if err != ErrorBodyTooLong {
			t.Error("For", pair.raw, "expected truncation, but got error", err)
		}
	} else {
		if err != nil {
			t.Error("For", pair.raw, "expected no error, but got", err)
		}
	}
}

func TestDecode(t *testing.T) {
	for _, pair := range decodelentests {
		ircmsg, err := ParseLineStrict(pair.raw, true, pair.length)
		validateTruncateError(pair, err, t)

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

		if !reflect.DeepEqual(ircmsg, pair.message) {
			t.Error(
				"For", pair.raw,
				"expected", pair.message,
				"got", ircmsg,
			)
		}
	}
	for _, pair := range decodetesterrors {
		_, err := ParseLineStrict(pair.raw, true, 0)
		if err != pair.err {
			t.Error(
				"For", pair.raw,
				"expected", pair.err,
				"got", err,
			)
		}
	}
}

var encodetests = []testcode{
	{":dan-!d@localhost PRIVMSG dan #test :What a cool message\r\n",
		MakeMessage(nil, "dan-!d@localhost", "PRIVMSG", "dan", "#test", "What a cool message")},
	{"@time=12732 TEST *a asda:fs :fhye tegh\r\n",
		MakeMessage(map[string]string{"time": "12732"}, "", "TEST", "*a", "asda:fs", "fhye tegh")},
	{"@time=12732 TEST *\r\n",
		MakeMessage(map[string]string{"time": "12732"}, "", "TEST", "*")},
	{"@re TEST *\r\n",
		MakeMessage(map[string]string{"re": ""}, "", "TEST", "*")},
}
var encodelentests = []testcodewithlen{
	{":dan-!d@lo\r\n", 12,
		MakeMessage(nil, "dan-!d@localhost", "PRIVMSG", "dan", "#test", "What a cool message"), true},
	{"@time=12732 TEST *\r\n", 52,
		MakeMessage(map[string]string{"time": "12732"}, "", "TEST", "*"), false},
	{"@riohwihowihirgowihre TEST *\r\n", 8,
		MakeMessage(map[string]string{"riohwihowihirgowihre": ""}, "", "TEST", "*", "*"), true},
}

func TestEncode(t *testing.T) {
	for _, pair := range encodetests {
		line, err := pair.message.LineBytes()
		if err != nil {
			t.Error(
				"For", pair.raw,
				"Failed to parse line:", err,
			)
		}

		if string(line) != pair.raw {
			t.Error(
				"For LineBytes of", pair.message,
				"expected", pair.raw,
				"got", line,
			)
		}
	}
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
		line, err := pair.message.LineBytesStrict(true, pair.length)
		validateTruncateError(pair, err, t)

		if string(line) != pair.raw {
			t.Error(
				"For", pair.message,
				"expected", pair.raw,
				"got", line,
			)
		}
	}
}

var encodeErrorTests = []struct {
	tags    map[string]string
	prefix  string
	command string
	params  []string
	err     error
}{
	{tags: nil, command: "PRIVMSG", params: []string{"", "hi"}, err: ErrorBadParam},
	{tags: nil, command: "KICK", params: []string{":nick", "message"}, err: ErrorBadParam},
	{tags: nil, command: "QUX", params: []string{"#baz", ":bat", "bar"}, err: ErrorBadParam},
	{tags: nil, command: "", params: []string{"hi"}, err: ErrorCommandMissing},
	{tags: map[string]string{"a\x00b": "hi"}, command: "PING", params: []string{"hi"}, err: ErrorInvalidTagContent},
	{tags: map[string]string{"ab": "h\x00i"}, command: "PING", params: []string{"hi"}, err: ErrorLineContainsBadChar},
	{tags: map[string]string{"ab": "\xff\xff"}, command: "PING", params: []string{"hi"}, err: ErrorInvalidTagContent},
	{tags: map[string]string{"ab": "hi"}, command: "PING", params: []string{"h\x00i"}, err: ErrorLineContainsBadChar},
	{tags: map[string]string{"ab": "hi"}, command: "PING", params: []string{"h\ni"}, err: ErrorLineContainsBadChar},
	{tags: map[string]string{"ab": "hi"}, command: "PING", params: []string{"hi\rQUIT"}, err: ErrorLineContainsBadChar},
	{tags: map[string]string{"ab": "hi"}, command: "NOTICE", params: []string{"#channel", "hi\r\nQUIT"}, err: ErrorLineContainsBadChar},
}

func TestEncodeErrors(t *testing.T) {
	for _, ep := range encodeErrorTests {
		msg := MakeMessage(ep.tags, ep.prefix, ep.command, ep.params...)
		_, err := msg.LineBytesStrict(true, 512)
		if err != ep.err {
			t.Errorf("For %#v, expected %v, got %v", msg, ep.err, err)
		}
	}
}

var testMessages = []Message{
	{
		tags:           map[string]string{"time": "2019-02-27T04:38:57.489Z", "account": "dan-"},
		clientOnlyTags: map[string]string{"+status": "typing"},
		Source:         "dan-!~user@example.com",
		Command:        "TAGMSG",
	},
	{
		clientOnlyTags: map[string]string{"+status": "typing"},
		Command:        "PING", // invalid PING command but we don't care
	},
	{
		tags:    map[string]string{"time": "2019-02-27T04:38:57.489Z"},
		Command: "PING", // invalid PING command but we don't care
		Params:  []string{"12345"},
	},
	{
		tags:    map[string]string{"time": "2019-02-27T04:38:57.489Z", "account": "dan-"},
		Source:  "dan-!~user@example.com",
		Command: "PRIVMSG",
		Params:  []string{"#ircv3", ":smiley:"},
	},
	{
		tags:    map[string]string{"time": "2019-02-27T04:38:57.489Z", "account": "dan-"},
		Source:  "dan-!~user@example.com",
		Command: "PRIVMSG",
		Params:  []string{"#ircv3", "\x01ACTION writes some specs!\x01"},
	},
	{
		Source:  "dan-!~user@example.com",
		Command: "PRIVMSG",
		Params:  []string{"#ircv3", ": long trailing command with langue française in it"},
	},
	{
		Source:  "dan-!~user@example.com",
		Command: "PRIVMSG",
		Params:  []string{"#ircv3", " : long trailing command with langue française in it "},
	},
	{
		Source:  "shivaram",
		Command: "KLINE",
		Params:  []string{"ANDKILL", "24h", "tkadich", "your", "client", "is", "disconnecting", "too", "much"},
	},
	{
		tags:    map[string]string{"time": "2019-02-27T06:01:23.545Z", "draft/msgid": "xjmgr6e4ih7izqu6ehmrtrzscy"},
		Source:  "שיברם",
		Command: "PRIVMSG",
		Params:  []string{"ויקם מלך חדש על מצרים אשר לא ידע את יוסף"},
	},
	{
		Source:  "shivaram!~user@2001:0db8::1",
		Command: "KICK",
		Params:  []string{"#darwin", "devilbat", ":::::::::::::: :::::::::::::"},
	},
}

func TestEncodeDecode(t *testing.T) {
	for _, message := range testMessages {
		encoded, err := message.LineBytesStrict(false, 0)
		if err != nil {
			t.Errorf("Couldn't encode %v: %v", message, err)
		}
		parsed, err := ParseLineStrict(string(encoded), true, 0)
		if err != nil {
			t.Errorf("Couldn't re-decode %v: %v", encoded, err)
		}
		if !reflect.DeepEqual(message, parsed) {
			t.Errorf("After encoding and re-parsing, got different messages:\n%v\n%v", message, parsed)
		}
	}
}

func TestForceTrailing(t *testing.T) {
	message := Message{
		Source:  "shivaram",
		Command: "PRIVMSG",
		Params:  []string{"#darwin", "nice"},
	}
	bytes, err := message.LineBytesStrict(true, 0)
	if err != nil {
		t.Error(err)
	}
	if string(bytes) != ":shivaram PRIVMSG #darwin nice\r\n" {
		t.Errorf("unexpected serialization: %s", bytes)
	}
	message.ForceTrailing()
	bytes, err = message.LineBytesStrict(true, 0)
	if err != nil {
		t.Error(err)
	}
	if string(bytes) != ":shivaram PRIVMSG #darwin :nice\r\n" {
		t.Errorf("unexpected serialization: %s", bytes)
	}
}

func TestErrorLineTooLongGeneration(t *testing.T) {
	message := Message{
		tags:    map[string]string{"draft/msgid": "SAXV5OYJUr18CNJzdWa1qQ"},
		Source:  "shivaram",
		Command: "PRIVMSG",
		Params:  []string{"aaaaaaaaaaaaaaaaaaaaa"},
	}
	_, err := message.LineBytesStrict(true, 0)
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 100; i += 1 {
		message.SetTag(fmt.Sprintf("+client-tag-%d", i), "ok")
	}
	line, err := message.LineBytesStrict(true, 0)
	if err != nil {
		t.Error(err)
	}
	if 4096 < len(line) {
		t.Errorf("line is too long: %d", len(line))
	}

	// add excess tag data, pushing us over the limit
	for i := 100; i < 500; i += 1 {
		message.SetTag(fmt.Sprintf("+client-tag-%d", i), "ok")
	}
	line, err = message.LineBytesStrict(true, 0)
	if err != ErrorTagsTooLong {
		t.Error(err)
	}

	message.clientOnlyTags = nil
	for i := 0; i < 500; i += 1 {
		message.SetTag(fmt.Sprintf("server-tag-%d", i), "ok")
	}
	line, err = message.LineBytesStrict(true, 0)
	if err != ErrorTagsTooLong {
		t.Error(err)
	}

	message.tags = nil
	message.clientOnlyTags = nil
	for i := 0; i < 200; i += 1 {
		message.SetTag(fmt.Sprintf("server-tag-%d", i), "ok")
		message.SetTag(fmt.Sprintf("+client-tag-%d", i), "ok")
	}
	// client cannot send this much tag data:
	line, err = message.LineBytesStrict(true, 0)
	if err != ErrorTagsTooLong {
		t.Error(err)
	}
	// but a server can, since the tags are split between client and server budgets:
	line, err = message.LineBytesStrict(false, 0)
	if err != nil {
		t.Error(err)
	}
}

var truncateTests = []string{
	"x", // U+0078, Latin Small Letter X, 1 byte
	"ç", // U+00E7, Latin Small Letter C with Cedilla, 2 bytes
	"ꙮ", // U+A66E, Cyrillic Letter Multiocular O, 3 bytes
	"🐬", // U+1F42C, Dolphin, 4 bytes
}

func assertEqual(found, expected interface{}) {
	if !reflect.DeepEqual(found, expected) {
		panic(fmt.Sprintf("expected %#v, found %#v", expected, found))
	}
}

func buildPingParam(initialLen, minLen int, encChar string) (result string) {
	var out strings.Builder
	for i := 0; i < initialLen; i++ {
		out.WriteByte('a')
	}
	for out.Len() <= minLen {
		out.WriteString(encChar)
	}
	return out.String()
}

func min(i, j int) int {
	if i < j {
		return i
	}
	return j
}

func TestTruncate(t *testing.T) {
	// OK, this test is weird: we're going to build a line with a final parameter
	// that consists of a bunch of a's, then some nonzero number of repetitions
	// of a different UTF8-encoded codepoint. we'll test all 4 possible lengths
	// for a codepoint, and a number of different alignments for the codepoint
	// relative to the 512-byte boundary. in all cases, we should produce valid
	// UTF8, and truncate at most 3 bytes below the 512-byte boundary.
	for idx, s := range truncateTests {
		// sanity check that we have the expected lengths:
		assertEqual(len(s), idx+1)
		r, _ := utf8.DecodeRuneInString(s)
		if r == utf8.RuneError {
			panic("invalid codepoint in test suite")
		}

		// "PING [param]\r\n", max parameter size is 512-7=505 bytes
		for initialLen := 490; initialLen < 500; initialLen++ {
			for i := 1; i < 50; i++ {
				param := buildPingParam(initialLen, initialLen+i, s)
				msg := MakeMessage(nil, "", "PING", param)
				msgBytes, err := msg.LineBytesStrict(false, 512)
				msgBytesNonTrunc, _ := msg.LineBytes()
				if len(msgBytes) == len(msgBytesNonTrunc) {
					if err != nil {
						t.Error("message was not truncated, but got error", err)
					}
				} else {
					if err != ErrorBodyTooLong {
						t.Error("message was truncated, but got error", err)
					}
				}
				if len(msgBytes) > 512 {
					t.Errorf("invalid serialized length %d", len(msgBytes))
				}
				if len(msgBytes) < min(512-3, len(msgBytesNonTrunc)) {
					t.Errorf("invalid serialized length %d", len(msgBytes))
				}
				if !utf8.Valid(msgBytes) {
					t.Errorf("PING %s encoded to invalid UTF8: %#v\n", param, msgBytes)
				}
				// skip over "PING "
				first, _ := utf8.DecodeRune(msgBytes[5:])
				assertEqual(first, rune('a'))
				last, _ := utf8.DecodeLastRune(bytes.TrimSuffix(msgBytes, []byte("\r\n")))
				assertEqual(last, r)
			}
		}
	}
}

func TestTruncateNonUTF8(t *testing.T) {
	for l := 490; l < 530; l++ {
		var buf strings.Builder
		for i := 0; i < l; i++ {
			buf.WriteByte('\xff')
		}
		param := buf.String()
		msg := MakeMessage(nil, "", "PING", param)
		msgBytes, err := msg.LineBytesStrict(false, 512)
		if !(err == nil || err == ErrorBodyTooLong) {
			panic(err)
		}
		if len(msgBytes) > 512 {
			t.Errorf("invalid serialized length %d", len(msgBytes))
		}
		// full length is "PING <param>\r\n", 7+len(param)
		if len(msgBytes) < min(512-3, 7+len(param)) {
			t.Errorf("invalid serialized length %d", len(msgBytes))
		}
	}
}

func BenchmarkGenerate(b *testing.B) {
	msg := MakeMessage(
		map[string]string{"time": "2019-02-28T08:12:43.480Z", "account": "shivaram"},
		"shivaram_hexchat!~user@irc.darwin.network",
		"PRIVMSG",
		"#darwin", "what's up guys",
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg.LineBytesStrict(false, 0)
	}
}

func BenchmarkParse(b *testing.B) {
	line := "@account=shivaram;draft/msgid=dqhkgglocqikjqikbkcdnv5dsq;time=2019-03-01T20:11:21.833Z :shivaram!~shivaram@good-fortune PRIVMSG #darwin :you're an EU citizen, right? it's illegal for you to be here now"
	for i := 0; i < b.N; i++ {
		ParseLineStrict(line, false, 0)
	}
}
