// Copyright 2009 Thomas Jager <mail@jager.no>  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Here's the concurrency design of this project (largely unchanged from thoj/go-ircevent):
A successful Connect() spawns 3 goroutines (readLoop, writeLoop, maintenanceLoop).
maintenanceLoop() waits for the other goroutines to make a clean stop, then runs another
Connect(). The system can be interrupted asynchronously by sending a message, e.g, with
Privmsg(), or by calling Reconnect() (which disconnects and forces a reconnection),
or by calling Quit(), which sends QUIT to the server and will eventually stop maintenanceLoop().

The stop mechanism is to close the (*Connection).end channel (which is only closed,
never sent-on normally), so every blocking operation in the 3 loops must also
select on `end` to make sure it stops in a timely fashion.

The state machine is like this:

[Successful connect]
ConnectionNotStarted --(Connect)--> ConnectionConnecting --(successful handshake)--> ConnectionActive
[Maintenance loop after successful connect]
ConnectionActive --(connection drops)--> ConnectionSleeping -> ConnectionReconnecting -> ConnectionActive
[Quit() called]
{Any state} --(Quit called)--> ConnectionStopping -> ConnectionStopped
[Unsuccessful initial connect]
ConnectionNotStarted --(Connect)--> ConnectionConnecting --(failed handshake)--> ConnectionNotStarted
*/

package ircevent

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ergochat/irc-go/ircmsg"
	"github.com/ergochat/irc-go/ircreader"
)

const (
	Version = "ergochat/irc-go"

	// prefix for keepalive ping parameters
	keepalivePrefix = "KeepAlive-"

	maxlenTags = 8192

	defaultNick = "ircevent"

	CAPTimeout = time.Second * 15
)

var (
	ClientDisconnected = errors.New("Could not send because client is disconnected")
	ServerTimedOut     = errors.New("Server did not respond in time")
	ServerDisconnected = errors.New("Disconnected by server")
	SASLFailed         = errors.New("SASL setup timed out. Does the server support SASL?")

	CapabilityNotNegotiated = errors.New("The IRCv3 capability required for this was not negotiated")
	NoLabeledResponse       = errors.New("The server failed to send a labeled response to the command")

	serverDidNotQuit        = errors.New("server did not respond to QUIT")
	connectionAlreadyActive = errors.New("connection is already active")
	ClientHasQuit           = errors.New("client has called Quit()")
)

// Call this on an error forcing a disconnection:
// record the error, then close the `end` channel, stopping all goroutines
func (irc *Connection) setError(err error) {
	irc.stateMutex.Lock()
	defer irc.stateMutex.Unlock()
	if irc.lastError == nil {
		irc.lastError = err
		irc.closeEndNoMutex()
	}
}

func (irc *Connection) getError() error {
	irc.stateMutex.Lock()
	defer irc.stateMutex.Unlock()
	return irc.lastError
}

// Send a keepalive PING in our timestamp-based format
func (irc *Connection) generatePing() string {
	return fmt.Sprintf("%s%d", keepalivePrefix, time.Now().UnixNano())
}

// Decode a keepalive PONG parameter back to a time, returning the zero time if invalid
func (irc *Connection) decodePong(pingParam string) (result time.Time) {
	ts := strings.TrimPrefix(pingParam, keepalivePrefix)
	if ts == pingParam {
		return
	}
	t, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return
	}
	return time.Unix(0, t)
}

// Record a PONG response
func (irc *Connection) recordPong(param string) {
	success := func() bool {
		irc.stateMutex.Lock()
		defer irc.stateMutex.Unlock()

		if param != "" && irc.pingSent == param {
			irc.pingSent = ""
			return true
		}
		return false
	}()

	if !success {
		return
	}

	if irc.Debug {
		if pong := irc.decodePong(param); !pong.IsZero() {
			irc.Log.Printf("Lag: %v\n", time.Since(pong))
		}
	}
}

// Read data from a connection. To be used as a goroutine.
func (irc *Connection) readLoop() {
	defer irc.wg.Done()

	defer func() {
		irc.expireBatches(true)
		if irc.registered {
			irc.runDisconnectCallbacks()
		}
	}()

	var reader ircreader.Reader
	reader.Initialize(irc.socket, 1024, irc.MaxLineLen+maxlenTags)

	lastExpireCheck := time.Now()

	for {
		msgBytes, err := reader.ReadLine()
		if err != nil {
			select {
			case <-irc.end:
				// shut down elsewhere, benign
			default:
				irc.setError(err)
			}
			return
		}
		msg := string(msgBytes)

		if irc.Debug {
			irc.Log.Printf("<-- %s\n", strings.TrimSpace(msg))
		}

		parsedMsg, err := ircmsg.ParseLine(msg)
		if err == nil {
			err = irc.runCallbacks(parsedMsg, msg)
			if err != nil {
				irc.setError(err)
				return
			}
		} else {
			irc.Log.Printf("invalid message from server: %v\n", err)
		}
		irc.runRawCallbacks(msg, parsedMsg, err)

		if irc.batchNegotiated() && time.Since(lastExpireCheck) > irc.Timeout {
			irc.expireBatches(false)
			lastExpireCheck = time.Now()
		}
	}
}

