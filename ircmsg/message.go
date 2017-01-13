// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircmsg

import (
	"errors"
	"strings"
)

var (
	// ErrorLineIsEmpty indicates that the given IRC line was empty.
	ErrorLineIsEmpty = errors.New("Line is empty")
)

// IrcMessage represents an IRC message, as defined by the RFCs and as
// extended by the IRCv3 Message Tags specification with the introduction
// of message tags.
type IrcMessage struct {
	Tags    map[string]TagValue
	Prefix  string
	Command string
	Params  []string
}

// ParseLine creates and returns an IrcMessage from the given IRC line.
//
// Quirks:
//
//   The RFCs say that last parameters with no characters MUST be a trailing.
//   IE, they need to be prefixed with ":". We disagree with that and handle
//   incoming last empty parameters whether they are trailing or ordinary
//   parameters. However, we do follow that rule when emitting lines.
func ParseLine(line string) (IrcMessage, error) {
	return parseLine(line, 0, 0, false)
}

// ParseLineMaxLen creates and returns an IrcMessage from the given IRC line,
// taking the maximum length into account and truncating the message as appropriate.
//
// Quirks:
//
//   The RFCs say that last parameters with no characters MUST be a trailing.
//   IE, they need to be prefixed with ":". We disagree with that and handle
//   incoming last empty parameters whether they are trailing or ordinary
//   parameters. However, we do follow that rule when emitting lines.
func ParseLineMaxLen(line string, maxlenTags, maxlenRest int) (IrcMessage, error) {
	return parseLine(line, maxlenTags, maxlenRest, true)
}

// parseLine does the actual line parsing for the above user-facing functions.
func parseLine(line string, maxlenTags, maxlenRest int, useMaxLen bool) (IrcMessage, error) {
	line = strings.Trim(line, "\r\n")
	var ircmsg IrcMessage

	if len(line) < 1 {
		return ircmsg, ErrorLineIsEmpty
	}

	// tags
	ircmsg.Tags = make(map[string]TagValue)
	if line[0] == '@' {
		splitLine := strings.SplitN(line, " ", 2)
		if len(splitLine) < 2 {
			return ircmsg, ErrorLineIsEmpty
		}
		tags := splitLine[0][1:]
		line = strings.TrimLeft(splitLine[1], " ")

		if len(line) < 1 {
			return ircmsg, ErrorLineIsEmpty
		}

		// truncate if desired
		if useMaxLen && len(tags) > maxlenTags {
			tags = tags[:maxlenTags]
		}

		for _, fulltag := range strings.Split(tags, ";") {
			var name string
			var val TagValue
			if strings.Contains(fulltag, "=") {
				val.HasValue = true
				splittag := strings.SplitN(fulltag, "=", 2)
				name = splittag[0]
				val.Value = UnescapeTagValue(splittag[1])
			} else {
				name = fulltag
				val.HasValue = false
			}

			ircmsg.Tags[name] = val
		}
	}

	// truncate if desired
	if useMaxLen && len(line) > maxlenRest {
		line = line[:maxlenRest]
	}

	// prefix
	if line[0] == ':' {
		splitLine := strings.SplitN(line, " ", 2)
		if len(splitLine) < 2 {
			return ircmsg, ErrorLineIsEmpty
		}
		ircmsg.Prefix = splitLine[0][1:]
		line = strings.TrimLeft(splitLine[1], " ")
	}

	if len(line) < 1 {
		return ircmsg, ErrorLineIsEmpty
	}

	// command
	splitLine := strings.SplitN(line, " ", 2)
	if len(splitLine[0]) == 0 {
		return ircmsg, ErrorLineIsEmpty
	}
	ircmsg.Command = strings.ToUpper(splitLine[0])
	if len(splitLine) > 1 {
		line = strings.TrimLeft(splitLine[1], " ")

		// parameters
		for {
			// handle trailing
			if len(line) > 0 && line[0] == ':' {
				ircmsg.Params = append(ircmsg.Params, line[1:])
				break
			}

			// regular params
			splitLine := strings.SplitN(line, " ", 2)
			if len(splitLine[0]) > 0 {
				ircmsg.Params = append(ircmsg.Params, splitLine[0])
			}

			if len(splitLine) > 1 {
				line = strings.TrimLeft(splitLine[1], " ")
			} else {
				break
			}
		}
	}

	return ircmsg, nil
}

// MakeMessage provides a simple way to create a new IrcMessage.
func MakeMessage(tags *map[string]TagValue, prefix string, command string, params ...string) IrcMessage {
	var ircmsg IrcMessage

	ircmsg.Tags = make(map[string]TagValue)
	if tags != nil {
		for tag, value := range *tags {
			ircmsg.Tags[tag] = value
		}
	}

	ircmsg.Prefix = prefix
	ircmsg.Command = command
	ircmsg.Params = params

	return ircmsg
}

// Line returns a sendable line created from an IrcMessage.
func (ircmsg *IrcMessage) Line() (string, error) {
	return ircmsg.line(0, 0, false)
}

// LineMaxLen returns a sendable line created from an IrcMessage, limited by maxlen.
func (ircmsg *IrcMessage) LineMaxLen(maxlenTags, maxlenRest int) (string, error) {
	return ircmsg.line(maxlenTags, maxlenRest, true)
}

// line returns a sendable line created from an IrcMessage.
func (ircmsg *IrcMessage) line(maxlenTags, maxlenRest int, useMaxLen bool) (string, error) {
	var tags, rest, line string

	if len(ircmsg.Command) < 1 {
		return "", errors.New("irc: IRC messages MUST have a command")
	}

	if len(ircmsg.Tags) > 0 {
		tags += "@"

		for tag, val := range ircmsg.Tags {
			tags += tag

			if val.HasValue {
				tags += "="
				tags += EscapeTagValue(val.Value)
			}

			tags += ";"
		}
		// TODO: this is ugly, but it works for now
		tags = strings.TrimSuffix(tags, ";")

		// truncate if desired
		if useMaxLen && len(tags) > maxlenTags {
			tags = tags[:maxlenTags]
		}

		tags += " "
	}

	if len(ircmsg.Prefix) > 0 {
		rest += ":"
		rest += ircmsg.Prefix
		rest += " "
	}

	rest += ircmsg.Command

	if len(ircmsg.Params) > 0 {
		for i, param := range ircmsg.Params {
			rest += " "
			if len(param) < 1 || strings.Contains(param, " ") || param[0] == ':' {
				if i != len(ircmsg.Params)-1 {
					return "", errors.New("irc: Cannot have an empty param, a param with spaces, or a param that starts with ':' before the last parameter")
				}
				rest += ":"
			}
			rest += param
		}
	}

	// truncate if desired
	// -2 for \r\n
	if useMaxLen && len(rest) > maxlenRest-2 {
		rest = rest[:maxlenRest-2]
	}

	line = tags + rest + "\r\n"

	return line, nil
}
