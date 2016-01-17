// written by Daniel Oaks <daniel@danieloaks.net>
// released under the CC0 Public Domain license

package ircmsg

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