// Loop to write to a connection. To be used as a goroutine.
func (irc *Connection) writeLoop() {
	defer irc.wg.Done()

	tick := 0
	ticker := time.NewTicker(irc.Timeout)
	defer ticker.Stop()

	for {
		select {
		case <-irc.end:
			return
		case b := <-irc.pwrite:
			if err := irc.performWrite(b); err != nil {
				return
			}
		case <-ticker.C:
			tick++
			msgs := irc.processTick(tick)
			for _, msg := range msgs {
				if err := irc.performWrite(msg); err != nil {
					return
				}
			}
		}
	}
}

func (irc *Connection) performWrite(b []byte) error {
	if len(b) == 0 {
		return nil
	}

	if irc.Debug {
		irc.Log.Printf("--> %s\n", bytes.TrimSpace(b))
	}

	irc.socket.SetWriteDeadline(time.Now().Add(irc.Timeout))
	_, err := irc.socket.Write(b)
	irc.socket.SetWriteDeadline(time.Time{})
	if err != nil {
		irc.setError(err)
	}
	return err
}

// check the status of the connection and take appropriate action
func (irc *Connection) processTick(tick int) (msgs [][]byte) {
	pingParam, shouldRenick, err := irc.checkStatusForTick(tick)
	if err != nil {
		irc.setError(err)
		return
	}
	if pingParam != "" {
		msgs = append(msgs, []byte(fmt.Sprintf("PING %s\r\n", pingParam)))
	}
	if shouldRenick {
		nickMsg := ircmsg.MakeMessage(nil, "", "NICK", irc.PreferredNick())
		if msg, err := nickMsg.LineBytesStrict(true, irc.MaxLineLen); err == nil {
			msgs = append(msgs, msg)
		}
	}
	return msgs
}

func (irc *Connection) checkStatusForTick(tick int) (pingParam string, shouldRenick bool, err error) {
	irc.stateMutex.Lock()
	defer irc.stateMutex.Unlock()

	// XXX: handle the server ignoring QUIT
	if irc.connectionState == ConnectionStopping && time.Since(irc.quitAt) >= irc.Timeout {
		err = serverDidNotQuit
		return
	}
	if irc.pingSent != "" {
		// unacked PING is fatal
		err = ServerTimedOut
		return
	}
	pingModulus := int(irc.KeepAlive / irc.Timeout)
	if tick%pingModulus == 0 {
		irc.pingSent = irc.generatePing()
		pingParam = irc.pingSent
		if irc.currentNick != irc.Nick {
			shouldRenick = true
		}
	}
	return
}

func (irc *Connection) isQuitting() bool {
	state := irc.State()
	return (state == ConnectionStopping || state == ConnectionStopped)
}

// Wait blocks until the IRC connection has been stopped intentionally, via Quit().
// Calls to Wait() are only valid after an initial Connect() call has succeeded.
func (irc *Connection) Wait() {
	if err := irc.normalizeConfig(); err != nil {
		return
	}
	// if the initial Connect() call succeeded, this gets unblocked by maintenanceLoop()
	// exiting after Quit(). if it didn't, this may never unblock, but there's not much
	// we can do about it:
	<-irc.quitEvent
}

// Loop blocks until the IRC connection has been stopped intentionally, via Quit().
// It is an alias for Wait(), retained for compatibility reasons.
func (irc *Connection) Loop() {
	irc.Wait()
}

// Main loop to control the connection.
func (irc *Connection) maintenanceLoop(lastReconnect time.Time) {
	defer func() {
		close(irc.quitEvent)
	}()

	for {
		state := irc.waitForStop(ConnectionSleeping)

		if state == ConnectionStopping || state == ConnectionStopped {
			return
		}

		if err := irc.getError(); err != nil {
			irc.Log.Printf("Error, disconnected: %s\n", err)
		}

		delay := time.Until(lastReconnect.Add(irc.ReconnectFreq))
		if delay > 0 {
			if irc.Debug {
				irc.Log.Printf("Waiting %v to reconnect", delay)
			}
			t := time.NewTimer(delay)
			select {
			case <-t.C:
			case <-irc.reconnSig:
				if irc.Debug {
					irc.Log.Printf("Sleep between reconnect attempts interrupted")
				}
				t.Stop()
			}
		} else {
			// drain any buffered Reconnect() request even if there was no delay
			select {
			case <-irc.reconnSig:
			default:
			}
		}

		lastReconnect = time.Now()
		err := irc.connectInternal(ConnectionReconnecting)
		if err != nil {
			// we are still stopped, the stop checks will return immediately
			irc.Log.Printf("Error while reconnecting: %s\n", err)
		}
	}
}

