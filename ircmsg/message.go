// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircmsg

import (
	"bytes"
	"errors"
	"strings"
)

var (
	// ErrorLineIsEmpty indicates that the given IRC line was empty.
	ErrorLineIsEmpty = errors.New("Line is empty")
	// ErrorTagsContainsBadChar indicates that the passed tag string contains a space or newline.
	ErrorTagsContainsBadChar = errors.New("Tag string contains bad character (such as a space or newline)")
)

// IrcMessage represents an IRC message, as defined by the RFCs and as
// extended by the IRCv3 Message Tags specification with the introduction
// of message tags.
type IrcMessage struct {
	Tags    map[string]TagValue
	Prefix  string
	Command string
	Params  []string
	// SourceLine represents the original line that constructed this message, when created from ParseLine.
	SourceLine string
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

	ircmsg.SourceLine = line

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

		var err error
		ircmsg.Tags, err = parseTags(tags, maxlenTags, useMaxLen)
		if err != nil {
			return ircmsg, err
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
	bytes, err := ircmsg.line(0, 0, false)
	return string(bytes), err
}

// LineBytes returns a sendable line, as a []byte, created from an IrcMessage.
func (ircmsg *IrcMessage) LineBytes() ([]byte, error) {
	return ircmsg.line(0, 0, false)
}

// LineMaxLen returns a sendable line created from an IrcMessage, limited by maxlen.
func (ircmsg *IrcMessage) LineMaxLen(maxlenTags, maxlenRest int) (string, error) {
	bytes, err := ircmsg.line(maxlenTags, maxlenRest, true)
	return string(bytes), err
}

// LineMaxLen returns a sendable line created from an IrcMessage, limited by maxlen,
// as a []byte.
func (ircmsg *IrcMessage) LineMaxLenBytes(maxlenTags, maxlenRest int) ([]byte, error) {
	return ircmsg.line(maxlenTags, maxlenRest, true)
}

// line returns a sendable line created from an IrcMessage.
func (ircmsg *IrcMessage) line(maxlenTags, maxlenRest int, useMaxLen bool) ([]byte, error) {
	var buf bytes.Buffer

	if len(ircmsg.Command) < 1 {
		return nil, errors.New("irc: IRC messages MUST have a command")
	}

	if len(ircmsg.Tags) > 0 {
		buf.WriteString("@")

		firstTag := true
		for tag, val := range ircmsg.Tags {
			if !firstTag {
				buf.WriteString(";") // delimiter
			}
			buf.WriteString(tag)
			if val.HasValue {
				buf.WriteString("=")
				buf.WriteString(EscapeTagValue(val.Value))
			}
			firstTag = false
		}

		// truncate if desired
		if useMaxLen && buf.Len() > maxlenTags {
			buf.Truncate(maxlenTags)
		}

		buf.WriteString(" ")
	}

	tagsLen := buf.Len()

	if len(ircmsg.Prefix) > 0 {
		buf.WriteString(":")
		buf.WriteString(ircmsg.Prefix)
		buf.WriteString(" ")
	}

	buf.WriteString(ircmsg.Command)

	if len(ircmsg.Params) > 0 {
		for i, param := range ircmsg.Params {
			buf.WriteString(" ")
			if len(param) < 1 || strings.Contains(param, " ") || param[0] == ':' {
				if i != len(ircmsg.Params)-1 {
					return nil, errors.New("irc: Cannot have an empty param, a param with spaces, or a param that starts with ':' before the last parameter")
				}
				buf.WriteString(":")
			}
			buf.WriteString(param)
		}
	}

	// truncate if desired
	// -2 for \r\n
	restLen := buf.Len() - tagsLen
	if useMaxLen && restLen > maxlenRest-2 {
		buf.Truncate(tagsLen + (maxlenRest - 2))
	}

	buf.WriteString("\r\n")

	return buf.Bytes(), nil
}
