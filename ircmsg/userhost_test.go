// Copyright (c) 2021 Shivaram Lingamneni
// Released under the MIT License

package ircmsg

import (
	"testing"
)

func TestParseNUH(t *testing.T) {
	var nuh NUH
	var err error

	nuh, err = ParseNUH("nick!user@host")
	assertEqual(nuh, NUH{"nick", "user", "host"})
	assertEqual(err, nil)
	assertEqual(nuh.String(), "nick!user@host")

	nuh, err = ParseNUH("nick!~user@host")
	assertEqual(nuh, NUH{"nick", "~user", "host"})
	assertEqual(err, nil)
	assertEqual(nuh.String(), "nick!~user@host")

	nuh, err = ParseNUH("nick!@host")
	assertEqual(nuh, NUH{"nick", "", "host"})
	assertEqual(err, nil)
	assertEqual(nuh.String(), "nick!@host")

	nuh, err = ParseNUH("nick!@")
	assertEqual(nuh, NUH{"nick", "", ""})
	assertEqual(err, nil)
	assertEqual(nuh.String(), "nick!@")

	// bare nick (ambiguous with server name)
	nuh, err = ParseNUH("nick")
	assertEqual(nuh, NUH{})
	assertEqual(err, IllFormedNUH)

	// server name (ambiguous with bare nick, under bad implementations;
	// see discussion on #57, #58):
	nuh, err = ParseNUH("testnet.ergo.chat")
	assertEqual(nuh, NUH{})
	assertEqual(err, IllFormedNUH)

	// these are erroneous forms and not attested
	for _, nuhStr := range []string{"nick@user!host", "nick!user", "nick@host"} {
		nuh, err = ParseNUH(nuhStr)
		assertEqual(nuh, NUH{})
		assertEqual(err, IllFormedNUH)
	}
}
