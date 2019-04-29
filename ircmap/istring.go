// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircmap

import (
	"errors"
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

// ChannelPrefixes are the allowed prefixes for channels, used in casefolding.
var ChannelPrefixes = map[byte]bool{
	// standard, well-used
	'#': true,
	'&': true,
	// standard, not well-used
	'!': true,
	'+': true,
	// znc uses for partylines
	'~': true,
}

// rfc1459Fold casefolds only the special chars defined by RFC1459 -- the
// others are handled by the strings.ToLower earlier.
func rfc1459Fold(r rune) rune {
	if '[' <= r && r <= ']' {
		r += '{' - '['
	}
	return r
}

var (
	// ErrCouldNotStabilize indicates that we could not stabilize the input string.
	ErrCouldNotStabilize = errors.New("Could not stabilize string while casefolding")
)

// Each pass of PRECIS casefolding is a composition of idempotent operations,
// but not idempotent itself. Therefore, the spec says "do it four times and hope
// it converges" (lolwtf). Golang's PRECIS implementation has a "repeat" option,
// which provides this functionality, but unfortunately it's not exposed publicly.
func iterateFolding(profile *precis.Profile, oldStr string) (str string, err error) {
	str = oldStr
	// follow the stabilizing rules laid out here:
	// https://tools.ietf.org/html/draft-ietf-precis-7564bis-10.html#section-7
	for i := 0; i < 4; i++ {
		str, err = profile.CompareKey(str)
		if err != nil {
			return "", err
		}
		if oldStr == str {
			break
		}
		oldStr = str
	}
	if oldStr != str {
		return "", ErrCouldNotStabilize
	}
	return str, nil
}

// PrecisCasefold returns a casefolded string, without doing any name or channel character checks.
func PrecisCasefold(str string) (string, error) {
	return iterateFolding(precis.UsernameCaseMapped, str)
}

// Casefold returns a string, lowercased/casefolded according to the given
// mapping as defined by this package (or an error if the given string is not
// valid in the chosen mapping).
func Casefold(mapping MappingType, input string) (string, error) {
	return CasefoldCustomChannelPrefixes(mapping, input, ChannelPrefixes)
}

// CasefoldCustomChannelPrefixes returns a string, lowercased/casefolded
// according to the given mapping as defined by this package (or an error if
// the given string is not valid in the chosen mapping), using a custom
// channel prefix map.
func CasefoldCustomChannelPrefixes(mapping MappingType, input string, channelPrefixes map[byte]bool) (string, error) {
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
		// skip channel prefixes to avoid bidi rule (as per spec)
		var start int
		for start = 0; start < len(input) && channelPrefixes[input[start]]; start++ {
		}

		lowered, err := PrecisCasefold(input[start:])
		if err != nil {
			return "", err
		}

		return input[:start] + lowered, err
	}

	return out, err
}