// wait for all goroutines to stop. XXX: this is not safe for concurrent
// use, call only from Connect() and maintenanceLoop() (which have a proper
// happens-before relation)
func (irc *Connection) waitForStop(nextState ConnectionState) ConnectionState {
	<-irc.end
	irc.wg.Wait() // wait for readLoop and writeLoop to terminate fully

	irc.stateMutex.Lock()
	defer irc.stateMutex.Unlock()
	switch irc.connectionState {
	case ConnectionStopping, ConnectionStopped:
		// after Quit() we only allow transitions forwards to the stopped state:
		irc.connectionState = ConnectionStopped
	default:
		irc.connectionState = nextState
	}
	irc.socket = nil
	// preserve old guarantee that CurrentNick() returns "" while disconnected:
	irc.currentNick = ""
	return irc.connectionState
}

// State returns the state of the connection to the IRC server.
func (irc *Connection) State() ConnectionState {
	irc.stateMutex.Lock()
	defer irc.stateMutex.Unlock()
	return irc.connectionState
}

// Quit starts the process of disconnecting from the IRC server,
// typically by sending the QUIT command. It returns immediately;
// use Wait() to block until the connection is fully stopped.
func (irc *Connection) Quit() {
	if err := irc.normalizeConfig(); err != nil {
		return
	}

	success := func() bool {
		irc.stateMutex.Lock()
		defer irc.stateMutex.Unlock()

		switch irc.connectionState {
		case ConnectionNotStarted:
			irc.connectionState = ConnectionStopped
			close(irc.quitEvent)
			return false
		case ConnectionConnecting, ConnectionActive, ConnectionSleeping, ConnectionReconnecting:
			irc.quitAt = time.Now()
			irc.connectionState = ConnectionStopping
			return true
		default:
			return false
		}
	}()

	if !success {
		return
	}

	// interrupt sleep if applicable
	select {
	case irc.reconnSig <- struct{}{}:
	default:
	}

	// send QUIT (this is a no-op if we are already stopped)
	quitMessage := irc.QuitMessage
	if quitMessage == "" {
		quitMessage = irc.Version
	}
	// the server will respond to this by closing our connection;
	// if it doesn't, processTick will eventually notice and close it
	irc.Send("QUIT", quitMessage)
}

func (irc *Connection) sendInternal(b []byte) (err error) {
	// XXX ensure that (end, pwrite) are from the same instantiation of Connect;
	// invocations of this function from callbacks originating in readLoop
	// do not need this synchronization (indeed they cannot occur at a time when
	// `end` is closed), but invocations from outside do (even though the race window
	// is very small).
	irc.stateMutex.Lock()
	state := irc.connectionState
	end := irc.end
	pwrite := irc.pwrite
	irc.stateMutex.Unlock()

	// XXX edge case: pwrite and end can both be nil during ConnectionConnecting
	// of the first Connect() (during the dial, before the socket is open)
	if state == ConnectionNotStarted || state == ConnectionSleeping ||
		state == ConnectionStopped || pwrite == nil {
		return ClientDisconnected
	}

	select {
	case pwrite <- b:
		return nil
	case <-end:
		return ClientDisconnected
	}
}

// SendIRCMessage sends a built ircmsg.Message.
func (irc *Connection) SendIRCMessage(msg ircmsg.Message) error {
	b, err := msg.LineBytesStrict(true, irc.MaxLineLen)
	if err != nil && !(irc.AllowTruncation && err == ircmsg.ErrorBodyTooLong) {
		if irc.Debug {
			irc.Log.Printf("couldn't assemble message: %v\n", err)
		}
		return err
	}
	return irc.sendInternal(b)
}

// SendWithTags sends an IRC message with IRCv3 message tags.
func (irc *Connection) SendWithTags(tags map[string]string, command string, params ...string) error {
	return irc.SendIRCMessage(ircmsg.MakeMessage(tags, "", command, params...))
}

// Send sends an IRC message.
func (irc *Connection) Send(command string, params ...string) error {
	return irc.SendWithTags(nil, command, params...)
}

