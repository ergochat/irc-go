// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircmsg

import (
	"errors"
	"strings"
)

// NUH represents the prefix (i.e. source) of an IRC message that originates
// from an end user (as opposed to a server numeric or notice). Such a prefix
// has three components: nickname, username, and hostname, arranged so:
// nick!user@host
type NUH struct {
	Nick string
	User string
	Host string
}

var (
	IllFormedNUH = errors.New("did not receive a well-formed nick!user@host")
)

// ParseUserhost takes a message prefix (i.e. source) and parses it into a NUH
// ("nick-user-host"). It returns an error for prefixes that are not well-formed
// NUHs (for example, server names).
func ParseNUH(rawNUH string) (result NUH, err error) {
	if i, j := strings.Index(rawNUH, "!"), strings.Index(rawNUH, "@"); i > -1 && j > -1 && i < j {
		result.Nick = rawNUH[0:i]
		result.User = rawNUH[i+1 : j]
		result.Host = rawNUH[j+1:]
		return
	}
	err = IllFormedNUH
	return
}

// String returns the canonical string representation of the NUH.
func (n *NUH) String() string {
	if n.Nick == "" {
		return ""
	}
	result := make([]byte, 0, (len(n.Nick) + 1 + len(n.User) + 1 + len(n.Host)))
	result = append(result, n.Nick...)
	result = append(result, '!')
	result = append(result, n.User...)
	result = append(result, '@')
	result = append(result, n.Host...)
	return string(result)
}
