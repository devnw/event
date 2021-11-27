package event

// ErrorStream is a read only stream of errors.
type ErrorStream <-chan error

// Interface returns a read only stream of errors as
// empty interface types instead of errors.
func (e ErrorStream) Interface() <-chan interface{} {
	out := make(chan interface{})

	go func() {
		defer close(out)

		for err := range e {
			out <- err
		}
	}()

	return out
}

// ErrorWriter is a write only stream of errors.
type ErrorWriter chan<- error

// EventStream is a read only stream of events.
// nolint:revive
type EventStream <-chan Event

// Interface returns a read only stream of events as
// empty interface types instead of Event types.
func (e EventStream) Interface() <-chan interface{} {
	out := make(chan interface{})

	go func() {
		defer close(out)

		for event := range e {
			out <- event
		}
	}()

	return out
}

// EventWriter is a write only stream of events.
// nolint:revive
type EventWriter chan<- Event

// EventFunc provides a function signature for event publishing
// which allows for lower memory allocation overhead. This happens
// by allowing the Publisher to only execute the EventFunc in the
// event there is actually a listener for events, rendering events
// only at that time. This is a performance optimization.
// nolint:revive
type EventFunc func() Event

// ErrorFunc provides a function signature for error publishing
// which allows for lower memory allocation overhead. This happens
// by allowing the Publisher to only execute the ErrorFunc in the
// event there is actually a listener for errors, rendering errors
// only at that time. This is a performance optimization.
type ErrorFunc func() error

// Combo is an interface description for types which implement
// both the Event interface and the error interface. Combo is
// treated as an error type.
type Combo interface {
	error
	Event
}

// Event defines an interface which can be implemented by any
// type which is to be published as an event. This interface
// is similar to how the error interface is setup such that the
// event supplies an Event function which returns a string.
type Event interface {
	Event() string
}