// SendWithLabel sends an IRC message using the IRCv3 labeled-response specification.
// Instead of being processed by normal event handlers, the server response to the
// command will be collected into a *Batch and passed to the provided callback.
// If the server fails to respond correctly, the callback will be invoked with `nil`
// as the argument.
func (irc *Connection) SendWithLabel(callback func(*Batch), tags map[string]string, command string, params ...string) error {
	if !irc.labelNegotiated() {
		return CapabilityNotNegotiated
	}

	label := irc.registerLabel(callback)

	msg := ircmsg.MakeMessage(tags, "", command, params...)
	msg.SetTag("label", label)
	err := irc.SendIRCMessage(msg)
	if err != nil {
		irc.unregisterLabel(label)
	}
	return err
}

// GetLabeledResponse sends an IRC message using the IRCv3 labeled-response
// specification, then synchronously waits for the response, which is returned
// as a *Batch. If the server fails to respond correctly, an error will be
// returned.
func (irc *Connection) GetLabeledResponse(tags map[string]string, command string, params ...string) (batch *Batch, err error) {
	done := make(chan empty)
	err = irc.SendWithLabel(func(b *Batch) {
		batch = b
		close(done)
	}, tags, command, params...)
	if err != nil {
		return
	}
	<-done
	if batch == nil {
		err = NoLabeledResponse
	}
	return
}

// Send a raw string.
func (irc *Connection) SendRaw(message string) error {
	mlen := len(message)
	buf := make([]byte, mlen+2)
	copy(buf[:mlen], message[:])
	copy(buf[mlen:], "\r\n")
	return irc.sendInternal(buf)
}

// SendBatch sends a group of messages as an IRCv3 client batch, adding
// BATCH start and end messages and an appropriate batch tag.
func (irc *Connection) SendBatch(msgs []ircmsg.Message, tags map[string]string, batchType string, batchParams ...string) error {
	combinedMsg, err := irc.composeClientBatch("", msgs, tags, batchType, batchParams...)
	if err != nil {
		return err
	}
	return irc.sendInternal(combinedMsg)
}

// SendBatchWithLabel sends a group of messages as an IRCv3 client batch,
// additionally using the IRCv3 labeled-response specification to collect
// the response.
func (irc *Connection) SendBatchWithLabel(callback func(*Batch), msgs []ircmsg.Message, tags map[string]string, batchType string, batchParams ...string) (err error) {
	if !irc.labelNegotiated() {
		return CapabilityNotNegotiated
	}
	label := irc.registerLabel(callback)
	defer func() {
		if err != nil {
			irc.unregisterLabel(label)
		}
	}()

	combinedMsg, err := irc.composeClientBatch(label, msgs, tags, batchType, batchParams...)
	if err != nil {
		return err
	}

	return irc.sendInternal(combinedMsg)
}

// GetLabeledResponseForBatch sends a group of messages as an IRCv3 client batch,
// using the IRCv3 labeled-response specification, then synchronously waits for
// the response.
func (irc *Connection) GetLabeledResponseForBatch(msgs []ircmsg.Message, tags map[string]string, batchType string, batchParams ...string) (batch *Batch, err error) {
	done := make(chan empty)
	err = irc.SendBatchWithLabel(func(b *Batch) {
		batch = b
		close(done)
	}, msgs, tags, batchType, batchParams...)
	if err != nil {
		return
	}
	<-done
	if batch == nil {
		err = NoLabeledResponse
	}
	return
}

func (irc *Connection) composeClientBatch(label string, msgs []ircmsg.Message, tags map[string]string, batchType string, batchParams ...string) (result []byte, err error) {
	var buf bytes.Buffer
	// only one client batch can be in flight at a time,
	// so we can use a constant batch ID of 1
	batchStartParams := []string{"+1", batchType}
	batchStartParams = append(batchStartParams, batchParams...)
	batchStart := ircmsg.MakeMessage(nil, "", "BATCH", batchStartParams...)
	for k, v := range tags {
		batchStart.SetTag(k, v)
	}
	if label != "" {
		batchStart.SetTag("label", label)
	}
	b, err := batchStart.LineBytesStrict(true, irc.MaxLineLen)
	if err != nil {
		return nil, err
	}
	buf.Write(b)

	for _, msg := range msgs {
		msg.SetTag("batch", "1")
		b, err = msg.LineBytesStrict(true, irc.MaxLineLen)
		if err != nil {
			return nil, err
		}
		buf.Write(b)
	}

	buf.WriteString("BATCH -1\r\n")

	return buf.Bytes(), nil
}

