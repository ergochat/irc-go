package ircevent

import (
	"bytes"
	"fmt"

	"github.com/ergochat/irc-go/ircmsg"
	"github.com/ergochat/irc-go/ircutils"
)

type saslResult struct {
	Failed bool
	Err    error
}

func sliceContains(str string, list []string) bool {
	for _, x := range list {
		if x == str {
			return true
		}
	}
	return false
}

func (irc *Connection) submitSASLResult(r saslResult) {
	select {
	case irc.saslChan <- r:
	default:
	}
}

func (irc *Connection) composeSaslPlainResponse() []byte {
	var buf bytes.Buffer
	buf.WriteString(irc.SASLLogin) // optional authzid, included for compatibility
	buf.WriteByte('\x00')
	buf.WriteString(irc.SASLLogin) // authcid
	buf.WriteByte('\x00')
	buf.WriteString(irc.SASLPassword) // passwd
	return buf.Bytes()
}

func (irc *Connection) setupSASLCallbacks() {
	irc.AddCallback("AUTHENTICATE", func(e ircmsg.Message) {
		switch irc.SASLMech {
		case "PLAIN":
			for _, resp := range ircutils.EncodeSASLResponse(irc.composeSaslPlainResponse()) {
				irc.Send("AUTHENTICATE", resp)
			}
		case "EXTERNAL":
			irc.Send("AUTHENTICATE", "+")
		default:
			// impossible, nothing to do
		}
	})

	irc.AddCallback(RPL_LOGGEDOUT, func(e ircmsg.Message) {
		irc.submitSASLResult(saslResult{true, saslError(&e)})
	})

	irc.AddCallback(ERR_NICKLOCKED, func(e ircmsg.Message) {
		irc.submitSASLResult(saslResult{true, saslError(&e)})
	})

	irc.AddCallback(RPL_SASLSUCCESS, func(e ircmsg.Message) {
		irc.submitSASLResult(saslResult{false, nil})
	})

	irc.AddCallback(ERR_SASLFAIL, func(e ircmsg.Message) {
		irc.submitSASLResult(saslResult{true, saslError(&e)})
	})

	// this could potentially happen with auto-login via certfp?
	irc.AddCallback(ERR_SASLALREADY, func(e ircmsg.Message) {
		irc.submitSASLResult(saslResult{false, nil})
	})
}

func saslError(e *ircmsg.Message) error {
	return fmt.Errorf("%s: %s", e.Command, lastParam(e))
}
