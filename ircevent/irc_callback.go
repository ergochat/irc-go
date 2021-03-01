package ircevent

import (
	"fmt"
	"log"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/goshuirc/irc-go/ircmsg"
)

const (
	// fake events that we manage specially
	registrationEvent = "*REGISTRATION"
)

// Tuple type for uniquely identifying callbacks
type CallbackID struct {
	eventCode string
	id        uint64
}

// Register a callback to a connection and event code. A callback is a function
// which takes only an Event object as parameter. Valid event codes are all
// IRC/CTCP commands and error/response codes. This function returns the ID of the
// registered callback for later management.
func (irc *Connection) AddCallback(eventCode string, callback func(Event)) CallbackID {
	return irc.addCallback(eventCode, Callback(callback), false, 0)
}

func (irc *Connection) addCallback(eventCode string, callback Callback, prepend bool, idNum uint64) CallbackID {
	eventCode = strings.ToUpper(eventCode)
	if eventCode == "" || strings.HasPrefix(eventCode, "*") {
		return CallbackID{}
	}

	irc.eventsMutex.Lock()
	defer irc.eventsMutex.Unlock()

	if irc.events == nil {
		irc.events = make(map[string][]callbackPair)
	}

	if idNum == 0 {
		idNum = irc.callbackCounter
		irc.callbackCounter++
	}
	id := CallbackID{eventCode: eventCode, id: idNum}
	newPair := callbackPair{id: id.id, callback: callback}
	current := irc.events[eventCode]
	newList := make([]callbackPair, len(current)+1)
	start := 0
	if prepend {
		newList[start] = newPair
		start++
	}
	copy(newList[start:], current)
	if !prepend {
		newList[len(newList)-1] = newPair
	}
	irc.events[eventCode] = newList
	return id
}

// Remove callback i (ID) from the given event code.
func (irc *Connection) RemoveCallback(id CallbackID) {
	irc.eventsMutex.Lock()
	defer irc.eventsMutex.Unlock()
	switch id.eventCode {
	case registrationEvent:
		irc.removeCallbackNoMutex(RPL_ENDOFMOTD, id.id)
		irc.removeCallbackNoMutex(ERR_NOMOTD, id.id)
	default:
		irc.removeCallbackNoMutex(id.eventCode, id.id)
	}
}

func (irc *Connection) removeCallbackNoMutex(code string, id uint64) {
	current := irc.events[code]
	if len(current) == 0 {
		return
	}
	newList := make([]callbackPair, 0, len(current)-1)
	for _, p := range current {
		if p.id != id {
			newList = append(newList, p)
		}
	}
	irc.events[code] = newList
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

	list := irc.events[id.eventCode]
	for i, p := range list {
		if p.id == id.id {
			list[i] = callbackPair{id: id.id, callback: callback}
			return true
		}
	}
	return false
}

// Convenience function to add a callback that will be called once the
// connection is completed (this is traditionally referred to as "connection
// registration").
func (irc *Connection) AddConnectCallback(callback func(Event)) (id CallbackID) {
	// XXX: forcibly use the same ID number for both copies of the callback
	id376 := irc.AddCallback(RPL_ENDOFMOTD, callback)
	irc.addCallback(ERR_NOMOTD, callback, false, id376.id)
	return CallbackID{eventCode: registrationEvent, id: id376.id}
}

