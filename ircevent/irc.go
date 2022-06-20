// Copyright 2009 Thomas Jager <mail@jager.no>  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Here's the concurrency design of this project (largely unchanged from thoj/go-ircevent):
Connect() spawns 3 goroutines (readLoop, writeLoop, pingLoop). The client then
calls Loop(), which monitors their state. Loop() will wait for them
to make a clean stop and then run another Connect(). The system can be
interrupted asynchronously by sending a message, e.g, with Privmsg(), or by
calling Reconnect() (which disconnects and forces a reconnection), or by calling
Quit(), which sends QUIT to the server and will eventually stop the Loop().

The stop mechanism is to close the (*Connection).end channel (which is only closed,
never sent-on normally), so every blocking operation in the 3 loops must also
select on `end` to make sure it stops in a timely fashion.
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

	writeQueueSize = 10

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

	serverDidNotQuit = errors.New("server did not respond to QUIT")
	clientHasQuit    = errors.New("client has called Quit()")
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
func (irc *Connection) ping() {
	param := fmt.Sprintf("%s%d", keepalivePrefix, time.Now().UnixNano())
	irc.Send("PING", param)
}

// Interpret the PONG from a keepalive ping
func (irc *Connection) recordPong(param string) {
	ts := strings.TrimPrefix(param, keepalivePrefix)
	if ts == param {
		return
	}
	t, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return
	}
	if irc.Debug {
		pong := time.Unix(0, t)
		irc.Log.Printf("Lag: %v\n", time.Since(pong))
	}

	irc.stateMutex.Lock()
	defer irc.stateMutex.Unlock()
	irc.pingSent = false
}

// Read data from a connection. To be used as a goroutine.
func (irc *Connection) readLoop() {
	defer irc.wg.Done()

	defer func() {
		if irc.registered {
			irc.runDisconnectCallbacks()
		}
	}()

	msgChan := make(chan string)
	errChan := make(chan error)
	go readMsgLoop(irc.socket, irc.MaxLineLen, msgChan, errChan, irc.end)

	lastExpireCheck := time.Now()

	for {
		select {
		case <-irc.end:
			return
		case msg := <-msgChan:
			if irc.Debug {
				irc.Log.Printf("<-- %s\n", strings.TrimSpace(msg))
			}

			parsedMsg, err := ircmsg.ParseLine(msg)
			if err == nil {
				irc.runCallbacks(parsedMsg)
			} else {
				irc.Log.Printf("invalid message from server: %v\n", err)
			}
		case err := <-errChan:
			irc.setError(err)
			return
		}

		if irc.batchNegotiated() && time.Since(lastExpireCheck) > irc.Timeout {
			irc.expireBatches(false)
			lastExpireCheck = time.Now()
		}
	}
}

func readMsgLoop(socket net.Conn, maxLineLen int, msgChan chan string, errChan chan error, end chan empty) {
	var reader ircreader.Reader
	reader.Initialize(socket, 1024, maxLineLen+maxlenTags)
	for {
		msgBytes, err := reader.ReadLine()
		if err == nil {
			select {
			case msgChan <- string(msgBytes):
			case <-end:
				return
			}
		} else {
			select {
			case errChan <- err:
			case <-end:
			}
			return
		}
	}
}

// Loop to write to a connection. To be used as a goroutine.
func (irc *Connection) writeLoop() {
	defer irc.wg.Done()

	for {
		select {
		case <-irc.end:
			return
		case b := <-irc.pwrite:
			if len(b) == 0 {
				continue
			}

			if irc.Debug {
				irc.Log.Printf("--> %s\n", bytes.TrimSpace(b))
			}

			if irc.Timeout != 0 {
				irc.socket.SetWriteDeadline(time.Now().Add(irc.Timeout))
			}
			_, err := irc.socket.Write(b)
			if irc.Timeout != 0 {
				irc.socket.SetWriteDeadline(time.Time{})
			}
			if err != nil {
				irc.setError(err)
				return
			}
		}
	}
}