// Join joins a channel.
func (irc *Connection) Join(channel string) error {
	return irc.Send("JOIN", channel)
}

// Part leaves a channel.
func (irc *Connection) Part(channel string) error {
	return irc.Send("PART", channel)
}

// Notice sends an IRC NOTICE to a given target.
func (irc *Connection) Notice(target, message string) error {
	return irc.Send("NOTICE", target, message)
}

// Noticef sends an IRC NOTICE, composed via format string.
func (irc *Connection) Noticef(target, format string, a ...interface{}) error {
	return irc.Notice(target, fmt.Sprintf(format, a...))
}

// Privmsg sends an IRC PRIVMSG to a given target.
func (irc *Connection) Privmsg(target, message string) error {
	return irc.Send("PRIVMSG", target, message)
}

// Privmsgf sends an IRC PRIVMSG, composed via format string.
func (irc *Connection) Privmsgf(target, format string, a ...interface{}) error {
	return irc.Privmsg(target, fmt.Sprintf(format, a...))
}

// Action sends a CTCP ACTION (the /me command in a typical client) to a given target.
func (irc *Connection) Action(target, message string) error {
	return irc.Privmsg(target, fmt.Sprintf("\001ACTION %s\001", message))
}

// Actionf sends a CTCP ACTION, composed via format string.
func (irc *Connection) Actionf(target, format string, a ...interface{}) error {
	return irc.Action(target, fmt.Sprintf(format, a...))
}

// SetNick changes the preferred nickname for the connection (the server may
// accept or deny the request).
func (irc *Connection) SetNick(n string) {
	irc.stateMutex.Lock()
	irc.Nick = n
	irc.stateMutex.Unlock()

	irc.Send("NICK", n)
}

// CurrentNick returns the nickname currently assigned by the server, which
// may differ from the requested nickname. If the handshake is incomplete or
// the connection is disconnected, the empty string is returned.
func (irc *Connection) CurrentNick() string {
	irc.stateMutex.Lock()
	defer irc.stateMutex.Unlock()
	return irc.currentNick
}

func (irc *Connection) getOrRequestUserHost() (currentNick, userHost string) {
	requestUserhost := false
	defer func() {
		if requestUserhost {
			// legacy fallback for learning the userhost
			irc.Send("WHO", currentNick)
		}
	}()

	irc.stateMutex.Lock()
	defer irc.stateMutex.Unlock()

	// query for the userhost if this is the first time we were asked about it
	// and we don't already have it
	if irc.userHost == "" && !irc.userHostRequested {
		requestUserhost = true
		irc.userHostRequested = true
	}

	return irc.currentNick, irc.userHost
}

// Calculate the max message size for PRIVMSG or NOTICE, after accounting
// for protocol overhead incurred when the server relays the message.
// This may be used to ensure sent messages do not exceed the IRC 512 byte
// limit (which would cause them to either be truncated or rejected by the
// ircd). Note that this value is not a strict guarantee because the server
// can change the client's NUH unilaterally at any time; implementations may
// wish to use a more conservative constant maximum instead.
func (irc *Connection) MaxMessageLength(target string) int {
	nick, userHost := irc.getOrRequestUserHost()
	var userhostLen int
	if userHost != "" {
		userhostLen = len(userHost)
	} else {
		userhostLen = 96 // sane default
	}

	// :nick!user@host PRIVMSG #target :payload\r\n
	result := 512
	result -= 1 // :
	result -= len(nick)
	result -= 1 // !
	result -= userhostLen
	result -= 1 // space
	result -= 7 // PRIVMSG
	result -= 1 // space
	result -= len(target)
	result -= 2 // space after target, : for trailing
	// payload goes here
	result -= 2 // \r\n
	return result
}

// Returns the expected or desired nickname for the connection;
// if the real nickname is different, the client will periodically
// attempt to change to this one.
func (irc *Connection) PreferredNick() string {
	irc.stateMutex.Lock()
	defer irc.stateMutex.Unlock()
	return irc.Nick
}

func (irc *Connection) setCurrentNick(nick string) {
	irc.stateMutex.Lock()
	defer irc.stateMutex.Unlock()
	irc.currentNick = nick
}

// Return IRCv3 CAPs actually enabled on the connection, together
// with their values if applicable. The resulting map is shared,
// so do not modify it.
func (irc *Connection) AcknowledgedCaps() (result map[string]string) {
	irc.stateMutex.Lock()
	defer irc.stateMutex.Unlock()
	return irc.capsAcked
}

