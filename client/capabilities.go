// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package gircclient

import "strings"
import "sort"

// ClientCapabilities holds the capabilities that can and have been enabled on
// a ServerConnection.
type ClientCapabilities struct {
	Avaliable map[string]*string
	Enabled   map[string]bool
	Wanted    []string
}

// NewClientCapabilities returns a newly-initialised ClientCapabilities.
func NewClientCapabilities() ClientCapabilities {
	var cc ClientCapabilities

	cc.Avaliable = make(map[string]*string, 0)
	cc.Enabled = make(map[string]bool, 0)
	cc.Wanted = make([]string, 0)

	return cc
}

// AddWantedCaps adds the given capabilities to our list of capabilities that
// we want from the server.
func (cc *ClientCapabilities) AddWantedCaps(caps ...string) {
	for _, name := range caps {
		// I'm not sure how fast this is, but speed isn't too much of a concern
		// here. Adding 'wanted capabilities' is something that generally only
		// happens at startup anyway.
		i := sort.Search(len(cc.Wanted), func(i int) bool { return cc.Wanted[i] >= name })

		if i >= len(cc.Wanted) || cc.Wanted[i] != name {
			cc.Wanted = append(cc.Wanted, name)
			sort.Strings(cc.Wanted)
		}
	}
}

// AddCaps adds capabilities from LS lists to our Avaliable map.
func (cc *ClientCapabilities) AddCaps(tags ...string) {
	var name string
	var value *string

	for _, tag := range tags {
		if strings.Contains(tag, "=") {
			vals := strings.SplitN(tag, "=", 2)
			name = vals[0]
			value = &vals[1]
		} else {
			name = tag
			value = nil
		}

		cc.Avaliable[name] = value
	}
}

// EnableCaps enables the given capabilities.
func (cc *ClientCapabilities) EnableCaps(caps ...string) {
	for _, name := range caps {
		cc.Enabled[name] = true
	}
}

// DisableCaps disbles the given capabilities.
func (cc *ClientCapabilities) DisableCaps(caps ...string) {
	for _, name := range caps {
		delete(cc.Enabled, name)
	}
}

// DelCaps removes the given capabilities.
func (cc *ClientCapabilities) DelCaps(caps ...string) {
	for _, name := range caps {
		delete(cc.Avaliable, name)
		delete(cc.Enabled, name)
	}
}

// ToRequestLine returns a line of capabilities to request, to be used in a
// CAP REQ line.
func (cc *ClientCapabilities) ToRequestLine() string {
	var caps []string
	caps = make([]string, 0)

	for _, name := range cc.Wanted {
		_, capIsAvailable := cc.Avaliable[name]
		_, capIsEnabled := cc.Enabled[name]

		if capIsAvailable && !capIsEnabled {
			caps = append(caps, name)
		}
	}

	return strings.Join(caps, " ")
}
