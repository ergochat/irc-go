// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package gircclient

import (
	"github.com/goshuirc/irc-go/ircfmt"
)

// Msg sends a message to the given target.
func (sc *ServerConnection) Msg(tags map[string]string, target string, message string, escaped bool) {
	if escaped {
		message = ircfmt.Unescape(message)
	}
	sc.Send(tags, "", "PRIVMSG", target, message)
}

// Notice sends a notice to the given target.
func (sc *ServerConnection) Notice(tags map[string]string, target string, message string, escaped bool) {
	if escaped {
		message = ircfmt.Unescape(message)
	}
	sc.Send(tags, "", "NOTICE", target, message)
}