func (irc *Connection) getCallbacks(code string) (result []callbackPair) {
	code = strings.ToUpper(code)

	irc.eventsMutex.Lock()
	defer irc.eventsMutex.Unlock()
	return irc.events[code]
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

	callbackPairs := irc.getCallbacks(event.Command)

	// just run the callbacks in serial, since it's not safe for them
	// to take a long time to execute in any case
	for _, pair := range callbackPairs {
		pair.callback(event)
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
	irc.AddCallback(ERR_NICKNAMEINUSE, irc.handleUnavailableNick)
	irc.AddCallback(ERR_UNAVAILRESOURCE, irc.handleUnavailableNick)

	// 001: RPL_WELCOME "Welcome to the Internet Relay Network <nick>!<user>@<host>"
	// Set irc.currentNick to the actually used nick in this connection.
	irc.AddCallback(RPL_WELCOME, irc.handleRplWelcome)

	// 005: RPL_ISUPPORT, conveys supported server features
	irc.AddCallback(RPL_ISUPPORT, irc.handleISupport)

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

	irc.AddCallback("CAP", irc.handleCAP)

	if irc.UseSASL {
		irc.setupSASLCallbacks()
	}

	if irc.EnableCTCP {
		irc.setupCTCPCallbacks()
	}

	// prepend our own callbacks for the end of registration,
	// so they happen before any client-added callbacks
	irc.addCallback(RPL_ENDOFMOTD, irc.handleRegistration, true, 0)
	irc.addCallback(ERR_NOMOTD, irc.handleRegistration, true, 0)
}

func (irc *Connection) handleRplWelcome(e Event) {
	irc.stateMutex.Lock()
	defer irc.stateMutex.Unlock()

	// set the nickname we actually received from the server
	if len(e.Params) > 0 {
		irc.currentNick = e.Params[0]
	}
}

func (irc *Connection) handleRegistration(e Event) {
	// wake up Connect() if applicable
	defer func() {
		select {
		case irc.welcomeChan <- empty{}:
		default:
		}
	}()

	irc.stateMutex.Lock()
	defer irc.stateMutex.Unlock()

	if irc.registered {
		return
	}
	irc.registered = true

	// mark the isupport complete
	irc.isupport = irc.isupportPartial
	irc.isupportPartial = nil

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

func (irc *Connection) handleISupport(e Event) {
	irc.stateMutex.Lock()
	defer irc.stateMutex.Unlock()

	// TODO handle 005 changes after registration
	if irc.isupportPartial == nil {
		return
	}
	if len(e.Params) < 3 {
		return
	}
	for _, token := range e.Params[1 : len(e.Params)-1] {
		equalsIdx := strings.IndexByte(token, '=')
		if equalsIdx == -1 {
			irc.isupportPartial[token] = "" // no value
		} else {
			irc.isupportPartial[token[:equalsIdx]] = unescapeISupportValue(token[equalsIdx+1:])
		}
	}
}

func unescapeISupportValue(in string) (out string) {
	if strings.IndexByte(in, '\\') == -1 {
		return in
	}
	var buf strings.Builder
	for i := 0; i < len(in); {
		if in[i] == '\\' && i+3 < len(in) && in[i+1] == 'x' {
			hex := in[i+2 : i+4]
			if octet, err := strconv.ParseInt(hex, 16, 8); err == nil {
				buf.WriteByte(byte(octet))
				i += 4
				continue
			}
		}
		buf.WriteByte(in[i])
		i++
	}
	return buf.String()
}

func (irc *Connection) handleCAP(e Event) {
	if len(e.Params) < 3 {
		return
	}
	ack := false
	// CAP <NICK | * > <SUBCOMMAND> PARAMS...
	switch e.Params[1] {
	case "LS":
		irc.handleCAPLS(e.Params[2:])
	case "ACK":
		ack = true
		fallthrough
	case "NAK":
		for _, token := range strings.Fields(e.Params[2]) {
			name, _ := splitCAPToken(token)
			if sliceContains(name, irc.RequestCaps) {
				select {
				case irc.capsChan <- capResult{capName: name, ack: ack}:
				default:
				}
			}
		}
	}
}

func (irc *Connection) handleCAPLS(params []string) {
	var capsToReq, capsNotFound []string
	defer func() {
		for _, c := range capsToReq {
			irc.Send("CAP", "REQ", c)
		}
		for _, c := range capsNotFound {
			select {
			case irc.capsChan <- capResult{capName: c, ack: false}:
			default:
			}
		}
	}()

	irc.stateMutex.Lock()
	defer irc.stateMutex.Unlock()

	if irc.registered {
		// TODO server could probably trick us into panic here by sending
		// additional LS before the end of registration
		return
	}

	if irc.capsAdvertised == nil {
		irc.capsAdvertised = make(map[string]string)
	}

	// multiline responses to CAP LS 302 start with a 4-parameter form:
	// CAP * LS * :account-notify away-notify [...]
	// and end with a 3-parameter form:
	// CAP * LS :userhost-in-names znc.in/playback [...]
	final := len(params) == 1
	for _, token := range strings.Fields(params[len(params)-1]) {
		name, value := splitCAPToken(token)
		irc.capsAdvertised[name] = value
	}

	if final {
		for _, c := range irc.RequestCaps {
			if _, ok := irc.capsAdvertised[c]; ok {
				capsToReq = append(capsToReq, c)
			} else {
				capsNotFound = append(capsNotFound, c)
			}
		}
	}
}

func splitCAPToken(token string) (name, value string) {
	equalIdx := strings.IndexByte(token, '=')
	if equalIdx == -1 {
		return token, ""
	} else {
		return token[:equalIdx], token[equalIdx+1:]
	}
}
