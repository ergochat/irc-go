// written by Daniel Oaks <daniel@danieloaks.net>
// released under the CC0 Public Domain license

package girc

import (
	"errors"
	"strings"
)

// TagValue represents the value of a tag. This is because tags may have
// no value at all or just an empty value, and this can represent both
// using the HasValue attribute.
type TagValue struct {
	HasValue bool
	Value    string
}

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
func ParseLine(line string) (IrcMessage, error) {
	line = strings.Trim(line, "\r\n")
	var ircmsg IrcMessage

	// tags
	ircmsg.Tags = make(map[string]TagValue)
	if line[0] == '@' {
		splitLine := strings.SplitN(line, " ", 2)
		tags := splitLine[0][1:]
		line = strings.TrimLeft(splitLine[1], " ")

		for _, fulltag := range strings.Split(tags, ";") {
			var name string
			var val TagValue
			if strings.Contains(fulltag, "=") {
				val.HasValue = true
				splittag := strings.SplitN(fulltag, "=", 2)
				name = splittag[0]
				val.Value = splittag[1]
				// TODO: unescape values
			} else {
				name = fulltag
				val.HasValue = false
			}

			ircmsg.Tags[name] = val
		}
	}

	// prefix
	if line[0] == ':' {
		splitLine := strings.SplitN(line, " ", 2)
		ircmsg.Prefix = splitLine[0][1:]
		line = strings.TrimLeft(splitLine[1], " ")
	}

	// command
	splitLine := strings.SplitN(line, " ", 2)
	ircmsg.Command = strings.ToUpper(splitLine[0])
	line = strings.TrimLeft(splitLine[1], " ")

	// parameters
	for {
		// handle trailing
		if line[0] == ':' {
			ircmsg.Params = append(ircmsg.Params, line[1:])
			break
		}

		// regular params
		splitLine := strings.SplitN(line, " ", 2)
		ircmsg.Params = append(ircmsg.Params, splitLine[0])

		if len(splitLine) > 1 {
			line = strings.TrimLeft(splitLine[1], " ")
		} else {
			break
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

// NoTagValue returns an empty TagValue.
func NoTagValue() TagValue {
	var tag TagValue
	tag.HasValue = false
	return tag
}

// MakeTagValue returns a TagValue with a defined value.
func MakeTagValue(value string) TagValue {
	var tag TagValue
	tag.HasValue = true
	tag.Value = value
	return tag
}

// MakeTags simplifies tag creation for new messages.
//
// For example: MakeTags("intent", "PRIVMSG", "account", "bunny", "noval", nil)
func MakeTags(values ...interface{}) *map[string]TagValue {
	var tags map[string]TagValue

	tags = make(map[string]TagValue)

	for len(values) > 1 {
		tag := values[0].(string)
		value := values[1]
		var val TagValue

		if value == nil {
			val = NoTagValue()
		} else {
			val = MakeTagValue(value.(string))
		}

		tags[tag] = val

		values = values[2:]
	}

	return &tags
}

// Line returns a sendable line created from an IrcMessage.
func (ircmsg *IrcMessage) Line() (string, error) {
	var line string

	if len(ircmsg.Command) < 1 {
		return "", errors.New("irc: IRC messages MUST have a command")
	}

	if len(ircmsg.Tags) > 0 {
		line += "@"

		for tag, val := range ircmsg.Tags {
			line += tag

			if val.HasValue {
				line += "="
				line += val.Value
				// TODO: escape values
			}

			line += ";"
		}
		// TODO: this is ugly, but it works for now
		line = strings.TrimSuffix(line, ";")

		line += " "
	}

	if len(ircmsg.Prefix) > 0 {
		line += ":"
		line += ircmsg.Prefix
		line += " "
	}

	line += ircmsg.Command

	if len(ircmsg.Params) > 0 {
		for i, param := range ircmsg.Params {
			line += " "
			if strings.Contains(param, " ") || len(param) < 1 {
				if i != len(ircmsg.Params)-1 {
					return "", errors.New("irc: Cannot have a param with spaces or an empty param before the last parameter")
				}
				line += ":"
			}
			line += param
		}
	}

	line += "\r\n"

	return line, nil
}
