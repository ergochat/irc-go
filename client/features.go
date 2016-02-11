// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package gircclient

import (
	"strconv"
	"strings"
)

// ServerFeatures holds a map of server features (RPL_ISUPPORT).
type ServerFeatures map[string]interface{}

// parseFeatureValue changes a raw RPL_ISUPPORT value into a better one.
func parseFeatureValue(name string, value string) interface{} {
	var val interface{}

	if name == "LINELEN" {
		num, err := strconv.Atoi(value)

		if err != nil || num < 0 {
			val = 512
		} else {
			val = num
		}
	} else if name == "NICKLEN" || name == "CHANNELLEN" || name == "TOPICLEN" || name == "USERLEN" {
		num, err := strconv.Atoi(value)

		if err != nil || num < 0 {
			val = nil
		} else {
			val = num
		}
	} else {
		val = value
	}

	return val
}

// Parse the given RPL_ISUPPORT-type tokens and add them to our support list.
func (sf *ServerFeatures) Parse(tokens ...string) {
	for _, token := range tokens {
		if strings.Contains(token, "=") {
			vals := strings.SplitN(token, "=", 2)
			name := strings.ToUpper(vals[0])
			value := vals[1]

			(*sf)[name] = parseFeatureValue(name, value)
		} else {
			(*sf)[strings.ToUpper(token)] = true
		}
	}
}
