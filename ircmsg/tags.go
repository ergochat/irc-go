// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircmsg

import "strings"

var (
	// valtoescape replaces real characters with message tag escapes.
	valtoescape = strings.NewReplacer("\\", "\\\\", ";", "\\:", " ", "\\s", "\r", "\\r", "\n", "\\n")

	// escapetoval contains the IRCv3 Tag Escapes and how they map to characters.
	escapetoval = map[rune]byte{
		':':  ';',
		's':  ' ',
		'\\': '\\',
		'r':  '\r',
		'n':  '\n',
	}
)

// EscapeTagValue takes a value, and returns an escaped message tag value.
//
// This function is automatically used when lines are created from an
// IrcMessage, so you don't need to call it yourself before creating a line.
func EscapeTagValue(in string) string {
	return valtoescape.Replace(in)
}

// UnescapeTagValue takes an escaped message tag value, and returns the raw value.
//
// This function is automatically used when lines are interpreted by ParseLine,
// so you don't need to call it yourself after parsing a line.
func UnescapeTagValue(inString string) string {
	in := []rune(inString)
	var out string
	for 0 < len(in) {
		if in[0] == '\\' && len(in) > 1 {
			val, exists := escapetoval[in[1]]
			if exists == true {
				out += string(val)
			} else {
				out += string(in[1])
			}
			in = in[2:]
		} else if in[0] == '\\' {
			// trailing slash
			in = in[1:]
		} else {
			out += string(in[0])
			in = in[1:]
		}
	}

	return out
}

// TagValue represents the value of a tag. This is because tags may have
// no value at all or just an empty value, and this can represent both
// using the HasValue attribute.
type TagValue struct {
	HasValue bool
	Value    string
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

// ParseTags takes a tag string such as "network=freenode;buffer=#chan;joined=1;topic=some\stopic" and outputs a TagValue map.
func ParseTags(tags string) (map[string]TagValue, error) {
	return parseTags(tags, 0, false)
}

// parseTags does the actual tags parsing for the above user-facing function.
func parseTags(tags string, maxlenTags int, useMaxLen bool) (map[string]TagValue, error) {
	tagMap := make(map[string]TagValue)

	// confirm no bad strings exist
	if strings.ContainsAny(tags, " \r\n") {
		return tagMap, ErrorTagsContainsBadChar
	}

	// truncate if desired
	if useMaxLen && len(tags) > maxlenTags {
		tags = tags[:maxlenTags]
	}

	for _, fulltag := range strings.Split(tags, ";") {
		// skip empty tag string values
		if len(fulltag) < 1 {
			continue
		}

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

		tagMap[name] = val
	}

	return tagMap, nil
}
