// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package gircclient

import "github.com/goshuirc/eventmgr"

// eventRegistration holds events that have not yet been registered.
type eventRegistration struct {
	Direction string
	Name      string
	Handler   eventmgr.HandlerFn
	Priority  int
}

// Reactor is the start-point for gircclient. It creates and manages
// ServerConnections.
type Reactor struct {
	ServerConnections map[string]*ServerConnection
	eventsToRegister  []eventRegistration
}

// NewReactor returns a new, empty Reactor.
func NewReactor() Reactor {
	var newReactor Reactor

	newReactor.ServerConnections = make(map[string]*ServerConnection, 0)
	newReactor.eventsToRegister = make([]eventRegistration, 0)

	// add the default handlers
	newReactor.RegisterEvent("in", "CAP", capHandler, -10)
	newReactor.RegisterEvent("in", "RPL_WELCOME", welcomeHandler, -10)
	newReactor.RegisterEvent("in", "RPL_ISUPPORT", featuresHandler, -10)
	newReactor.RegisterEvent("in", "PING", pingHandler, -10)
	newReactor.RegisterEvent("in", "ERR_NICKNAMEINUSE", nicknameInUseHandler, -10)

	return newReactor
}

// CreateServer creates a ServerConnection and returns it.
func (r *Reactor) CreateServer(name string) *ServerConnection {
	sc := newServerConnection(name)

	r.ServerConnections[name] = sc

	for _, e := range r.eventsToRegister {
		sc.RegisterEvent(e.Direction, e.Name, e.Handler, e.Priority)
	}

	return sc
}

// Shutdown shuts down all ServerConnections.
func (r *Reactor) Shutdown(message string) {
	for _, sc := range r.ServerConnections {
		sc.Shutdown(message)
	}
}

// RegisterEvent registers an event with all current and new ServerConnections.
func (r *Reactor) RegisterEvent(direction string, name string, handler eventmgr.HandlerFn, priority int) {
	for _, sc := range r.ServerConnections {
		sc.RegisterEvent(direction, name, handler, priority)
	}

	// for future servers
	var newEvent eventRegistration
	newEvent.Direction = direction
	newEvent.Name = name
	newEvent.Handler = handler
	newEvent.Priority = priority
	r.eventsToRegister = append(r.eventsToRegister, newEvent)
}
