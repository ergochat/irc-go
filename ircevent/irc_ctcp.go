package ircevent

import (
	"fmt"
	"strings"
	"time"

	"github.com/ergochat/irc-go/ircmsg"
)

func eventRewriteCTCP(event *ircmsg.Message) {
	// XXX rewrite event.Command for CTCP
	if !(event.Command == "PRIVMSG" && len(event.Params) == 2 && strings.HasPrefix(event.Params[1], "\x01")) {
		return
	}

	msg := event.Params[1]
	event.Command = "CTCP" //Unknown CTCP

	if i := strings.LastIndex(msg, "\x01"); i > 0 {
		msg = msg[1:i]
	} else {
		return
	}

	if msg == "VERSION" {
		event.Command = "CTCP_VERSION"
	} else if msg == "TIME" {
		event.Command = "CTCP_TIME"
	} else if strings.HasPrefix(msg, "PING") {
		event.Command = "CTCP_PING"
	} else if msg == "USERINFO" {
		event.Command = "CTCP_USERINFO"
	} else if msg == "CLIENTINFO" {
		event.Command = "CTCP_CLIENTINFO"
	} else if strings.HasPrefix(msg, "ACTION") {
		event.Command = "CTCP_ACTION"
		if len(msg) > 6 {
			msg = msg[7:]
		} else {
			msg = ""
		}
	}

	event.Params[len(event.Params)-1] = msg
}

func (irc *Connection) setupCTCPCallbacks() {
	irc.AddCallback("CTCP_VERSION", func(e ircmsg.Message) {
		irc.SendRaw(fmt.Sprintf("NOTICE %s :\x01VERSION %s\x01", e.Nick(), irc.Version))
	})

	irc.AddCallback("CTCP_USERINFO", func(e ircmsg.Message) {
		irc.SendRaw(fmt.Sprintf("NOTICE %s :\x01USERINFO %s\x01", e.Nick(), irc.User))
	})

	irc.AddCallback("CTCP_CLIENTINFO", func(e ircmsg.Message) {
		irc.SendRaw(fmt.Sprintf("NOTICE %s :\x01CLIENTINFO PING VERSION TIME USERINFO CLIENTINFO\x01", e.Nick()))
	})

	irc.AddCallback("CTCP_TIME", func(e ircmsg.Message) {
		irc.SendRaw(fmt.Sprintf("NOTICE %s :\x01TIME %s\x01", e.Nick(), time.Now().UTC().Format(time.RFC1123)))
	})

	irc.AddCallback("CTCP_PING", func(e ircmsg.Message) {
		irc.SendRaw(fmt.Sprintf("NOTICE %s :\x01%s\x01", e.Nick(), e.Params[1]))
	})
}
