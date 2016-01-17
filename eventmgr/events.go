// written by Daniel Oaks <daniel@danieloaks.net>
// released under the CC0 Public Domain license

package eventmgr

import "sort"

type handlerfn func(map[string]interface{})

// EventHandler holds the priority and handler function of an event.
type EventHandler struct {
	Handler  handlerfn
	Priority int
}

// Handlers holds a list of EventHandlers, including keeping them sorted.
type Handlers struct {
	Handlers []EventHandler
}

// Attach attaches a handler to our internal list.
func (handlers Handlers) Attach(eventhandler EventHandler) {
	handlers.Handlers = append(handlers.Handlers, eventhandler)
	sort.Sort(handlers)
}

// Dispatch dispatches an event to all of our handlers.
func (handlers Handlers) Dispatch(info map[string]interface{}) {
	for _, eventhandler := range handlers.Handlers {
		eventhandler.Handler(info)
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
func (manager *EventManager) Attach(event string, handler handlerfn, priority int) {
	var fullhandler EventHandler
	fullhandler.Handler = handler
	fullhandler.Priority = priority

	_, exists := manager.Events[event]
	if !exists {
		var handlers Handlers
		handlers.Handlers = make([]EventHandler, 2)
		manager.Events[event] = handlers
	}

	manager.Events[event].Attach(fullhandler)
}

// Dispatch dispatches the given event/info to all the matching event handlers.
func (manager *EventManager) Dispatch(event string, info map[string]interface{}) {
	manager.Events[event].Dispatch(info)
}