// check the status of the connection and take appropriate action
func (irc *Connection) processTick(tick int) {
	var err error
	var shouldPing, shouldRenick bool

	defer func() {
		if err != nil {
			irc.setError(err)
			return
		}
		if shouldPing {
			irc.ping()
		}
		if shouldRenick {
			irc.Send("NICK", irc.PreferredNick())
		}
	}()

	irc.stateMutex.Lock()
	defer irc.stateMutex.Unlock()

	// XXX: handle the server ignoring QUIT
	if irc.quit && time.Since(irc.quitAt) >= irc.Timeout {
		err = serverDidNotQuit
		return
	}
	if irc.pingSent {
		// unacked PING is fatal
		err = ServerTimedOut
		return
	}
	pingModulus := int(irc.KeepAlive / irc.Timeout)
	if tick%pingModulus == 0 {
		shouldPing = true
		irc.pingSent = true
		if irc.currentNick != irc.Nick {
			shouldRenick = true
		}
	}
	return
}

// handles all periodic tasks for the connection:
// 1. sending PING approximately every KeepAlive seconds, monitoring for PONG
// 2. fixing up NICK if we didn't get our preferred one
func (irc *Connection) pingLoop() {
	ticker := time.NewTicker(irc.Timeout)

	defer func() {
		irc.wg.Done()
		ticker.Stop()
	}()

	tick := 0
	for {
		select {
		case <-irc.end:
			return
		case <-ticker.C:
			tick++
			irc.processTick(tick)
		}
	}
}

func (irc *Connection) isQuitting() bool {
	irc.stateMutex.Lock()
	defer irc.stateMutex.Unlock()
	return irc.quit
}

// Main loop to control the connection.
func (irc *Connection) Loop() {
	var lastReconnect time.Time
	for {
		irc.waitForStop()

		if irc.isQuitting() {
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
				t.Stop()
			}
		}

		lastReconnect = time.Now()
		err := irc.Connect()
		if err != nil {
			// we are still stopped, the stop checks will return immediately
			irc.Log.Printf("Error while reconnecting: %s\n", err)
		}
	}
}

// wait for all goroutines to stop. XXX: this is not safe for concurrent
// use, call only from Connect() and Loop() (which will be on the same
// client goroutine)
func (irc *Connection) waitForStop() {
	<-irc.end
	irc.wg.Wait() // wait for readLoop and pingLoop to terminate fully

	if irc.socket != nil {
		irc.socket.Close()
	}

	irc.expireBatches(true)
}

// Quit the current connection and disconnect from the server
// RFC 1459 details: https://tools.ietf.org/html/rfc1459#section-4.1.6
func (irc *Connection) Quit() {
	quitMessage := irc.QuitMessage
	if quitMessage == "" {
		quitMessage = irc.Version
	}

	now := time.Now()
	irc.stateMutex.Lock()
	irc.quit = true
	irc.quitAt = now
	irc.stateMutex.Unlock()

	// the server will respond to this by closing our connection;
	// if it doesn't, pingLoop will eventually notice and close it
	irc.Send("QUIT", quitMessage)
}

func (irc *Connection) sendInternal(b []byte) (err error) {
	// XXX ensure that (end, pwrite) are from the same instantiation of Connect;
	// invocations of this function from callbacks originating in readLoop
	// do not need this synchronization (indeed they cannot occur at a time when
	// `end` is closed), but invocations from outside do (even though the race window
	// is very small).
	irc.stateMutex.Lock()
	running := irc.running
	end := irc.end
	pwrite := irc.pwrite
	irc.stateMutex.Unlock()

	if !running {
		return ClientDisconnected
	}

	select {
	case pwrite <- b:
		return nil
	case <-end:
		return ClientDisconnected
	}
}

// Send a built ircmsg.Message.
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

// Send an IRC message with tags.
func (irc *Connection) SendWithTags(tags map[string]string, command string, params ...string) error {
	return irc.SendIRCMessage(ircmsg.MakeMessage(tags, "", command, params...))
}

// Send an IRC message without tags.
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

// Use the connection to join a given channel.
// RFC 1459 details: https://tools.ietf.org/html/rfc1459#section-4.2.1
func (irc *Connection) Join(channel string) error {
	return irc.Send("JOIN", channel)
}

