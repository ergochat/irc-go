// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircmsg

import (
	"errors"
	"fmt"
	"strings"
)

// NUH holds a parsed nick!user@host source of an IRC message
type NUH struct {
	Nick string
	User string
	Host string
}

var (
	IllFormedNUH = errors.New("did not receive a well-formed nick!user@host")
)

// ParseUserhost takes a userhost string and returns a UserHost instance.
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

// String returns the canonical string representation of the userhost.
func (n *NUH) String() string {
	if n.Nick == "" {
		return ""
	}
	return fmt.Sprintf("%s!%s@%s", n.Nick, n.User, n.Host)
}
