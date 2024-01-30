package ircevent

import (
	"bytes"
	"encoding/base64"
	"errors"

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

func splitSaslResponse(raw []byte) (result []string) {
	// https://ircv3.net/specs/extensions/sasl-3.1#the-authenticate-command
	// "The response is encoded in Base64 (RFC 4648), then split to 400-byte chunks,
	// and each chunk is sent as a separate AUTHENTICATE command. Empty (zero-length)
	// responses are sent as AUTHENTICATE +. If the last chunk was exactly 400 bytes
	// long, it must also be followed by AUTHENTICATE + to signal end of response."

	if len(raw) == 0 {
		return []string{"+"}
	}

	response := base64.StdEncoding.EncodeToString(raw)
	lastLen := 0
	for len(response) > 0 {
		// TODO once we require go 1.21, this can be: lastLen = min(len(response), 400)
		lastLen = len(response)
		if lastLen > 400 {
			lastLen = 400
		}
		result = append(result, response[:lastLen])
		response = response[lastLen:]
	}

	if lastLen == 400 {
		result = append(result, "+")
	}

	return result
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
			for _, resp := range splitSaslResponse(irc.composeSaslPlainResponse()) {
				irc.Send("AUTHENTICATE", resp)
			}
		case "EXTERNAL":
			irc.Send("AUTHENTICATE", "+")
		default:
			// impossible, nothing to do
		}
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