// Returns the 005 RPL_ISUPPORT tokens sent by the server when the
// connection was initiated, parsed into key-value form as a map.
// The resulting map is shared, so do not modify it.
func (irc *Connection) ISupport() (result map[string]string) {
	irc.stateMutex.Lock()
	defer irc.stateMutex.Unlock()
	// XXX modifications to isupport are not permitted after registration
	return irc.isupport
}

// Returns true if the connection is connected to an IRC server.
func (irc *Connection) Connected() bool {
	return irc.State() == ConnectionActive
}

// Reconnect forces the client to reconnect to the server. It returns
// immediately.
func (irc *Connection) Reconnect() {
	if err := irc.normalizeConfig(); err != nil {
		return
	}

	switch irc.State() {
	case ConnectionNotStarted:
		return // Reconnect() is invalid before initial Connect()
	case ConnectionConnecting, ConnectionReconnecting:
		return // no-op, wait for the ongoing Connect() to complete
	case ConnectionStopping, ConnectionStopped:
		return // no-op, we can't reconnect
	case ConnectionActive, ConnectionSleeping:
		// fall through:
	}

	// halt any existing connection:
	irc.closeEnd()

	// wake up maintenanceLoop if necessary
	select {
	case irc.reconnSig <- empty{}:
	default:
	}
}

func (irc *Connection) closeEnd() {
	irc.stateMutex.Lock()
	defer irc.stateMutex.Unlock()
	irc.closeEndNoMutex()
}

func (irc *Connection) closeEndNoMutex() {
	if !irc.endClosed {
		irc.endClosed = true
		close(irc.end)
		// close the socket to interrupt readLoop, on a separate goroutine
		// so we don't block while holding the mutex
		go func() {
			defer irc.wg.Done()
			irc.socket.Close()
		}()
	}
}

func (irc *Connection) dial() (socket net.Conn, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), irc.Timeout)
	defer cancel()
	socket, err = irc.DialContext(ctx, "tcp", irc.Server)
	if err != nil {
		return
	}
	if !irc.UseTLS {
		return
	}

	// see tls.DialWithDialer
	if irc.TLSConfig == nil {
		irc.TLSConfig = &tls.Config{}
	}
	if irc.TLSConfig.ServerName == "" && !irc.TLSConfig.InsecureSkipVerify {
		host, _, err := net.SplitHostPort(irc.Server)
		if err == nil {
			irc.TLSConfig.ServerName = host
		} else {
			irc.TLSConfig.ServerName = irc.Server
		}
	}
	tlsSocket := tls.Client(socket, irc.TLSConfig)
	err = tlsSocket.HandshakeContext(ctx)
	if err != nil {
		socket.Close()
		return nil, err
	}
	return tlsSocket, nil
}

func (irc *Connection) performConfigNormalization() (err error) {
	irc.stateMutex.Lock()
	defer irc.stateMutex.Unlock()

	// config normalization error is unrecoverable
	defer func() {
		if err != nil {
			irc.connectionState = ConnectionStopped
			close(irc.quitEvent)
		}
	}()

	// these are initialized only once in the lifetime of the Connection object:
	irc.reconnSig = make(chan empty, 1)
	irc.quitEvent = make(chan empty)

	if irc.Server == "" {
		return errors.New("No server provided")
	}
	if len(irc.Nick) == 0 {
		irc.Nick = defaultNick
	}
	if irc.User == "" {
		irc.User = irc.Nick
	}
	if irc.RealName == "" {
		irc.RealName = irc.User
	}
	if irc.Log == nil {
		irc.Log = log.New(os.Stdout, "", log.LstdFlags)
	}
	if irc.KeepAlive <= 0 {
		irc.KeepAlive = 4 * time.Minute
	}
	if irc.Timeout <= 0 {
		irc.Timeout = 1 * time.Minute
	}
	if irc.KeepAlive < irc.Timeout {
		return errors.New("KeepAlive must be at least Timeout")
	}
	if irc.ReconnectFreq <= 0 {
		irc.ReconnectFreq = 2 * time.Minute
	}
	if irc.SASLLogin != "" && irc.SASLPassword != "" {
		irc.UseSASL = true
	}
	if irc.UseSASL {
		// ensure 'sasl' is in the cap list if necessary
		if !sliceContains("sasl", irc.RequestCaps) {
			irc.RequestCaps = append(irc.RequestCaps, "sasl")
		}
	}
	if irc.FetchUserHost {
		// ensure 'chghost' is in the cap list if necessary
		if !sliceContains("chghost", irc.RequestCaps) {
			irc.RequestCaps = append(irc.RequestCaps, "chghost")
		}
	}

	if irc.SASLMech == "" {
		irc.SASLMech = "PLAIN"
	}
	if !(irc.SASLMech == "PLAIN" || irc.SASLMech == "EXTERNAL") {
		return fmt.Errorf("unsupported SASL mechanism %s", irc.SASLMech)
	}
	if irc.MaxLineLen <= 0 {
		irc.MaxLineLen = 512
	}
	if irc.MaxTotalBatchSize <= 0 {
		irc.MaxTotalBatchSize = 8 * 1024 * 1024
	}
	if irc.WriteQueueSize <= 0 {
		irc.WriteQueueSize = 16
	}
	if irc.Version == "" {
		irc.Version = Version
	}
	if irc.DialContext == nil {
		irc.DialContext = (&net.Dialer{}).DialContext
	}

	return nil
}

