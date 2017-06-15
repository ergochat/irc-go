// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package gircclient

import (
	"errors"
	"strconv"

	"github.com/goshuirc/eventmgr"
)

// EventTransforms holds the set of event transformations we apply when
// simplifying given events.
var EventTransforms = map[string]EventTransform{
	"RPL_WELCOME": {
		StringParams: map[int]string{
			1: "message",
		},
	},
}

// EventTransform holds a set of event transformations that should take place
// when simplifying the given event.
type EventTransform struct {
	// StringParams maps the given parameter (int) to the given key in the
	// InfoMap as a string.
	StringParams map[int]string
	// IntParams maps the given parameter (int) to the given key in the InfoMap
	// as an integer.
	IntParams map[int]string
}

// SimplifyEvent simplifies the given event in-place. This includes better
// argument names, convenience attributes, and native objects instead of
// strings where appropriate.
func SimplifyEvent(e eventmgr.InfoMap) error {
	transforms, exists := EventTransforms[e["command"].(string)]

	// no transforms found
	if exists == false {
		return nil
	}

	// apply transformations
	if len(transforms.StringParams) > 0 {
		for i, param := range e["params"].([]string) {
			name, exists := transforms.StringParams[i]
			if exists {
				e[name] = param
			}
		}
	}
	if len(transforms.IntParams) > 0 {
		for i, param := range e["params"].([]string) {
			name, exists := transforms.IntParams[i]
			if exists {
				num, err := strconv.Atoi(param)
				if err == nil {
					e[name] = num
				} else {
					return errors.New("Param " + param + " was not an integer in " + e["command"].(string) + " event")
				}
			}
		}
	}

	// we were successful!
	return nil
}
