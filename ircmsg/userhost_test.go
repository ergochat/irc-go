// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircmsg

import (
	"fmt"
	"testing"
)

type nuhSplitTest struct {
	NUH
	Source    string
	Canonical bool
}

var nuhTests = []nuhSplitTest{
	{
		Source:    "coolguy",
		NUH:       NUH{"coolguy", "", ""},
		Canonical: true,
	},
	{
		Source:    "coolguy!ag@127.0.0.1",
		NUH:       NUH{"coolguy", "ag", "127.0.0.1"},
		Canonical: true,
	},
	{
		Source:    "coolguy!~ag@localhost",
		NUH:       NUH{"coolguy", "~ag", "localhost"},
		Canonical: true,
	},
	// missing components:
	{
		Source:    "!ag@127.0.0.1",
		NUH:       NUH{"", "ag", "127.0.0.1"},
		Canonical: true,
	},
	{
		Source: "coolguy!@127.0.0.1",
		NUH:    NUH{"coolguy", "", "127.0.0.1"},
	},
	{
		Source:    "coolguy@127.0.0.1",
		NUH:       NUH{"coolguy", "", "127.0.0.1"},
		Canonical: true,
	},
	{
		Source: "coolguy!ag@",
		NUH:    NUH{"coolguy", "ag", ""},
	},
	{
		Source:    "coolguy!ag",
		NUH:       NUH{"coolguy", "ag", ""},
		Canonical: true,
	},
	// resilient to weird characters:
	{
		Source:    "coolguy!ag@net\x035w\x03ork.admin",
		NUH:       NUH{"coolguy", "ag", "net\x035w\x03ork.admin"},
		Canonical: true,
	},
	{
		Source:    "coolguy!~ag@n\x02et\x0305w\x0fork.admin",
		NUH:       NUH{"coolguy", "~ag", "n\x02et\x0305w\x0fork.admin"},
		Canonical: true,
	},
	{
		Source:    "testnet.ergo.chat",
		NUH:       NUH{"testnet.ergo.chat", "", ""},
		Canonical: true,
	},
}

func assertEqualNUH(found, expected NUH) {
	if found.Name != expected.Name || found.User != expected.User || found.Host != expected.Host {
		panic(fmt.Sprintf("expected %#v, found %#v", expected.Canonical(), found.Canonical()))
	}
}

func TestSplittingNUH(t *testing.T) {
	for _, test := range nuhTests {
		out, err := ParseNUH(test.Source)
		if err != nil {
			t.Errorf("could not parse nuh test [%s] got [%s]", test.Source, err.Error())
		}
		assertEqualNUH(out, test.NUH)
		canonical := out.Canonical()
		if test.Canonical {
			assertEqual(canonical, test.Source)
		}
	}
}
