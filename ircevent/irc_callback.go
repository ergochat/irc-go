package ircevent

import (
	"errors"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

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
	case "BATCH":
		irc.removeBatchCallbackNoMutex(id.id)
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

// AddBatchCallback adds a callback for handling BATCH'ed server responses.
// All available BATCH callbacks will be invoked in an undefined order,
// stopping at the first one to return a value of true (indicating successful
// processing). If no batch callback returns true, the batch will be "flattened"
// (i.e., its messages will be processed individually by the normal event
// handlers). Batch callbacks can be removed as usual with RemoveCallback.
func (irc *Connection) AddBatchCallback(callback func(*Batch) bool) CallbackID {
	irc.eventsMutex.Lock()
	defer irc.eventsMutex.Unlock()

	idNum := irc.callbackCounter
	irc.callbackCounter++
	nbc := make([]batchCallbackPair, len(irc.batchCallbacks)+1)
	copy(nbc, irc.batchCallbacks)
	nbc[len(nbc)-1] = batchCallbackPair{id: idNum, callback: callback}
	irc.batchCallbacks = nbc
	return CallbackID{eventCode: "BATCH", id: idNum}
}

func (irc *Connection) removeBatchCallbackNoMutex(idNum uint64) {
	current := irc.batchCallbacks
	if len(current) == 0 {
		return
	}
	newList := make([]batchCallbackPair, 0, len(current)-1)
	for _, p := range current {
		if p.id != idNum {
			newList = append(newList, p)
		}
	}
	irc.batchCallbacks = newList
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

func (irc *Connection) getBatchCallbacks() (result []batchCallbackPair) {
	irc.eventsMutex.Lock()
	defer irc.eventsMutex.Unlock()

	return irc.batchCallbacks
}

var (
	// ad-hoc internal errors for batch processing
	// these indicate invalid data from the server (or else local corruption)
	errorDuplicateBatchID = errors.New("found duplicate batch ID")
	errorNoParentBatchID  = errors.New("parent batch ID not found")
	errorBatchNotOpen     = errors.New("tried to close batch, but batch ID not found")
	errorUnknownLabel     = errors.New("received labeled response from server, but we don't recognize the label")
)

func (irc *Connection) handleBatchCommand(msg ircmsg.IRCMessage) {
	if len(msg.Params) < 1 || len(msg.Params[0]) < 2 {
		irc.Log.Printf("Invalid BATCH command from server\n")
		return
	}

	start := msg.Params[0][0] == '+'
	if !start && msg.Params[0][0] != '-' {
		irc.Log.Printf("Invalid BATCH ID from server: %s\n", msg.Params[0])
		return
	}
	batchID := msg.Params[0][1:]
	isNested, parentBatchID := msg.GetTag("batch")
	var label int64
	if start {
		if present, labelStr := msg.GetTag("label"); present {
			label = deserializeLabel(labelStr)
		}
	}

	finishedBatch, callback, err := func() (finishedBatch *Batch, callback LabelCallback, err error) {
		irc.batchMutex.Lock()
		defer irc.batchMutex.Unlock()

		if start {
			if _, ok := irc.batches[batchID]; ok {
				err = errorDuplicateBatchID
				return
			}
			batchObj := new(Batch)
			batchObj.IRCMessage = msg
			irc.batches[batchID] = batchInProgress{
				createdAt: time.Now(),
				batch:     batchObj,
				label:     label,
			}
			if isNested {
				parentBip := irc.batches[parentBatchID]
				if parentBip.batch == nil {
					err = errorNoParentBatchID
					return
				}
				parentBip.batch.Items = append(parentBip.batch.Items, batchObj)
			}
		} else {
			bip := irc.batches[batchID]
			if bip.batch == nil {
				err = errorBatchNotOpen
				return
			}
			delete(irc.batches, batchID)
			if !isNested {
				finishedBatch = bip.batch
				if bip.label != 0 {
					callback = irc.getLabelCallbackNoMutex(bip.label)
					if callback == nil {
						err = errorUnknownLabel
					}

				}
			}
		}
		return
	}()

	if err != nil {
		irc.Log.Printf("batch error: %v (batchID=`%s`, parentBatchID=`%s`)", err, batchID, parentBatchID)
	} else if callback != nil {
		callback(finishedBatch)
	} else if finishedBatch != nil {
		irc.HandleBatch(finishedBatch)
	}
}

func (irc *Connection) getLabelCallbackNoMutex(label int64) (callback LabelCallback) {
	if lc, ok := irc.labelCallbacks[label]; ok {
		callback = lc.callback
		delete(irc.labelCallbacks, label)
	}
	return
}

func (irc *Connection) getLabelCallback(label int64) (callback LabelCallback) {
	irc.batchMutex.Lock()
	defer irc.batchMutex.Unlock()
	return irc.getLabelCallbackNoMutex(label)
}

// HandleBatch handles a *Batch using available handlers, "flattening" it if
// no handler succeeds. This can be used in a batch or labeled-response callback
// to process inner batches.
func (irc *Connection) HandleBatch(batch *Batch) {
	if batch == nil {
		return
	}

	success := false
	for _, bh := range irc.getBatchCallbacks() {
		if bh.callback(batch) {
			success = true
			break
		}
	}
	if !success {
		irc.handleBatchNaively(batch)
	}
}

// recursively "flatten" the nested batch; process every command individually
func (irc *Connection) handleBatchNaively(batch *Batch) {
	if batch.Command != "BATCH" {
		irc.HandleEvent(Event{IRCMessage: batch.IRCMessage})
	}
	for _, item := range batch.Items {
		irc.handleBatchNaively(item)
	}
}

func (irc *Connection) handleBatchedCommand(msg ircmsg.IRCMessage, batchID string) {
	irc.batchMutex.Lock()
	defer irc.batchMutex.Unlock()

	bip := irc.batches[batchID]
	if bip.batch == nil {
		irc.Log.Printf("ignoring command with unknown batch ID %s\n", batchID)
		return
	}
	bip.batch.Items = append(bip.batch.Items, &Batch{IRCMessage: msg})
}

// Execute all callbacks associated with a given event.
func (irc *Connection) runCallbacks(msg ircmsg.IRCMessage) {
	if !irc.AllowPanic {
		defer func() {
			if r := recover(); r != nil {
				irc.Log.Printf("Caught panic in callback: %v\n%s", r, debug.Stack())
			}
		}()
	}

	// handle batch start or end
	if irc.batchNegotiated {
		if msg.Command == "BATCH" {
			irc.handleBatchCommand(msg)
			return
		} else if hasBatchTag, batchID := msg.GetTag("batch"); hasBatchTag {
			irc.handleBatchedCommand(msg, batchID)
			return
		}
	}

	// handle labeled single command, or labeled ACK
	if irc.labelNegotiated {
		if hasLabel, labelStr := msg.GetTag("label"); hasLabel {
			var labelCallback LabelCallback
			if label := deserializeLabel(labelStr); label != 0 {
				labelCallback = irc.getLabelCallback(label)
			}
			if labelCallback == nil {
				irc.Log.Printf("received unrecognized label from server: %s\n", labelStr)
				return
			} else {
				labelCallback(&Batch{
					IRCMessage: msg,
				})
			}
			return
		}
	}

	// OK, it's a normal IRC command
	irc.HandleEvent(Event{IRCMessage: msg})
}

// HandleEvent handles an IRC line using the available handlers. This can be
// used in a batch or labeled-response callback to process an individual line.
func (irc *Connection) HandleEvent(event Event) {
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

	irc.AddCallback("FAIL", irc.handleStandardReplies)
	irc.AddCallback("WARN", irc.handleStandardReplies)
	irc.AddCallback("NOTE", irc.handleStandardReplies)
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

// labeled-response

func (irc *Connection) registerLabel(callback LabelCallback) string {
	irc.batchMutex.Lock()
	defer irc.batchMutex.Unlock()
	irc.labelCounter++ // increment first: 0 is an invalid label
	label := irc.labelCounter
	irc.labelCallbacks[label] = pendingLabel{
		createdAt: time.Now(),
		callback:  callback,
	}
	return serializeLabel(label)
}

func (irc *Connection) unregisterLabel(labelStr string) {
	label := deserializeLabel(labelStr)
	if label == 0 {
		return
	}
	irc.batchMutex.Lock()
	defer irc.batchMutex.Unlock()
	delete(irc.labelCallbacks, label)
}

// expire open batches from the server that weren't closed in a
// timely fashion. `force` expires all label callbacks regardless
// of time created (so they can be cleaned up when the connection
// fails).
func (irc *Connection) expireBatches(force bool) {
	var failedCallbacks []LabelCallback
	defer func() {
		for _, bcb := range failedCallbacks {
			bcb(nil)
		}
	}()

	irc.batchMutex.Lock()
	defer irc.batchMutex.Unlock()
	now := time.Now()

	for label, lcb := range irc.labelCallbacks {
		if force || now.Sub(lcb.createdAt) > irc.KeepAlive {
			failedCallbacks = append(failedCallbacks, lcb.callback)
			delete(irc.labelCallbacks, label)
		}
	}

	for batchID, bip := range irc.batches {
		if now.Sub(bip.createdAt) > irc.KeepAlive {
			delete(irc.batches, batchID)
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

func (irc *Connection) handleStandardReplies(e Event) {
	// unconditionally print messages for FAIL and WARN;
	// re. NOTE, if debug is enabled, we print the raw line anyway
	switch e.Command {
	case "FAIL", "WARN":
		irc.Log.Printf("Received error code from server: %s %s\n", e.Command, strings.Join(e.Params, " "))
	}
}

const (
	labelBase = 32
)

func serializeLabel(label int64) string {
	return strconv.FormatInt(label, labelBase)
}

func deserializeLabel(str string) int64 {
	if p, err := strconv.ParseInt(str, labelBase, 64); err == nil {
		return p
	}
	return 0
}
