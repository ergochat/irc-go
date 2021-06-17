package ircevent

import (
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/ergochat/irc-go/ircmsg"
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

func (irc *Connection) setupSASLCallbacks() {
	irc.AddCallback("AUTHENTICATE", func(e ircmsg.Message) {
		str := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s\x00%s\x00%s", irc.SASLLogin, irc.SASLLogin, irc.SASLPassword)))
		irc.Send("AUTHENTICATE", str)
	})

	irc.AddCallback(RPL_LOGGEDOUT, func(e ircmsg.Message) {
		irc.SendRaw("CAP END")
		irc.SendRaw("QUIT")
		irc.submitSASLResult(saslResult{true, errors.New(e.Params[1])})
	})

	irc.AddCallback(ERR_NICKLOCKED, func(e ircmsg.Message) {
		irc.SendRaw("CAP END")
		irc.SendRaw("QUIT")
		irc.submitSASLResult(saslResult{true, errors.New(e.Params[1])})
	})

	irc.AddCallback(RPL_SASLSUCCESS, func(e ircmsg.Message) {
		irc.submitSASLResult(saslResult{false, nil})
	})

	irc.AddCallback(ERR_SASLFAIL, func(e ircmsg.Message) {
		irc.SendRaw("CAP END")
		irc.SendRaw("QUIT")
		irc.submitSASLResult(saslResult{true, errors.New(e.Params[1])})
	})

	// this could potentially happen with auto-login via certfp?
	irc.AddCallback(ERR_SASLALREADY, func(e ircmsg.Message) {
		irc.submitSASLResult(saslResult{false, nil})
	})
}
