package ircevent

import (
	"fmt"
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	source := "nick!~user@host"
	nick, user, host := SplitNUH(source)

	if nick != "nick" {
		t.Fatal("Parse failed: nick")
	}
	if user != "~user" {
		t.Fatal("Parse failed: user")
	}
	if host != "host" {
		t.Fatal("Parse failed: host")
	}
}

func assertEqual(found, expected interface{}) {
	if !reflect.DeepEqual(found, expected) {
		panic(fmt.Sprintf("expected `%#v`, got `%#v`\n", expected, found))
	}
}

func TestUnescapeIsupport(t *testing.T) {
	assertEqual(unescapeISupportValue(""), "")
	assertEqual(unescapeISupportValue("a"), "a")
	assertEqual(unescapeISupportValue(`\x20`), " ")
	assertEqual(unescapeISupportValue(`\x20b`), " b")
	assertEqual(unescapeISupportValue(`a\x20`), "a ")
	assertEqual(unescapeISupportValue(`a\x20b`), "a b")
	assertEqual(unescapeISupportValue(`\x20\x20`), "  ")
}
