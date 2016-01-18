// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package eventmgr

import "sort"

// HandlerFn defines a function we can call as a handler.
type HandlerFn func(string, InfoMap)

// InfoMap defines a map we pass through event dispatches.
type InfoMap map[string]interface{}

// NewInfoMap returns a new, empty InfoMap.
func NewInfoMap() InfoMap {
	var info InfoMap
	info = make(InfoMap)
	return info
}

// EventHandler holds the priority and handler function of an event.
type EventHandler struct {
	Handler  HandlerFn
	Priority int
}

// Handlers holds a list of EventHandlers, including keeping them sorted.
type Handlers struct {
	Handlers []EventHandler
}

// Attach attaches a handler to our internal list and returns a new Handlers.
//
// This isn't optimised as much as we probably could. I think most event
// handlers will attach everything at startup, so it's not a massive issue,
// though it is something to look at.
func (handlers Handlers) Attach(eventhandler EventHandler) Handlers {
	if handlers.Handlers == nil {
		handlers.Handlers = make([]EventHandler, 0)
	}
	handlers.Handlers = append(handlers.Handlers, eventhandler)
	sort.Sort(handlers)

	return handlers
}

// Dispatch dispatches an event to all of our handlers.
func (handlers Handlers) Dispatch(event string, info map[string]interface{}) {
	for _, eventhandler := range handlers.Handlers {
		eventhandler.Handler(event, info)
	}
}

// Len returns the length of the HandlerList
func (handlers Handlers) Len() int {
	return len(handlers.Handlers)
}

// Less returns whether i is less than j.
func (handlers Handlers) Less(i, j int) bool {
	return handlers.Handlers[i].Priority < handlers.Handlers[j].Priority
}

// Swap swaps i and j.
func (handlers Handlers) Swap(i, j int) {
	handlers.Handlers[i], handlers.Handlers[j] = handlers.Handlers[j], handlers.Handlers[i]
}

// EventManager lets you attach to and dispatch events.
type EventManager struct {
	Events map[string]Handlers
}

// Attach lets you attach a handler to the given event.
func (manager *EventManager) Attach(event string, handler HandlerFn, priority int) {
	var fullhandler EventHandler
	fullhandler.Handler = handler
	fullhandler.Priority = priority

	if manager.Events == nil {
		manager.Events = make(map[string]Handlers)
	}

	_, exists := manager.Events[event]
	if !exists {
		var handlers Handlers
		manager.Events[event] = handlers
	}

	manager.Events[event] = manager.Events[event].Attach(fullhandler)
}

// Dispatch dispatches the given event/info to all the matching event handlers.
func (manager *EventManager) Dispatch(event string, info map[string]interface{}) {
	events, exists := manager.Events[event]
	if exists {
		events.Dispatch(event, info)
	}
}
