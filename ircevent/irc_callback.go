package ircevent

import (
	"fmt"
	"log"
	"runtime/debug"
	"strings"

	"github.com/goshuirc/irc-go/ircmsg"
)

// Tuple type for uniquely identifying callbacks
type CallbackID struct {
	eventCode string
	id        uint64
}

// Register a callback to a connection and event code. A callback is a function
// which takes only an Event pointer as parameter. Valid event codes are all
// IRC/CTCP commands and error/response codes. To register a callback for all
// events pass "*" as the event code. This function returns the ID of the
// registered callback for later management.
func (irc *Connection) AddCallback(eventCode string, callback func(Event)) CallbackID {
	eventCode = strings.ToUpper(eventCode)

	irc.eventsMutex.Lock()
	defer irc.eventsMutex.Unlock()

	if irc.events == nil {
		irc.events = make(map[string]map[uint64]Callback)
	}

	_, ok := irc.events[eventCode]
	if !ok {
		irc.events[eventCode] = make(map[uint64]Callback)
	}
	id := CallbackID{eventCode: eventCode, id: irc.idCounter}
	irc.idCounter++
	irc.events[eventCode][id.id] = Callback(callback)
	return id
}

// Remove callback i (ID) from the given event code.
func (irc *Connection) RemoveCallback(id CallbackID) {
	irc.eventsMutex.Lock()
	defer irc.eventsMutex.Unlock()
	delete(irc.events[id.eventCode], id.id)
}

// Remove all callbacks from a given event code.
func (irc *Connection) ClearCallback(eventcode string) {
	eventcode = strings.ToUpper(eventcode)

	irc.eventsMutex.Lock()
	defer irc.eventsMutex.Unlock()
	delete(irc.events, eventcode)
}

// Replace callback i (ID) associated with a given event code with a new callback function.
func (irc *Connection) ReplaceCallback(id CallbackID, callback func(Event)) bool {
	irc.eventsMutex.Lock()
	defer irc.eventsMutex.Unlock()

	if _, ok := irc.events[id.eventCode][id.id]; ok {
		irc.events[id.eventCode][id.id] = callback
		return true
	}
	return false
}

func (irc *Connection) getCallbacks(code string) (result []Callback) {
	code = strings.ToUpper(code)

	irc.eventsMutex.Lock()
	defer irc.eventsMutex.Unlock()

	cMap := irc.events[code]
	starMap := irc.events["*"]
	length := len(cMap) + len(starMap)
	if length == 0 {
		return
	}
	result = make([]Callback, 0, length)
	for _, c := range cMap {
		result = append(result, c)
	}
	for _, c := range starMap {
		result = append(result, c)
	}
	return
}

// Execute all callbacks associated with a given event.
func (irc *Connection) runCallbacks(msg ircmsg.IRCMessage) {
	if !irc.AllowPanic {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Caught panic in callback: %v\n%s", r, debug.Stack())
			}
		}()
	}

	event := Event{IRCMessage: msg}

	if irc.EnableCTCP {
		eventRewriteCTCP(&event)
	}

	callbacks := irc.getCallbacks(event.Command)

	// just run the callbacks in serial, since it's not safe for them
	// to take a long time to execute in any case
	for _, callback := range callbacks {
		callback(event)
	}
}

// Set up some initial callbacks to handle the IRC/CTCP protocol.
func (irc *Connection) setupCallbacks() {
	irc.stateMutex.Lock()
	needBaseCallbacks := !irc.hasBaseCallbacks
	irc.hasBaseCallbacks = true
	irc.stateMutex.Unlock()

	if !needBaseCallbacks {
		return
	}

	// PING: we must respond with the correct PONG
	irc.AddCallback("PING", func(e Event) { irc.Send("PONG", e.Message()) })

	// PONG: record time to make sure the server is responding to us
	irc.AddCallback("PONG", func(e Event) {
		irc.recordPong(e.Message())
	})

	// 433: ERR_NICKNAMEINUSE "<nick> :Nickname is already in use"
	// 437: ERR_UNAVAILRESOURCE "<nick/channel> :Nick/channel is temporarily unavailable"
	irc.AddCallback("433", irc.handleUnavailableNick)
	irc.AddCallback("437", irc.handleUnavailableNick)

	// 1: RPL_WELCOME "Welcome to the Internet Relay Network <nick>!<user>@<host>"
	// Set irc.currentNick to the actually used nick in this connection.
	irc.AddCallback("001", irc.handleRplWelcome)

	// respond to NICK from the server (in response to our own NICK, or sent unprompted)
	irc.AddCallback("NICK", func(e Event) {
		if e.Nick() == irc.CurrentNick() && len(e.Params) > 0 {
			irc.setCurrentNick(e.Params[0])
		}
	})

	irc.AddCallback("ERROR", func(e Event) {
		if !irc.isQuitting() {
			irc.Log.Printf("ERROR received from server: %s", strings.Join(e.Params, " "))
		}
	})

	irc.AddCallback("CAP", func(e Event) {
		if len(e.Params) != 3 {
			return
		}
		command := e.Params[1]
		capsChan := irc.capsChan

		// TODO this assumes all the caps on one line
		// TODO support CAP LS 302
		if command == "LS" {
			capsList := strings.Fields(e.Params[2])
			for _, capName := range irc.RequestCaps {
				if sliceContains(capName, capsList) {
					irc.Send("CAP", "REQ", capName)
				} else {
					select {
					case capsChan <- capResult{capName, false}:
					default:
					}
				}
			}
		} else if command == "ACK" || command == "NAK" {
			for _, capName := range strings.Fields(e.Params[2]) {
				select {
				case capsChan <- capResult{capName, command == "ACK"}:
				default:
				}
			}
		}
	})

	if irc.UseSASL {
		irc.setupSASLCallbacks()
	}

	if irc.EnableCTCP {
		irc.setupCTCPCallbacks()
	}
}

func (irc *Connection) handleRplWelcome(e Event) {
	irc.stateMutex.Lock()
	defer irc.stateMutex.Unlock()

	// set the nickname we actually received from the server
	if len(e.Params) > 0 {
		irc.currentNick = e.Params[0]
	}

	// wake up Connect() if applicable
	select {
	case irc.welcomeChan <- empty{}:
	default:
	}
}

func (irc *Connection) handleUnavailableNick(e Event) {
	// only try to change the nick if we're not registered yet,
	// otherwise we'll change in response to pingLoop unsuccessfully
	// trying to restore the intended nick (swapping one undesired nick
	// for another)
	var nickToTry string
	irc.stateMutex.Lock()
	if irc.currentNick == "" {
		nickToTry = fmt.Sprintf("%s_%d", irc.Nick, irc.nickCounter)
		irc.nickCounter++
	}
	irc.stateMutex.Unlock()

	if nickToTry != "" {
		irc.Send("NICK", nickToTry)
	}
}
