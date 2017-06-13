// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircmap

import (
	"strings"

	"golang.org/x/text/secure/precis"

	"github.com/DanielOaks/go-idn/idna2003/stringprep"
)

// MappingType values represent the types of IRC casemapping we support.
type MappingType int

const (
	// NONE represents no casemapping.
	NONE MappingType = 0 + iota

	// ASCII represents the traditional "ascii" casemapping.
	ASCII

	// RFC1459 represents the casemapping defined by "rfc1459"
	RFC1459

	// RFC3454 represents the UTF-8 nameprep casefolding as used by mammon-ircd.
	RFC3454

	// RFC7613 represents the UTF-8 casefolding currently being drafted by me
	// with the IRCv3 WG.
	RFC7613
)

var (
	// Mappings is a mapping of ISUPPORT CASEMAP strings to our MappingTypes.
	Mappings = map[string]MappingType{
		"ascii":   ASCII,
		"rfc1459": RFC1459,
		"rfc3454": RFC3454,
		"rfc7613": RFC7613,
	}
)

// rfc1459Fold casefolds only the special chars defined by RFC1459 -- the
// others are handled by the strings.ToLower earlier.
func rfc1459Fold(r rune) rune {
	if '[' <= r && r <= ']' {
		r += '{' - '['
	}
	return r
}

// Casefold returns a string, lowercased/casefolded according to the given
// mapping as defined by this package (or an error if the given string is not
// valid in the chosen mapping).
func Casefold(mapping MappingType, input string) (string, error) {
	var out string
	var err error

	if mapping == ASCII || mapping == RFC1459 {
		// strings.ToLower ONLY replaces a-z, no unicode stuff so we're safe
		// to use that here without any issues.
		out = strings.ToLower(input)

		if mapping == RFC1459 {
			out = strings.Map(rfc1459Fold, out)
		}
	} else if mapping == RFC3454 {
		out, err = stringprep.Nameprep(input)
	} else if mapping == RFC7613 {
		out, err = precis.UsernameCaseMapped.CompareKey(input)
	}

	return out, err
}
