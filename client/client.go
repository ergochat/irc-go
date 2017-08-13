// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package gircclient

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/goshuirc/eventmgr"
	"github.com/goshuirc/irc-go/ircmap"
	"github.com/goshuirc/irc-go/ircmsg"
)

// ServerConnection is a connection to a single server.
type ServerConnection struct {
	Name        string
	Connected   bool
	Registered  bool
	Casemapping ircmap.MappingType

	// internal stuff
	RawConnection  net.Conn
	eventsIn       eventmgr.EventManager
	eventsOut      eventmgr.EventManager
	channelsToJoin []channel

	// data we keep track of
	Features ServerFeatures
	Caps     ClientCapabilities

	// details users must supply before connection
	Nick              string
	InitialNick       string
	FallbackNicks     []string
	fallbackNickIndex int
	InitialUser       string
	InitialRealName   string
	ConnectionPass    string

	// options
	SimplifyEvents bool
}

// newServerConnection returns an initialised ServerConnection, for internal
// use.
func newServerConnection(name string) *ServerConnection {
	var sc ServerConnection

	sc.Name = name
	sc.Caps = NewClientCapabilities()
	sc.Features = make(ServerFeatures)

	sc.Caps.AddWantedCaps("account-notify", "away-notify", "extended-join", "multi-prefix", "sasl")
	sc.Caps.AddWantedCaps("account-tag", "cap-notify", "chghost", "invite-notify", "server-time", "userhost-in-names")

	sc.Features.Parse("CHANTYPES=#", "LINELEN=512", "PREFIX=(ov)@+")

	sc.SimplifyEvents = true

	return &sc
}

// Connect connects to the given address.
func (sc *ServerConnection) Connect(address string, ssl bool, tlsconfig *tls.Config) error {
	// check the required attributes
	if sc.InitialNick == "" || sc.InitialUser == "" {
		return errors.New("InitialNick and InitialUser must be set before connecting")
	}

	// connect
	var conn net.Conn
	var err error

	if ssl {
		conn, err = tls.Dial("tcp", address, tlsconfig)
	} else {
		conn, err = net.Dial("tcp", address)
	}

	if err != nil {
		return err
	}

	sc.RawConnection = conn
	sc.Connected = true

	sc.Send(nil, "", "CAP", "LS", "302")

	return nil
}

// JoinChannel joins a channel, or marks the channel as to be joined after registration.
func (sc *ServerConnection) JoinChannel(name string, key string, useKey bool) {
	if sc.Registered {
		params := []string{name}
		if useKey {
			params = []string{name, key}
		}
		sc.Send(nil, "", "JOIN", params...)
	} else {
		sc.channelsToJoin = append(sc.channelsToJoin, channel{
			Name:   name,
			Key:    key,
			UseKey: useKey,
		})
	}
}

// WaitForConnection waits for the serverConnection to become available.
// This is used when writing a custom event loop.
func (sc *ServerConnection) WaitForConnection() {
	waitTime := 10 * time.Millisecond
	for sc.RawConnection == nil {
		time.Sleep(waitTime)
	}
}

// ProcessIncomingLine processes the incoming IRC line.
// This is used when writing a custom event loop.
func (sc *ServerConnection) ProcessIncomingLine(line string) {
	line = strings.Trim(line, "\r\n")

	// ignore empty lines
	if len(line) < 1 {
		return
	}

	// dispatch raw
	rawInfo := eventmgr.NewInfoMap()
	rawInfo["server"] = sc
	rawInfo["direction"] = "in"
	rawInfo["data"] = line

	sc.dispatchRawIn(rawInfo)

	// dispatch events
	message, err := ircmsg.ParseLine(line)

	// convert numerics to names
	cmd := message.Command
	num, err := strconv.Atoi(cmd)
	if err == nil {
		name, exists := Numerics[num]
		if exists {
			cmd = name
		}
	}

	info := eventmgr.NewInfoMap()
	info["server"] = sc
	info["direction"] = "in"
	info["tags"] = message.Tags
	info["prefix"] = message.Prefix
	info["command"] = cmd
	info["params"] = message.Params

	// simplify event
	if sc.SimplifyEvents {
		err = SimplifyEvent(info)

		if err != nil {
			fmt.Println("Could not simplify incoming IRC message, skipping line.")
			fmt.Println("line:", line)
			fmt.Println("error:", err)
			fmt.Println("info:", info)
			return
		}
	}

	// IRC commands are case-insensitive
	sc.dispatchIn(strings.ToUpper(cmd), info)
}