func (irc *Connection) normalizeConfig() error {
	irc.normalizeOnce.Do(func() {
		irc.normalizeErr = irc.performConfigNormalization()
		if irc.normalizeErr == nil {
			irc.setupCallbacks()
		}
	})
	return irc.normalizeErr
}

// Connect connects to the configured IRC server. If the returned error is nil,
// the connection is ready for use and will be maintained (including automatic reconnection)
// until Quit() is called. If it is not nil, the connection is inactive.
func (irc *Connection) Connect() (err error) {
	return irc.connectInternal(ConnectionConnecting)
}

func (irc *Connection) connectInternal(newState ConnectionState) (err error) {
	if err := irc.normalizeConfig(); err != nil {
		return err
	}

	// invariant: after Connect we are in one of two states:
	// (a) success: return nil, socket open, goroutines launched, ready for Loop/Wait()
	//     state is ConnectionActive
	// (b) failure: return error, socket closed, goroutines stopped,
	//     ready for another call to Connect (possibly from Loop)
	//     state is whatever it was previously
	prevState, err := func() (prevState ConnectionState, err error) {
		irc.stateMutex.Lock()
		defer irc.stateMutex.Unlock()

		prevState = irc.connectionState
		switch prevState {
		case ConnectionStopping, ConnectionStopped:
			return prevState, ClientHasQuit
		case ConnectionActive, ConnectionConnecting, ConnectionReconnecting:
			return prevState, connectionAlreadyActive
		case ConnectionNotStarted:
			if newState == ConnectionConnecting {
				irc.connectionState = newState
				return prevState, nil
			} else {
				return prevState, errors.New("Impossible state (reconnect before connect)")
			}
		case ConnectionSleeping:
			if newState == ConnectionReconnecting {
				irc.connectionState = newState
				return prevState, nil
			} else {
				// Connect() instead of Reconnect() during sleep, rejected
				return prevState, connectionAlreadyActive
			}
		default:
			return prevState, ClientHasQuit // impossible
		}
	}()

	if err != nil {
		return err
	}

	socketOpen := false
	connectTime := time.Now()

	// maintain invariant described above:
	defer func() {
		var currentState ConnectionState
		if err == nil {
			startLoop := false
			irc.stateMutex.Lock()
			switch irc.connectionState {
			case ConnectionStopping, ConnectionStopped:
				err = ClientHasQuit // take the error path and shut down
			case ConnectionConnecting, ConnectionReconnecting:
				irc.connectionState = ConnectionActive // success
				// start the maintenance loop iff this is the first successful Connect()
				startLoop = (prevState == ConnectionNotStarted)
			default:
				irc.Log.Printf(
					"impossible state after successful connection (prev=%d, current=%d)",
					prevState, irc.connectionState)
			}
			currentState = irc.connectionState
			irc.stateMutex.Unlock()
			if startLoop {
				go irc.maintenanceLoop(connectTime)
			}
		}
		if err != nil {
			if socketOpen {
				// dial succeeded but we had a layer 7 failure
				irc.closeEnd()
				currentState = irc.waitForStop(prevState)
			} else {
				// dial failed
				irc.stateMutex.Lock()
				switch irc.connectionState {
				case ConnectionStopping, ConnectionStopped:
					irc.connectionState = ConnectionStopped
				default:
					irc.connectionState = prevState
				}
				currentState = irc.connectionState
				irc.stateMutex.Unlock()
			}
		}
		if prevState == ConnectionNotStarted && (currentState == ConnectionStopping || currentState == ConnectionStopped) {
			// maintenanceLoop will never start, so we need to unblock Wait() here
			close(irc.quitEvent)
		}
	}()

	if irc.Debug {
		irc.Log.Printf("Connecting to %s (TLS: %t)\n", irc.Server, irc.UseTLS)
	}

	socket, err := irc.dial()
	if err != nil {
		return err
	}

	if irc.Debug {
		irc.Log.Printf("Connected to %s (%s)\n", irc.Server, socket.RemoteAddr())
	}

	// reset all connection state
	irc.stateMutex.Lock()
	irc.socket = socket
	irc.end = make(chan empty)
	irc.endClosed = false
	irc.pwrite = make(chan []byte, irc.WriteQueueSize)
	// 2 goroutines, plus the asynchronous socket close:
	irc.wg.Add(3)
	irc.capsChan = make(chan capResult, len(irc.RequestCaps))
	irc.saslChan = make(chan saslResult, 1)
	irc.welcomeChan = make(chan empty)
	irc.registered = false
	irc.lastError = nil
	irc.pingSent = ""
	irc.currentNick = ""
	irc.userHost = ""
	irc.userHostRequested = false
	irc.isupportPartial = make(map[string]string)
	irc.isupport = nil
	irc.capsAcked = make(map[string]string)
	irc.capsAdvertised = nil
	state := irc.connectionState
	irc.stateMutex.Unlock()
	irc.batchMutex.Lock()
	irc.batches = make(map[string]*batchInProgress)
	irc.totalBatchSize = 0
	irc.labelCallbacks = make(map[int64]pendingLabel)
	irc.labelCounter = 0
	irc.batchMutex.Unlock()

	go irc.readLoop()
	go irc.writeLoop()

	socketOpen = true

	if state == ConnectionStopping || state == ConnectionStopped {
		// if we got Quit() during dial, stop here without doing the layer 7 handshake;
		// eventually we may make dial itself preemptible via context cancellation
		return ClientHasQuit
	}

	err = irc.performHandshake()
	return err
}

