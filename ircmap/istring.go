// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircmap

import "strings"

// MappingType values represent the types of IRC casemapping we support.
type MappingType int

const (
	// ASCII represents the traditional "ascii" casemapping.
	ASCII MappingType = 1 + iota

	// RFC1459 represents the casemapping defined by "rfc1459"
	RFC1459

	// RFC3454 represents the UTF-8 casefolding defined by mammon.io and
	// ircv3-harmony. Not supported yet due to no appropriate libs to do it.
	// RFC3454
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
// mapping, as defined by this package.
func Casefold(mapping MappingType, in string) string {
	var out string

	if mapping == ASCII || mapping == RFC1459 {
		// strings.ToLower ONLY replaces a-z, no unicode stuff so we're safe
		// to use that here without any issues.
		out = strings.ToLower(in)

		if mapping == RFC1459 {
			out = strings.Map(rfc1459Fold, out)
		}
	}

	return out
}