// Leave a given channel.
// RFC 1459 details: https://tools.ietf.org/html/rfc1459#section-4.2.2
func (irc *Connection) Part(channel string) error {
	return irc.Send("PART", channel)
}

// Send a notification to a nickname. This is similar to Privmsg but must not receive replies.
// RFC 1459 details: https://tools.ietf.org/html/rfc1459#section-4.4.2
func (irc *Connection) Notice(target, message string) error {
	return irc.Send("NOTICE", target, message)
}

// Send a formated notification to a nickname.
// RFC 1459 details: https://tools.ietf.org/html/rfc1459#section-4.4.2
func (irc *Connection) Noticef(target, format string, a ...interface{}) error {
	return irc.Notice(target, fmt.Sprintf(format, a...))
}

// Send (private) message to a target (channel or nickname).
// RFC 1459 details: https://tools.ietf.org/html/rfc1459#section-4.4.1
func (irc *Connection) Privmsg(target, message string) error {
	return irc.Send("PRIVMSG", target, message)
}

// Send formated string to specified target (channel or nickname).
func (irc *Connection) Privmsgf(target, format string, a ...interface{}) error {
	return irc.Privmsg(target, fmt.Sprintf(format, a...))
}

// Send (action) message to a target (channel or nickname).
// No clear RFC on this one...
func (irc *Connection) Action(target, message string) error {
	return irc.Privmsg(target, fmt.Sprintf("\001ACTION %s\001", message))
}

// Send formatted (action) message to a target (channel or nickname).
func (irc *Connection) Actionf(target, format string, a ...interface{}) error {
	return irc.Action(target, fmt.Sprintf(format, a...))
}

// Set (new) nickname.
// RFC 1459 details: https://tools.ietf.org/html/rfc1459#section-4.1.2
func (irc *Connection) SetNick(n string) {
	irc.stateMutex.Lock()
	irc.Nick = n
	irc.stateMutex.Unlock()

	irc.Send("NICK", n)
}

// Determine nick currently used with the connection.
func (irc *Connection) CurrentNick() string {
	irc.stateMutex.Lock()
	defer irc.stateMutex.Unlock()
	return irc.currentNick
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
	irc.stateMutex.Lock()
	defer irc.stateMutex.Unlock()
	return irc.running
}

// Reconnect forces the client to reconnect to the server.
// TODO try to ensure buffered messages are sent?
func (irc *Connection) Reconnect() {
	irc.closeEnd()
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
	if irc.running {
		irc.running = false
		close(irc.end)
	}
}