// Disconnect closes the IRC socket.
// It is used when writing your own event loop.
func (sc *ServerConnection) Disconnect() {
	sc.Connected = false
	sc.RawConnection.Close()
	info := eventmgr.NewInfoMap()
	info["server"] = sc
	sc.dispatchOut("server disconnected", info)
}

// ReceiveLoop runs a loop of receiving and dispatching new messages.
func (sc *ServerConnection) ReceiveLoop() {
	// wait for the connection to become available
	sc.WaitForConnection()

	reader := bufio.NewReader(sc.RawConnection)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		sc.ProcessIncomingLine(line)
	}

	sc.Disconnect()
}

// RegisterEvent registers a new handler for the given event.
//
// The standard directions are "in" and "out".
//
// 'name' can either be the name of an event, "all", or "raw". Note that "all"
// will not catch "raw" events, but will catch all others.
func (sc *ServerConnection) RegisterEvent(direction string, name string, handler eventmgr.HandlerFn, priority int) {
	if direction == "in" || direction == "both" {
		sc.eventsIn.Attach(name, handler, priority)
	}
	if direction == "out" || direction == "both" {
		sc.eventsOut.Attach(name, handler, priority)
	}
}

// Shutdown closes the connection to the server.
func (sc *ServerConnection) Shutdown(message string) {
	sc.Send(nil, "", "QUIT", message)
	sc.Connected = false
	sc.RawConnection.Close()
}

// Casefold folds the given string using the server's casemapping.
func (sc *ServerConnection) Casefold(message string) (string, error) {
	return ircmap.Casefold(sc.Casemapping, message)
}

// Send sends an IRC message to the server. If the message cannot be converted
// to a raw IRC line, an error is returned.
func (sc *ServerConnection) Send(tags *map[string]ircmsg.TagValue, prefix string, command string, params ...string) error {
	msg := ircmsg.MakeMessage(tags, prefix, command, params...)
	line, err := msg.Line()
	if err != nil {
		return err
	}
	fmt.Fprintf(sc.RawConnection, line)

	// dispatch raw event
	info := eventmgr.NewInfoMap()
	info["server"] = sc
	info["direction"] = "out"
	info["data"] = line
	sc.dispatchRawOut(info)

	var outTags map[string]ircmsg.TagValue
	if tags == nil {
		outTags = *ircmsg.MakeTags()
	} else {
		outTags = *tags
	}

	// dispatch real event
	info = eventmgr.NewInfoMap()
	info["server"] = sc
	info["direction"] = "out"
	info["tags"] = outTags
	info["prefix"] = prefix
	info["command"] = command
	info["params"] = params

	// IRC commands are case-insensitive
	sc.dispatchOut(strings.ToUpper(command), info)

	return nil
}

// dispatchRawIn dispatches raw inbound messages.
func (sc *ServerConnection) dispatchRawIn(info eventmgr.InfoMap) {
	sc.eventsIn.Dispatch("raw", info)
}

// dispatchIn dispatches inbound messages.
func (sc *ServerConnection) dispatchIn(name string, info eventmgr.InfoMap) {
	sc.eventsIn.Dispatch(name, info)
	sc.eventsIn.Dispatch("all", info)
}

// dispatchRawOut dispatches raw outbound messages.
func (sc *ServerConnection) dispatchRawOut(info eventmgr.InfoMap) {
	sc.eventsOut.Dispatch("raw", info)
}

// dispatchOut dispatches outbound messages.
func (sc *ServerConnection) dispatchOut(name string, info eventmgr.InfoMap) {
	sc.eventsOut.Dispatch(name, info)
	sc.eventsOut.Dispatch("all", info)
}

// IsChannel returns true if the given target is a channel.
func (sc *ServerConnection) IsChannel(target string) bool {
	channelChars := sc.Features["CHANTYPES"].(string)
	return strings.ContainsAny(string(target[0]), channelChars)
}
