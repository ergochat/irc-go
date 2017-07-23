// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package gircclient

import (
	"fmt"
	"strings"

	"github.com/goshuirc/eventmgr"
	"github.com/goshuirc/irc-go/ircmap"
)

// welcomeHandler sets the nick to the first parameter of the 001 message.
// This ensures that when we connect to IRCds that silently truncate the
// nickname, we keep the correct one.
func welcomeHandler(event string, info eventmgr.InfoMap) {
	sc := info["server"].(*ServerConnection)
	sc.Nick = info["params"].([]string)[0]

	sc.Registered = true

	// join channels if we have any to join
	for _, channel := range sc.channelsToJoin {
		params := []string{channel.Name}
		if channel.UseKey {
			params = []string{channel.Name, channel.Key}
		}
		sc.Send(nil, "", "JOIN", params...)
	}
	sc.channelsToJoin = []channel{} // empty array
}

func featuresHandler(event string, info eventmgr.InfoMap) {
	sc := info["server"].(*ServerConnection)

	// parse features into our internal list
	tags := info["params"].([]string)
	tags = tags[1 : len(tags)-1] // remove first and last params
	sc.Features.Parse(tags...)

	if sc.Casemapping == ircmap.NONE {
		name, exists := sc.Features["CASEMAPPING"]
		if exists {
			sc.Casemapping = ircmap.Mappings[name.(string)]
		}
	}
}

func capHandler(event string, info eventmgr.InfoMap) {
	sc := info["server"].(*ServerConnection)
	params := info["params"].([]string)
	subcommand := strings.ToUpper(params[1])

	if subcommand == "ACK" {
		sc.Caps.EnableCaps(strings.Split(params[2], " ")...)
	} else if subcommand == "LS" {
		if len(params) > 3 {
			sc.Caps.AddCaps(strings.Split(params[3], " ")...)
		} else {
			sc.Caps.AddCaps(strings.Split(params[2], " ")...)
			capsToRequest := sc.Caps.ToRequestLine()

			if len(capsToRequest) > 0 {
				sc.Send(nil, "", "CAP", "REQ", capsToRequest)
			}

			if !sc.Registered {
				sc.Send(nil, "", "CAP", "END")
			}
		}
	} else if subcommand == "NEW" {
		sc.Caps.AddCaps(strings.Split(params[2], " ")...)
		capsToRequest := sc.Caps.ToRequestLine()

		if len(capsToRequest) > 0 {
			sc.Send(nil, "", "CAP", "REQ", capsToRequest)
		}
	} else if subcommand == "DEL" {
		sc.Caps.DelCaps(strings.Split(params[2], " ")...)
	}

	if !sc.Registered && (subcommand == "ACK" || subcommand == "NAK") {
		sendRegistration(sc)
	}
}

func pingHandler(event string, info eventmgr.InfoMap) {
	sc := info["server"].(*ServerConnection)
	sc.Send(nil, "", "PONG", info["params"].([]string)...)
}

func nicknameInUseHandler(event string, info eventmgr.InfoMap) {
	sc := info["server"].(*ServerConnection)
	if sc.Registered {
		return
	}

	// set new nickname
	if len(sc.FallbackNicks) <= sc.fallbackNickIndex {
		sc.Nick = fmt.Sprintf("%s_", sc.Nick)
	} else {
		sc.Nick = sc.FallbackNicks[sc.fallbackNickIndex]
		sc.fallbackNickIndex++
	}

	sc.Send(nil, "", "NICK", sc.Nick)
}

func sendRegistration(sc *ServerConnection) {
	sc.Nick = sc.InitialNick
	if sc.ConnectionPass != "" {
		sc.Send(nil, "", "PASS", sc.ConnectionPass)
	}
	sc.Send(nil, "", "NICK", sc.InitialNick)
	sc.Send(nil, "", "USER", sc.InitialUser, "0", "*", sc.InitialRealName)
}