func (irc *Connection) performHandshake() error {
	if len(irc.WebIRC) > 0 {
		irc.Send("WEBIRC", irc.WebIRC...)
	}

	if len(irc.Password) > 0 {
		irc.Send("PASS", irc.Password)
	}

	remainingCaps := len(irc.RequestCaps)
	capsRequested := remainingCaps != 0
	acknowledgedCaps := make([]string, 0, remainingCaps)

	if capsRequested {
		// get all CAP values if available
		irc.Send("CAP", "LS", "302")
		// then blindly request all CAPs we know about
		for _, capab := range irc.RequestCaps {
			irc.Send("CAP", "REQ", capab)
		}
	}
	// then send NICK and USER
	irc.Send("NICK", irc.PreferredNick())
	irc.Send("USER", irc.User, "s", "e", irc.RealName)

	// Three possibilities:
	// 1. The server doesn't support CAP or we didn't request any CAPs;
	// the server will terminate registration with NICK/USER and send 001
	// 2. The server supports CAPs and will start sending CAP LS / ACK / NAK replies
	// 3. We time out before getting an intelligible response, so set a timer:
	timer := time.NewTimer(irc.Timeout)
	defer timer.Stop()

CAPLOOP:
	for {
		select {
		case result := <-irc.capsChan:
			remainingCaps--
			if result.ack {
				acknowledgedCaps = append(acknowledgedCaps, result.capName)
			}
			if remainingCaps <= 0 {
				break CAPLOOP // got ACK or NAK for all our CAPs
			}
		case <-irc.welcomeChan:
			break CAPLOOP // server does not support CAP
		case <-timer.C:
			return ServerTimedOut
		case <-irc.end:
			return ServerDisconnected
		}
	}

	irc.processAckedCaps(acknowledgedCaps)

	saslSucceeded := false
	var saslError error

	if irc.UseSASL && sliceContains("sasl", acknowledgedCaps) {
		// perform SASL and wait synchronously for the result;
		// we must wait because on conventional ircd+services stacks,
		// CAP END will terminate an in-progress SASL session
		irc.Send("AUTHENTICATE", irc.SASLMech)

		select {
		case res := <-irc.saslChan:
			saslSucceeded = !res.Failed
			if !saslSucceeded {
				saslError = res.Err
			}
		case <-timer.C:
			// technically we could proceed, but our view of the
			// registration timeout has expired
			return ServerTimedOut
		case <-irc.end:
			return ServerDisconnected
		}
	}

	if irc.UseSASL && !irc.SASLOptional && !saslSucceeded {
		if saslError == nil {
			saslError = SASLFailed
		}
		irc.SendRaw("QUIT")
		return saslError
	}

	// if we did successful CAP negotiation with the server
	// then we need CAP END to terminate registration
	if capsRequested && remainingCaps <= 0 {
		irc.Send("CAP", "END")
	}

	// wait for registration to complete, or fail
	select {
	case <-irc.welcomeChan:
		return nil
	case <-timer.C:
		return ServerTimedOut
	case <-irc.end:
		return ServerDisconnected
	}
}
