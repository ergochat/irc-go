// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircutils

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrorNUHIsEmpty indicates that the given NUH was empty.
	ErrorNUHIsEmpty = errors.New("NUH is empty")

	// ErrorNUHContainsBadChar indicates that the NUH contained invalid characters
	ErrorNUHContainsBadChar = errors.New("NUH contains invalid characters")
)

// NUH holds a nick+username+host combination
type NUH struct {
	Nick string
	User string
	Host string
}

func ParseNUH(in string) (out NUH, err error) {
	if len(in) == 0 {
		return out, ErrorNUHIsEmpty
	}
	if strings.IndexByte(in, ' ') > -1 {
		return out, ErrorNUHContainsBadChar
	}

	hostStart := strings.IndexByte(in, '@')
	if hostStart != -1 {
		out.Host = in[hostStart+1:]
		in = in[:hostStart]
	}
	userStart := strings.IndexByte(in, '!')
	if userStart != -1 {
		out.User = in[userStart+1:]
		in = in[:userStart]
	}
	out.Nick = in

	return
}

// Canonical returns the canonical string representation of the nuh.
func (nuh *NUH) Canonical() (out string) {
	out = nuh.Nick
	if nuh.User != "" {
		out = fmt.Sprintf("%s!%s", out, nuh.User)
	}
	if nuh.Host != "" {
		out = fmt.Sprintf("%s@%s", out, nuh.Host)
	}
	return
}
