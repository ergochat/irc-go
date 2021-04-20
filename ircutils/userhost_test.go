// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircutils

import (
	"fmt"
	"testing"

	"gopkg.in/yaml.v2"
)

const nuhTests = `
# IRC parser tests
# splitting userhosts into atoms

# Written in 2015 by Daniel Oaks <daniel@danieloaks.net>
#
# To the extent possible under law, the author(s) have dedicated all copyright
# and related and neighboring rights to this software to the public domain
# worldwide. This software is distributed without any warranty.
#
# You should have received a copy of the CC0 Public Domain Dedication along
# with this software. If not, see
# <http://creativecommons.org/publicdomain/zero/1.0/>.

tests:
  # source is the usthost

  # the atoms dict has the keys:
  #   * nick: nick string
  #   * user: user string
  #   * host: host string
  # if a key does not exist, assume it is an empty string

  # simpler
  - source: "coolguy"
    atoms:
      nick: "coolguy"

  # simple
  - source: "coolguy!ag@127.0.0.1"
    atoms:
      nick: "coolguy"
      user: "ag"
      host: "127.0.0.1"

  - source: "coolguy!~ag@localhost"
    atoms:
      nick: "coolguy"
      user: "~ag"
      host: "localhost"

  # without atoms
  - source: "!ag@127.0.0.1"
    atoms:
      user: "ag"
      host: "127.0.0.1"

  - source: "coolguy!@127.0.0.1"
    atoms:
      nick: "coolguy"
      host: "127.0.0.1"

  - source: "coolguy@127.0.0.1"
    atoms:
      nick: "coolguy"
      host: "127.0.0.1"

  - source: "coolguy!ag@"
    atoms:
      nick: "coolguy"
      user: "ag"

  - source: "coolguy!ag"
    atoms:
      nick: "coolguy"
      user: "ag"

  # weird control codes, does happen
  - source: "coolguy!ag@net\x035w\x03ork.admin"
    atoms:
      nick: "coolguy"
      user: "ag"
      host: "net\x035w\x03ork.admin"

  - source: "coolguy!~ag@n\x02et\x0305w\x0fork.admin"
    atoms:
      nick: "coolguy"
      user: "~ag"
      host: "n\x02et\x0305w\x0fork.admin"
`

type NUHSplitTest struct {
	Source string
	Atoms  struct {
		Nick string
		User string
		Host string
	}
}

type TestFile struct {
	Tests []NUHSplitTest
}

func assertEqualNUH(found, expected NUH) {
	if found.Nick != expected.Nick || found.User != expected.User || found.Host != expected.Host {
		panic(fmt.Sprintf("expected %#v, found %#v", expected.Canonical(), found.Canonical()))
	}
}
func TestSplittingNUH(t *testing.T) {
	var tF *TestFile
	err := yaml.Unmarshal([]byte(nuhTests), &tF)
	if err != nil {
		t.Errorf("could not load nuh splitting test file [%q]", err.Error())
	}

	for _, test := range tF.Tests {
		out, err := ParseNUH(test.Source)
		if err != nil {
			t.Errorf("could not parse nuh test [%s] got [%s]", test.Source, err.Error())
		}
		comp := NUH{
			Nick: test.Atoms.Nick,
			User: test.Atoms.User,
			Host: test.Atoms.Host,
		}
		assertEqualNUH(out, comp)
	}
}
