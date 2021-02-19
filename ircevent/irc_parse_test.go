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
