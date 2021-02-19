package ircevent

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
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

// Check if a space-separated list of arguments contains a value.
func listContains(list string, value string) bool {
	for _, arg_name := range strings.Split(strings.TrimSpace(list), " ") {
		if arg_name == value {
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
	irc.AddCallback("CAP", func(e Event) {
		if len(e.Params) == 3 {
			if e.Params[1] == "LS" {
				if !listContains(e.Params[2], "sasl") {
					irc.submitSASLResult(saslResult{true, errors.New("no SASL capability " + e.Params[2])})
				}
			}
			if e.Params[1] == "ACK" && listContains(e.Params[2], "sasl") {
				irc.Send("AUTHENTICATE", irc.SASLMech)
			}
		}
	})

	irc.AddCallback("AUTHENTICATE", func(e Event) {
		str := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s\x00%s\x00%s", irc.SASLLogin, irc.SASLLogin, irc.SASLPassword)))
		irc.Send("AUTHENTICATE", str)
	})

	irc.AddCallback("901", func(e Event) {
		irc.SendRaw("CAP END")
		irc.SendRaw("QUIT")
		irc.submitSASLResult(saslResult{true, errors.New(e.Params[1])})
	})

	irc.AddCallback("902", func(e Event) {
		irc.SendRaw("CAP END")
		irc.SendRaw("QUIT")
		irc.submitSASLResult(saslResult{true, errors.New(e.Params[1])})
	})

	irc.AddCallback("903", func(e Event) {
		irc.submitSASLResult(saslResult{false, nil})
	})

	irc.AddCallback("904", func(e Event) {
		irc.SendRaw("CAP END")
		irc.SendRaw("QUIT")
		irc.submitSASLResult(saslResult{true, errors.New(e.Params[1])})
	})
}
