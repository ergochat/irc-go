package ircevent

import (
	"testing"
)

func TestParse(t *testing.T) {
	event := new(Event)
	event.Prefix = "nick!~user@host"

	if event.Nick() != "nick" {
		t.Fatal("Parse failed: nick")
	}
	if event.User() != "~user" {
		t.Fatal("Parse failed: user")
	}
	if event.Host() != "host" {
		t.Fatal("Parse failed: host")
	}
}

func assertEqual(found, expected string, t *testing.T) {
	if found != expected {
		t.Errorf("expected `%s`, got `%s`\n", expected, found)
	}
}

func TestUnescapeIsupport(t *testing.T) {
	assertEqual(unescapeISupportValue(""), "", t)
	assertEqual(unescapeISupportValue("a"), "a", t)
	assertEqual(unescapeISupportValue(`\x20`), " ", t)
	assertEqual(unescapeISupportValue(`\x20b`), " b", t)
	assertEqual(unescapeISupportValue(`a\x20`), "a ", t)
	assertEqual(unescapeISupportValue(`a\x20b`), "a b", t)
	assertEqual(unescapeISupportValue(`\x20\x20`), "  ", t)
}