func (irc *Connection) dial() (socket net.Conn, err error) {
	if irc.DialContext == nil {
		irc.DialContext = (&net.Dialer{}).DialContext
	}
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

// Connect to a given server using the current connection configuration.
// This function also takes care of identification if a password is provided.
// RFC 1459 details: https://tools.ietf.org/html/rfc1459#section-4.1
func (irc *Connection) Connect() (err error) {
	// invariant: after Connect we are in one of two states:
	// (a) success: return nil, socket open, goroutines launched, ready for Loop
	// (b) failure: return error, socket closed, goroutines stopped,
	//     ready for another call to Connect (possibly from Loop)
	err = func() error {
		irc.stateMutex.Lock()
		defer irc.stateMutex.Unlock()

		if irc.quit {
			return clientHasQuit // check this again in case of Quit() while we were asleep
		}

		// mark Server as stopped since there can be an error during connect
		irc.running = false
		irc.socket = nil
		irc.currentNick = ""
		irc.lastError = nil
		irc.pingSent = false

		if irc.Server == "" {
			return errors.New("No server provided")
		}
		if len(irc.Nick) == 0 {
			irc.Nick = defaultNick
		}
		if irc.User == "" {
			irc.User = irc.Nick
		}
		if irc.Log == nil {
			irc.Log = log.New(os.Stdout, "", log.LstdFlags)
		}
		if irc.KeepAlive == 0 {
			irc.KeepAlive = 4 * time.Minute
		}
		if irc.Timeout == 0 {
			irc.Timeout = 1 * time.Minute
		}
		if irc.KeepAlive < irc.Timeout {
			return errors.New("KeepAlive must be at least Timeout")
		}
		if irc.ReconnectFreq == 0 {
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
		if irc.SASLMech == "" {
			irc.SASLMech = "PLAIN"
		}
		if irc.SASLMech != "PLAIN" {
			return errors.New("only SASL PLAIN is supported")
		}
		if irc.MaxLineLen == 0 {
			irc.MaxLineLen = 512
		}
		if irc.Version == "" {
			irc.Version = Version
		}
		// this only runs on first Connect() invocation;
		// unlike other synch primitives it is shared across reconnections:
		if irc.reconnSig == nil {
			irc.reconnSig = make(chan empty)
		}
		return nil
	}()

	if err != nil {
		return err
	}

	irc.setupCallbacks()

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
	irc.running = true
	irc.end = make(chan empty)
	irc.pwrite = make(chan []byte, writeQueueSize)
	irc.wg.Add(3)
	irc.capsChan = make(chan capResult, len(irc.RequestCaps))
	irc.saslChan = make(chan saslResult, 1)
	irc.welcomeChan = make(chan empty, 1)
	irc.registered = false
	irc.isupportPartial = make(map[string]string)
	irc.isupport = nil
	irc.capsAcked = make(map[string]string)
	irc.capsAdvertised = nil
	irc.stateMutex.Unlock()
	irc.batchMutex.Lock()
	irc.batches = make(map[string]batchInProgress)
	irc.labelCallbacks = make(map[int64]pendingLabel)
	irc.labelCounter = 0
	irc.batchMutex.Unlock()

	go irc.readLoop()
	go irc.writeLoop()
	go irc.pingLoop()

	// now we have an open socket and goroutines; we need to clean up
	// if there's a layer 7 failure
	defer func() {
		if err != nil {
			irc.closeEnd()
			irc.waitForStop()
		}
	}()

	if len(irc.WebIRC) > 0 {
		irc.Send("WEBIRC", irc.WebIRC...)
	}

	if len(irc.Password) > 0 {
		irc.Send("PASS", irc.Password)
	}

	err = irc.negotiateCaps()
	if err != nil {
		return err
	}

	realname := irc.User
	if irc.RealName != "" {
		realname = irc.RealName
	}
	irc.Send("NICK", irc.PreferredNick())
	irc.Send("USER", irc.User, "s", "e", realname)
	timeout := time.NewTimer(irc.Timeout)
	defer timeout.Stop()
	select {
	case <-irc.welcomeChan:
	case <-irc.end:
		err = ServerDisconnected
	case <-timeout.C:
		err = ServerTimedOut
	}
	return
}

// Negotiate IRCv3 capabilities
func (irc *Connection) negotiateCaps() error {
	if len(irc.RequestCaps) == 0 {
		return nil
	}

	var acknowledgedCaps []string
	defer func() {
		irc.processAckedCaps(acknowledgedCaps)
	}()

	irc.Send("CAP", "LS", "302")
	defer func() {
		irc.Send("CAP", "END")
	}()

	remaining_caps := len(irc.RequestCaps)

	timer := time.NewTimer(CAPTimeout)
	for remaining_caps > 0 {
		select {
		case result := <-irc.capsChan:
			timer.Stop()
			remaining_caps--
			if result.ack {
				acknowledgedCaps = append(acknowledgedCaps, result.capName)
			}
		case <-timer.C:
			// The server probably doesn't implement CAP LS, which is "normal".
			return nil
		case <-irc.end:
			return ServerDisconnected
		}
	}

	saslError := func(err error) error {
		if !irc.SASLOptional {
			return err
		} else {
			return nil
		}
	}

	if irc.UseSASL {
		if !sliceContains("sasl", acknowledgedCaps) {
			return saslError(SASLFailed)
		} else {
			irc.Send("AUTHENTICATE", irc.SASLMech)
		}
		timeout := time.NewTimer(CAPTimeout)
		defer timeout.Stop()
		select {
		case res := <-irc.saslChan:
			if res.Failed {
				return saslError(res.Err)
			}
		case <-timeout.C:
			// if we expect to be able to SASL, failure to SASL should be treated
			// as a connection error:
			return saslError(SASLFailed)
		case <-irc.end:
			return ServerDisconnected
		}
	}

	return nil
}
